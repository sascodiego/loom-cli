// Package cli carries the subcommand implementations of the router binary and
// the contract they share: JSON on stdout, human text on stderr, and the exit
// codes declared by the design.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// Exit codes. Exit 4 (state conflict) is reserved for the v2 reconciliation
// engine and is never returned by v1.
const (
	ExitOK         = 0
	ExitValidation = 2
	ExitProvider   = 3
)

// Env is everything a subcommand needs from its surroundings.
type Env struct {
	// Root is the absolute path of the repository root.
	Root string
	// Stdout carries machine-readable JSON only.
	Stdout io.Writer
	// Stderr carries human-readable text only.
	Stderr io.Writer
	// Stdin backs --input - during an interactive pre-flight.
	Stdin io.Reader
}

// Handler implements one subcommand. It returns nil on success or an error
// that Handle maps onto an exit code.
type Handler func(ctx context.Context, env *Env, args []string) error

var handlers = map[string]Handler{}

// Register binds a subcommand name to its handler. Each subcommand file calls
// it from init so main needs no knowledge of the milestone that added it.
func Register(name string, h Handler) {
	if _, dup := handlers[name]; dup {
		panic("cli: duplicate subcommand " + name)
	}
	handlers[name] = h
}

// Names returns the registered subcommand names in stable order.
func Names() []string {
	names := make([]string, 0, len(handlers))
	for name := range handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ExitError carries a specific exit code through the error chain.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// Fail builds a validation error (exit 2) with a formatted message.
func Fail(format string, a ...any) error {
	return &ExitError{Code: ExitValidation, Err: fmt.Errorf(format, a...)}
}

// FailProvider builds a provider error (exit 3).
func FailProvider(format string, a ...any) error {
	return &ExitError{Code: ExitProvider, Err: fmt.Errorf(format, a...)}
}

// ExitCode maps an error onto the contract: nil is 0, an ExitError keeps its
// code, anything else is a validation failure.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return ExitValidation
}

// Handle dispatches one subcommand invocation.
func Handle(ctx context.Context, env *Env, name string, args []string) error {
	h, ok := handlers[name]
	if !ok {
		return Fail("unknown subcommand %q (known: %s)", name, strings.Join(Names(), ", "))
	}
	return h(ctx, env, args)
}

// Usage line for one subcommand, e.g. "router decide --input <file|->".
const MainUsage = `router — method router for the loom workspace

usage:
  router detect                  report open records for the current branch
  router decide --input <file|-> classify a declared pre-flight (Clef)
  router record --input <file|-> persist the confirmed decision
  router re-route --routing-id <id> --method <method> [--justification <text>] [--requested-by agent|user]
                                 supersede an open record with a new method
  router close --routing-id <id>  mark an open record closed
  router metrics [--json]        agreement rate over recorded decisions

output: JSON on stdout, human text on stderr.
exit:   0 success, 2 usage or validation error, 3 provider failure after retry.
`

// NewFlagSet returns a flag set that reports errors through the returned error
// instead of exiting, keeping the exit-code contract in one place.
func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// ParseFlags parses args and maps any failure onto a validation error.
func ParseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return Fail("%s: %v", fs.Name(), err)
	}
	if fs.NArg() > 0 {
		return Fail("%s: unexpected argument %q", fs.Name(), fs.Arg(0))
	}
	return nil
}

// RepoRoot resolves the repository root from dir using git itself, so linked
// worktrees resolve to their own root rather than the main checkout.
func RepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", Fail("not inside a git repository (%s)", dir)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", Fail("git returned an empty repository root for %s", dir)
	}
	return root, nil
}

// CurrentBranch resolves the checked-out branch of the repository at root.
// A detached HEAD is a validation error: routing state is keyed by branch.
func CurrentBranch(root string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", Fail("detached HEAD: routing state requires a branch")
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", Fail("could not resolve the current branch")
	}
	return branch, nil
}

// ReadInput reads a --input payload from a file path or from stdin when the
// value is "-".
func ReadInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "" {
		return nil, Fail("missing --input <file|->")
	}
	if path == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, Fail("read stdin: %v", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, Fail("read %s: %v", path, err)
	}
	return data, nil
}

// WriteJSON prints one JSON document followed by a newline. It is the only
// stdout path in the package, keeping stdout machine-readable.
func WriteJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Fail("encode output: %v", err)
	}
	if _, err := fmt.Fprintf(w, "%s\n", data); err != nil {
		return Fail("write output: %v", err)
	}
	return nil
}
