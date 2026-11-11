// Command router is the method router for the loom workspace: it detects open
// routing records for the current branch, classifies a declared pre-flight
// through Clef, and records the decision the user confirmed.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"loom/router/internal/cli"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole binary minus os.Exit, so the CLI-layer exit-code contract is
// testable without spawning a process.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, cli.MainUsage)
		return cli.ExitValidation
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, _ = fmt.Fprint(stdout, cli.MainUsage)
		return cli.ExitOK
	}

	root, err := cli.RepoRoot(".")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "router: %v\n", err)
		return cli.ExitCode(err)
	}

	env := &cli.Env{Root: root, Stdout: stdout, Stderr: stderr, Stdin: os.Stdin}
	if err := cli.Handle(context.Background(), env, args[0], args[1:]); err != nil {
		_, _ = fmt.Fprintf(stderr, "router %s: %v\n", args[0], err)
		return cli.ExitCode(err)
	}
	return cli.ExitOK
}
