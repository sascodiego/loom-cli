package cli

import (
	"context"

	"loom/router/internal/state"
)

func init() { Register("close", runClose) }

// runClose marks an open record closed at ship time and appends the closing
// event. It never touches the working tree.
func runClose(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("close")
	routingID := fs.String("routing-id", "", "routing id of the record to close")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}
	if *routingID == "" {
		return Fail("close: missing --routing-id")
	}
	rec, _, err := state.FindRecord(env.Root, *routingID)
	if err != nil {
		return Fail("close: %v", err)
	}
	switch rec.Status {
	case state.StatusClosed:
		return Fail("close: record %s is already closed", rec.RoutingID)
	case state.StatusSuperseded:
		return Fail("close: record %s was superseded by %s and cannot be closed", rec.RoutingID, rec.SupersededBy)
	}
	rec.Status = state.StatusClosed
	if err := state.WriteRecord(env.Root, rec); err != nil {
		return Fail("close: write record: %v", err)
	}
	if err := state.Append(env.Root, state.Event{
		Event:         state.EventClosed,
		DecidedBy:     rec.DecidedBy,
		HumanApproved: rec.HumanApproved,
		UserChoice:    rec.UserChoice,
		Record:        rec,
	}); err != nil {
		return Fail("close: append history: %v", err)
	}
	return WriteJSON(env.Stdout, rec)
}
