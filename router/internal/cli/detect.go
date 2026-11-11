package cli

import (
	"context"

	"loom/router/internal/state"
)

func init() { Register("detect", runDetect) }

// runDetect prints the open records for the current branch. It never calls the
// network and never reads git history: state on disk is the only authority.
func runDetect(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("detect")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}
	branch, err := CurrentBranch(env.Root)
	if err != nil {
		return err
	}
	res, err := state.Detect(env.Root, branch)
	if err != nil {
		return Fail("detect: %v", err)
	}
	return WriteJSON(env.Stdout, res)
}
