package cli

import (
	"context"
	"fmt"
	"time"

	"loom/router/internal/registry"
	"loom/router/internal/state"
)

func init() { Register("re-route", runReRoute) }

// recordWriter is the narrow slice of the state package the supersede sequence
// needs. Narrowing it keeps the rollback path testable without a filesystem.
type recordWriter interface {
	Write(*state.Record) error
	Delete(*state.Record) error
}

// fsRecordStore adapts the state package's file-backed writers.
type fsRecordStore struct{ root string }

func (s fsRecordStore) Write(r *state.Record) error  { return state.WriteRecord(s.root, r) }
func (s fsRecordStore) Delete(r *state.Record) error { return state.DeleteRecord(s.root, r) }

// supersedeRecords writes the successor first and the superseded predecessor
// second.
//
// The order matters: a new file cannot destroy anything, so a failure there
// leaves the branch untouched. A failure on the predecessor's write would
// otherwise leave a record marked superseded pointing at a successor that never
// landed, so the successor is rolled back and the branch keeps exactly one open
// record — recoverable, and true.
func supersedeRecords(w recordWriter, old, fresh *state.Record) error {
	if err := w.Write(fresh); err != nil {
		return fmt.Errorf("write the successor record: %w (nothing changed)", err)
	}
	old.Status = state.StatusSuperseded
	old.SupersededBy = fresh.RoutingID
	if err := w.Write(old); err != nil {
		if derr := w.Delete(fresh); derr != nil {
			return fmt.Errorf("write the superseded record: %w (rollback of successor %s also failed: %v)",
				err, fresh.RoutingID, derr)
		}
		return fmt.Errorf("write the superseded record: %w (successor %s rolled back, record left open)",
			err, fresh.RoutingID)
	}
	return nil
}

// runReRoute supersedes an open record with a new one under the target method.
// It re-classifies the re-declared pre-flight for real (S2): the classification
// runs before anything is written, so a failure there leaves the predecessor
// untouched. The recorded method stays the user-approved --method; the call
// supplies provenance and evidence, not authority. It never touches the
// working tree: the discard of in-flight artifacts is the user's and their
// orchestration agent's job — the CLI only records the supersede event and
// emits the fresh decision.
func runReRoute(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("re-route")
	routingID := fs.String("routing-id", "", "routing id of the open record to supersede")
	method := fs.String("method", "", "target method of the re-route")
	justification := fs.String("justification", "", "why the change is being re-routed")
	requestedBy := fs.String("requested-by", "agent", "who requested the re-route: agent or user")
	input := fs.String("input", "", "path to the re-declared pre-flight JSON, or - for stdin")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}

	switch {
	case *routingID == "":
		return Fail("missing --routing-id")
	case *method == "":
		return Fail("missing --method")
	case *input == "":
		return Fail("missing --input <file|->")
	case *requestedBy != "agent" && *requestedBy != "user":
		return Fail("invalid --requested-by %q (want agent or user)", *requestedBy)
	}

	old, _, err := state.FindRecord(env.Root, *routingID)
	if err != nil {
		return Fail("%v", err)
	}
	switch old.Status {
	case state.StatusClosed:
		return Fail("record %s is closed and cannot be superseded", old.RoutingID)
	case state.StatusSuperseded:
		return Fail("record %s was already superseded by %s", old.RoutingID, old.SupersededBy)
	}

	reg, err := registry.Load(env.Root)
	if err != nil {
		return Fail("%v", err)
	}
	target, known := reg.Method(*method)
	switch {
	case !known:
		return Fail("unknown method %q", *method)
	case !target.Available:
		return Fail("method %q is unavailable (excluded from v1 dispatch)", *method)
	}

	// The classification runs before anything is persisted: a bad declaration,
	// an unknown subsystem, a missing config or a provider failure leaves the
	// predecessor exactly as it stands, with no history appended.
	data, err := ReadInput(*input, env.Stdin)
	if err != nil {
		return err
	}
	c, err := classify(ctx, env, data)
	if err != nil {
		return err
	}

	newID, err := state.NewRoutingID()
	if err != nil {
		return Fail("mint routing_id: %v", err)
	}

	// Snapshot of the old record as it stands before the supersede, for the
	// re-route request event.
	before := *old

	// The fresh record carries the identity the predecessor owned (slug,
	// branch) and fresh classification provenance for everything else: nothing
	// is inherited from the old decision.
	fresh := state.Record{
		SchemaVersion:   old.SchemaVersion,
		RoutingID:       newID,
		Slug:            old.Slug,
		Branch:          old.Branch,
		BranchEncoded:   old.BranchEncoded,
		Method:          *method,
		Tier:            c.Tier,
		DecidedAt:       time.Now().UTC(),
		RequestID:       c.RequestID,
		Provider:        c.Provider,
		Model:           c.Model,
		CriteriaHash:    c.CriteriaHash,
		DecidedBy:       state.DecidedByOverrideReRoute,
		HumanApproved:   false,
		SuggestedMethod: c.SuggestedMethod,
		SuggestedTier:   c.Tier,
		UserChoice:      *method,
		PayloadHash:     c.PayloadHash,
		Status:          state.StatusOpen,
		RequestedBy:     *requestedBy,
	}

	if err := supersedeRecords(fsRecordStore{root: env.Root}, old, &fresh); err != nil {
		return Fail("%v", err)
	}

	// Audit trail: the request on the old routing id, the supersede that
	// closed it, and the fresh decision under its own id.
	events := []state.Event{
		{
			Event:         state.EventReRouteRequested,
			DecidedBy:     before.DecidedBy,
			HumanApproved: before.HumanApproved,
			UserChoice:    *method,
			RequestedBy:   *requestedBy,
			Justification: *justification,
			Record:        &before,
		},
		{
			Event:         state.EventSuperseded,
			DecidedBy:     old.DecidedBy,
			HumanApproved: old.HumanApproved,
			UserChoice:    old.UserChoice,
			RequestedBy:   old.RequestedBy,
			Record:        old,
		},
		{
			Event:         state.EventReRouteRequested,
			DecidedBy:     fresh.DecidedBy,
			HumanApproved: fresh.HumanApproved,
			UserChoice:    fresh.UserChoice,
			RequestedBy:   fresh.RequestedBy,
			Justification: *justification,
			Record:        &fresh,
		},
	}
	for _, ev := range events {
		if err := state.Append(env.Root, ev); err != nil {
			return Fail("append history: %v", err)
		}
	}

	_, _ = fmt.Fprintf(env.Stderr, "router re-route: %s superseded by %s (method %s, requested by %s)\n",
		old.RoutingID, newID, *method, *requestedBy)
	return WriteJSON(env.Stdout, &fresh)
}
