package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Cloudflare talks to the Workers AI REST endpoint that serves the Clef
// decision model.
//
// Request shape (the adapter's binding assumption, pinned by the golden test):
//
//	POST {Endpoint}
//	Authorization: Bearer {Token}
//	{"model":"@cf/cloudflare/clef",
//	 "input":{"state":<pre-flight payload>,"questions":[<typed questions>]}}
//
// Response shape:
//
//	{"success":true,
//	 "result":{"request_id":"<trace>","answers":{"<question>":{"<option>":<p>}}}}
type Cloudflare struct {
	// Endpoint is the full URL, including account and model path when the
	// provider needs it.
	Endpoint string
	// Model is the model id sent in the body.
	Model string
	// Token is the API key. It is read from the environment by the caller and
	// never persisted.
	Token string
	// HTTPClient is injectable so tests control timeouts.
	HTTPClient *http.Client
	// MaxAttempts is the total number of calls, retries included. Zero means 3.
	MaxAttempts int
	// Backoff is the base delay for exponential retry. Zero means 250ms.
	Backoff time.Duration
}

type cfRequest struct {
	Model string         `json:"model"`
	Input cfRequestInput `json:"input"`
}

type cfRequestInput struct {
	State     json.RawMessage `json:"state"`
	Questions []Question      `json:"questions"`
}

type cfResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	Result struct {
		RequestID string                        `json:"request_id"`
		Answers   map[string]map[string]float64 `json:"answers"`
	} `json:"result"`
}

func (c *Cloudflare) maxAttempts() int {
	if c.MaxAttempts <= 0 {
		return 3
	}
	return c.MaxAttempts
}

func (c *Cloudflare) backoff(attempt int) time.Duration {
	base := c.Backoff
	if base <= 0 {
		base = 250 * time.Millisecond
	}
	d := base
	for i := 1; i < attempt; i++ {
		d *= 2
	}
	return d
}

// Decide sends one classification request, retrying only transient failures:
// three attempts total, exponential backoff between them.
func (c *Cloudflare) Decide(ctx context.Context, state json.RawMessage, questions []Question) (*Decision, error) {
	if c.Endpoint == "" {
		return nil, &Failure{Kind: FailureTransport, Err: errors.New("no endpoint configured")}
	}
	if _, err := url.Parse(c.Endpoint); err != nil {
		return nil, &Failure{Kind: FailureTransport, Err: fmt.Errorf("invalid endpoint: %w", err)}
	}

	body, err := json.Marshal(cfRequest{
		Model: c.Model,
		Input: cfRequestInput{State: state, Questions: questions},
	})
	if err != nil {
		return nil, &Failure{Kind: FailureMalformed, Err: fmt.Errorf("encode request: %w", err)}
	}

	attempts := 0
	var last *Failure
	for attempts < c.maxAttempts() {
		attempts++
		dec, failure, retryable := c.attempt(ctx, body, attempts)
		if failure == nil {
			return dec, nil
		}
		last = failure
		if !retryable {
			return nil, failure
		}
		if attempts < c.maxAttempts() {
			select {
			case <-ctx.Done():
				return nil, &Failure{Kind: FailureTimeout, Attempts: attempts, Err: ctx.Err()}
			case <-time.After(c.backoff(attempts)):
			}
		}
	}
	return nil, last
}

// attempt performs exactly one call and classifies its outcome.
func (c *Cloudflare) attempt(ctx context.Context, body []byte, attempt int) (*Decision, *Failure, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, &Failure{Kind: FailureTransport, Attempts: attempt, Err: err}, false
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, classifyTransport(err, attempt), isTransient(err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &Failure{Kind: FailureAuth, Status: resp.StatusCode, Attempts: attempt}, false
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, &Failure{Kind: FailureQuota, Status: resp.StatusCode, Attempts: attempt}, true
	case resp.StatusCode >= 500:
		return nil, &Failure{Kind: FailureServer, Status: resp.StatusCode, Attempts: attempt}, true
	case resp.StatusCode >= 400:
		return nil, &Failure{Kind: FailureMalformed, Status: resp.StatusCode, Attempts: attempt}, false
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Failure{Kind: FailureTimeout, Status: resp.StatusCode, Attempts: attempt, Err: err}, true
	}
	var parsed cfResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, &Failure{Kind: FailureMalformed, Status: resp.StatusCode, Attempts: attempt, Err: err}, false
	}
	if !parsed.Success {
		msg := "endpoint reported failure"
		if len(parsed.Errors) > 0 {
			msg = parsed.Errors[0].Message
		}
		return nil, &Failure{Kind: FailureMalformed, Status: resp.StatusCode, Attempts: attempt,
			Err: errors.New(msg)}, false
	}
	if len(parsed.Result.Answers) == 0 {
		return nil, &Failure{Kind: FailureMalformed, Status: resp.StatusCode, Attempts: attempt,
			Err: errors.New("envelope carried no answers")}, false
	}
	return &Decision{
		RequestID:     parsed.Result.RequestID,
		Probabilities: parsed.Result.Answers,
		Attempts:      attempt,
	}, nil, false
}

// classifyTransport turns a client-side error into a typed failure.
func classifyTransport(err error, attempt int) *Failure {
	kind := FailureTransport
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &netErr) && netErr.Timeout()) ||
		strings.Contains(err.Error(), "context deadline exceeded") {
		kind = FailureTimeout
	}
	return &Failure{Kind: kind, Attempts: attempt, Err: err}
}

// isTransient reports whether the transport error could succeed on a retry.
func isTransient(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// A connection reset or refused may clear; a malformed URL will not.
	return !errors.Is(err, context.Canceled)
}
