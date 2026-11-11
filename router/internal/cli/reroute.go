package cli

import (
	"context"
	"fmt"
	"time"

	"loom/router/internal/registry"
	"loom/router/internal/state"
)

func init() { Register("re-route", runReRoute) }

// runReRoute supersedes an open record with a new one under the target method.
// It never touches the working tree: the discard of in-flight artifacts is the
// user's and their orchestration agent's job — the CLI only records the
// supersede event and emits the fresh decision.
func runReRoute(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("re-route")
	routingID := fs.String("routing-id", "", "routing id of the open record to supersede")
	method := fs.String("method", "", "target method of the re-route")
	justification := fs.String("justification", "", "why the change is being re-routed")
	requestedBy := fs.String("requested-by", "agent", "who requested the re-route: agent or user")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}

	switch {
	case *routingID == "":
		return Fail("missing --routing-id")
	case *method == "":
		return Fail("missing --method")
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

	newID, err := state.NewRoutingID()
	if err != nil {
		return Fail("mint routing_id: %v", err)
	}

	// Snapshot of the old record as it stands before the supersede, for the
	// re-route request event.
	before := *old

	fresh := *old
	fresh.RoutingID = newID
	fresh.Method = *method
	fresh.DecidedBy = state.DecidedByOverrideReRoute
	fresh.HumanApproved = false
	fresh.UserChoice = *method
	fresh.RequestedBy = *requestedBy
	fresh.DecidedAt = time.Now().UTC()
	fresh.Status = state.StatusOpen
	fresh.SupersededBy = ""

	old.Status = state.StatusSuperseded
	old.SupersededBy = newID
	if err := state.WriteRecord(env.Root, old); err != nil {
		return Fail("write superseded record: %v", err)
	}
	if err := state.WriteRecord(env.Root, &fresh); err != nil {
		return Fail("write new record: %v", err)
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
