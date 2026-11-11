// Package provider is the single place that knows the classification endpoint:
// what a typed question looks like, how the request is shaped, how failures are
// typed, and how retries are bounded. Swapping classifiers means implementing
// Provider; nothing else in the design changes.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
)

// Option is one allowed answer, with the description the endpoint sees. For the
// method question the description is the method's CRITERIA.md verbatim.
type Option struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// Question is one typed question: a key and its closed option set. The endpoint
// returns a calibrated probability for every option in one pass.
type Question struct {
	Key     string   `json:"key"`
	Options []Option `json:"options"`
}

// Decision is the classifier's answer.
type Decision struct {
	// RequestID is the endpoint's own request or trace id, recorded so a
	// decision can be traced back to the call that produced it.
	RequestID string `json:"request_id"`
	// Probabilities maps question key to option value to probability.
	Probabilities map[string]map[string]float64 `json:"probabilities"`
	// Attempts is how many calls the answer took, evidence for the retry path.
	Attempts int `json:"attempts"`
}

// Provider is the one interface the router depends on.
type Provider interface {
	Decide(ctx context.Context, state json.RawMessage, questions []Question) (*Decision, error)
}

// Failure kinds. The joint-fallback conversation is triggered by any of these:
// the user must see why the classifier could not answer.
const (
	FailureAuth      = "auth"               // 401/403: credentials are wrong; retrying cannot help
	FailureQuota     = "quota"              // 429 after retries
	FailureServer    = "server"             // 5xx after retries
	FailureTimeout   = "timeout"            // the call exceeded its deadline
	FailureTransport = "transport"          // the endpoint could not be reached
	FailureMalformed = "malformed_response" // the answer could not be interpreted
)

// Failure is the typed envelope the joint-fallback path consumes.
type Failure struct {
	Kind     string
	Status   int
	Attempts int
	Err      error
}

func (f *Failure) Error() string {
	parts := []string{f.Kind}
	if f.Status != 0 {
		parts = append(parts, fmt.Sprintf("status %d", f.Status))
	}
	if f.Attempts > 0 {
		parts = append(parts, fmt.Sprintf("after %d attempt(s)", f.Attempts))
	}
	msg := "provider failure: " + join(parts)
	if f.Err != nil {
		msg += ": " + f.Err.Error()
	}
	return msg
}

func (f *Failure) Unwrap() error { return f.Err }

func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
