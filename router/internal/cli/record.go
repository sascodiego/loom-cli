package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"loom/router/internal/state"
)

func init() { Register("record", runRecord) }

// RecordInput is the confirmed decision `router record` persists: the fields
// `decide` produced plus the outcome of the confirmation gate. The branch is
// never taken from input — it is read from git so the caller cannot route a
// record onto a branch it is not on.
type RecordInput struct {
	Slug            string          `json:"slug"`
	Method          string          `json:"method"`
	Tier            string          `json:"tier"`
	RequestID       string          `json:"request_id"`
	Provider        string          `json:"provider"`
	Model           string          `json:"model"`
	CriteriaHash    string          `json:"criteria_hash"`
	DecidedBy       state.DecidedBy `json:"decided_by"`
	HumanApproved   bool            `json:"human_approved"`
	SuggestedMethod string          `json:"suggested_method"`
	SuggestedTier   string          `json:"suggested_tier"`
	UserChoice      string          `json:"user_choice,omitempty"`
	PayloadHash     string          `json:"payload_hash"`
}

// Validate enforces the gate semantics the schema declares: clef means the user
// accepted the suggestion, an override means they picked something else, and
// override_re_route belongs to `router re-route`, never to a first record.
func (in *RecordInput) Validate() error {
	switch {
	case in.Slug == "":
		return fmt.Errorf("missing slug")
	case in.Method == "":
		return fmt.Errorf("missing method")
	case in.Tier == "":
		return fmt.Errorf("missing tier")
	case in.RequestID == "":
		return fmt.Errorf("missing request_id (run `router decide` first)")
	case in.Provider == "":
		return fmt.Errorf("missing provider")
	case in.Model == "":
		return fmt.Errorf("missing model")
	case in.CriteriaHash == "":
		return fmt.Errorf("missing criteria_hash")
	case in.PayloadHash == "":
		return fmt.Errorf("missing payload_hash")
	case !in.DecidedBy.Valid():
		return fmt.Errorf("invalid decided_by %q", in.DecidedBy)
	case in.DecidedBy == state.DecidedByOverrideReRoute:
		return fmt.Errorf("decided_by override_re_route belongs to `router re-route`, not `router record`")
	}

	switch in.DecidedBy {
	case state.DecidedByClef:
		if !in.HumanApproved {
			return fmt.Errorf("decided_by clef requires human_approved: true (an unapproved suggestion is not a decision)")
		}
		if in.SuggestedMethod != "" && in.Method != in.SuggestedMethod {
			return fmt.Errorf("decided_by clef with method %q != suggested_method %q is an override; use user_gate_override", in.Method, in.SuggestedMethod)
		}
	case state.DecidedByUserGateOverride:
		if in.HumanApproved {
			return fmt.Errorf("decided_by user_gate_override requires human_approved: false")
		}
		if in.SuggestedMethod != "" && in.Method == in.SuggestedMethod {
			return fmt.Errorf("user_gate_override with method equal to suggested_method is not an override")
		}
		if in.UserChoice == "" {
			return fmt.Errorf("user_gate_override requires user_choice")
		}
	case state.DecidedByJointFallback:
		if in.HumanApproved {
			return fmt.Errorf("decided_by joint_fallback requires human_approved: false (no clef suggestion was approved)")
		}
	}
	return nil
}

// runRecord persists the confirmed decision and opens the routing record.
func runRecord(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("record")
	input := fs.String("input", "", "path to the confirmed decision JSON, or - for stdin")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}
	data, err := ReadInput(*input, env.Stdin)
	if err != nil {
		return err
	}
	var in RecordInput
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return Fail("decode input: %v", err)
	}
	if err := in.Validate(); err != nil {
		return Fail("invalid record input: %v", err)
	}

	branch, err := CurrentBranch(env.Root)
	if err != nil {
		return err
	}

	// Warn, never block: recording while records are open is legitimate (a
	// branch can hold several requests), but the user must see it.
	if existing, derr := state.Detect(env.Root, branch); derr == nil && len(existing.Candidates) > 0 {
		_, _ = fmt.Fprintf(env.Stderr,
			"router record: note — %d open record(s) already on branch %q; the new record gets its own routing_id\n",
			len(existing.Candidates), branch)
	}

	routingID, err := state.NewRoutingID()
	if err != nil {
		return Fail("mint routing_id: %v", err)
	}
	rec := &state.Record{
		SchemaVersion:   state.SchemaVersion,
		RoutingID:       routingID,
		Slug:            in.Slug,
		Branch:          branch,
		BranchEncoded:   state.EncodeBranch(branch),
		Method:          in.Method,
		Tier:            in.Tier,
		DecidedAt:       time.Now().UTC(),
		RequestID:       in.RequestID,
		Provider:        in.Provider,
		Model:           in.Model,
		CriteriaHash:    in.CriteriaHash,
		DecidedBy:       in.DecidedBy,
		HumanApproved:   in.HumanApproved,
		SuggestedMethod: in.SuggestedMethod,
		SuggestedTier:   in.SuggestedTier,
		UserChoice:      in.UserChoice,
		PayloadHash:     in.PayloadHash,
		Status:          state.StatusOpen,
	}
	if err := state.WriteRecord(env.Root, rec); err != nil {
		return Fail("write record: %v", err)
	}
	ev := state.Event{
		Event:         eventFor(in.DecidedBy),
		DecidedBy:     in.DecidedBy,
		HumanApproved: in.HumanApproved,
		UserChoice:    in.UserChoice,
		Record:        rec,
	}
	if err := state.Append(env.Root, ev); err != nil {
		return Fail("append history: %v", err)
	}
	return WriteJSON(env.Stdout, rec)
}

// eventFor maps a decision origin onto its first history event.
func eventFor(d state.DecidedBy) string {
	switch d {
	case state.DecidedByUserGateOverride:
		return state.EventGateOverridden
	case state.DecidedByJointFallback:
		return state.EventJointFallback
	default:
		return state.EventDecided
	}
}
