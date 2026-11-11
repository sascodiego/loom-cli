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
