package main

import (
	"bytes"
	"strings"
	"testing"

	"loom/router/internal/cli"
)

// The CLI-layer contract: JSON-or-nothing on stdout, human text on stderr, and
// the declared exit codes. Extended in M4 with the provider and state paths.

func runCapture(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestNoArgsIsUsageError(t *testing.T) {
	code, stdout, stderr := runCapture(t)
	if code != cli.ExitValidation {
		t.Fatalf("exit = %d, want %d", code, cli.ExitValidation)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Fatalf("stderr = %q, want usage text", stderr)
	}
}

func TestHelpSucceeds(t *testing.T) {
	code, stdout, stderr := runCapture(t, "--help")
	if code != cli.ExitOK {
		t.Fatalf("exit = %d, want %d", code, cli.ExitOK)
	}
	if !strings.Contains(stdout, "router decide") {
		t.Fatalf("stdout = %q, want subcommand list", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestUnknownSubcommandIsValidationError(t *testing.T) {
	code, stdout, stderr := runCapture(t, "frobnicate")
	if code != cli.ExitValidation {
		t.Fatalf("exit = %d, want %d", code, cli.ExitValidation)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unknown subcommand") {
		t.Fatalf("stderr = %q, want unknown-subcommand message", stderr)
	}
}

// A real subcommand's validation failure must reach exit 2 through run(), so
// the per-subcommand mapping in the design (close emits 2 on validation,
// re-route emits 2 on closed or unknown records) holds at the binary layer.
func TestSubcommandValidationFailureIsExitTwo(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"close without routing id", []string{"close"}, "missing --routing-id"},
		{"re-route without routing id", []string{"re-route", "--method", "odd"}, "missing --routing-id"},
		{"re-route without method", []string{"re-route", "--routing-id", "x"}, "missing --method"},
		{"re-route without input", []string{"re-route", "--routing-id", "x", "--method", "odd"}, "missing --input"},
		{"metrics with a positional argument", []string{"metrics", "extra"}, "unexpected argument"},
		{"re-route with a bad requester", []string{"re-route", "--routing-id", "x", "--method", "odd", "--input", "-", "--requested-by", "robot"}, "requested-by"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCapture(t, tt.args...)
			if code != cli.ExitValidation {
				t.Fatalf("exit = %d, want %d", code, cli.ExitValidation)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, tt.want) {
				t.Fatalf("stderr = %q, want %q", stderr, tt.want)
			}
		})
	}
}
