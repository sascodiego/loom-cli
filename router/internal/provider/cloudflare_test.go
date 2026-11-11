package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A minimal questions fixture: two typed questions, the shape decide sends.
func testQuestions() []Question {
	return []Question{
		{Key: "effort_tier", Options: []Option{
			{Value: "low", Description: "one bounded change"},
			{Value: "medium", Description: "several steps, one session"},
			{Value: "high", Description: "several subsystems or sessions"},
		}},
		{Key: "method", Options: []Option{
			{Value: "odd", Description: "the floor"},
			{Value: "ftd", Description: "fast track"},
		}},
	}
}

func successBody() string {
	return `{"success":true,"result":{"request_id":"trace-1","answers":{
		"effort_tier":{"low":0.10,"medium":0.72,"high":0.18},
		"method":{"odd":0.81,"ftd":0.19}}}}`
}

func newClient(t *testing.T, handler http.HandlerFunc) (*Cloudflare, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Cloudflare{
		Endpoint:    srv.URL,
		Model:       "@cf/cloudflare/clef",
		Token:       "test-token",
		HTTPClient:  &http.Client{Timeout: 2 * time.Second},
		MaxAttempts: 3,
		Backoff:     time.Millisecond,
	}, srv
}

func TestDecideSuccess(t *testing.T) {
	var hits int32
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(successBody()))
	})

	d, err := c.Decide(context.Background(), json.RawMessage(`{"schema_version":1}`), testQuestions())
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if d.RequestID != "trace-1" {
		t.Fatalf("request id = %q", d.RequestID)
	}
	if got := d.Probabilities["method"]["odd"]; got != 0.81 {
		t.Fatalf("method.odd = %v, want 0.81", got)
	}
	if d.Attempts != 1 {
		t.Fatalf("attempts = %d, want 1", d.Attempts)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hits = %d, want 1", hits)
	}
}

func TestDecideRetriesTransientThenSucceeds(t *testing.T) {
	var hits int32
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(successBody()))
	})

	d, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if d.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3", d.Attempts)
	}
	if atomic.LoadInt32(&hits) != 3 {
		t.Fatalf("server hits = %d, want 3", hits)
	}
}

func TestDecideExhaustsThreeAttempts(t *testing.T) {
	var hits int32
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	if err == nil {
		t.Fatal("Decide succeeded against a permanently failing server")
	}
	var f *Failure
	if !errors.As(err, &f) {
		t.Fatalf("error is not a *Failure: %T %v", err, err)
	}
	if f.Kind != FailureServer {
		t.Fatalf("kind = %q, want %q", f.Kind, FailureServer)
	}
	if f.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3 (2 retries)", f.Attempts)
	}
	if atomic.LoadInt32(&hits) != 3 {
		t.Fatalf("server hits = %d, want 3", hits)
	}
}

func TestDecideQuotaFailure(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	var f *Failure
	if !errors.As(err, &f) || f.Kind != FailureQuota {
		t.Fatalf("err = %v, want a quota failure", err)
	}
}

func TestDecideAuthFailureIsNotRetried(t *testing.T) {
	var hits int32
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusUnauthorized)
	})
	_, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	var f *Failure
	if !errors.As(err, &f) || f.Kind != FailureAuth {
		t.Fatalf("err = %v, want an auth failure", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hits = %d, want 1: an auth failure must not be retried", hits)
	}
}

func TestDecideMalformedResponseIsNotRetried(t *testing.T) {
	var hits int32
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"success":true,"result":`))
	})
	_, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	var f *Failure
	if !errors.As(err, &f) || f.Kind != FailureMalformed {
		t.Fatalf("err = %v, want a malformed-response failure", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("server hits = %d, want 1", hits)
	}
}

func TestDecideTimeoutIsTyped(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		_, _ = w.Write([]byte(successBody()))
	})
	c.HTTPClient = &http.Client{Timeout: 10 * time.Millisecond}
	c.MaxAttempts = 2

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := c.Decide(ctx, json.RawMessage(`{}`), testQuestions())
	var f *Failure
	if !errors.As(err, &f) || f.Kind != FailureTimeout {
		t.Fatalf("err = %v, want a timeout failure", err)
	}
}

func TestDecideRejectsAnEnvelopeWithoutAnswers(t *testing.T) {
	c, _ := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"result":{"request_id":"x"}}`))
	})
	_, err := c.Decide(context.Background(), json.RawMessage(`{}`), testQuestions())
	var f *Failure
	if !errors.As(err, &f) || f.Kind != FailureMalformed {
		t.Fatalf("err = %v, want malformed (no answers)", err)
	}
}

func TestFailureCarriesTheJointFallbackEvidence(t *testing.T) {
	f := &Failure{Kind: FailureQuota, Status: 429, Attempts: 3}
	msg := f.Error()
	for _, want := range []string{"quota", "429", "3"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Failure.Error() = %q, want it to mention %q", msg, want)
		}
	}
}
