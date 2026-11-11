package preflight

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"loom/router/internal/provider"
)

func fixtureCriteria() []provider.Criterion {
	return []provider.Criterion{
		{Method: "odd", Content: "odd criteria"},
		{Method: "ftd", Content: "ftd criteria"},
	}
}

func TestDeriveBuildsStateQuestionsAndPayloadHash(t *testing.T) {
	raw := json.RawMessage(validDeclaration)
	decl, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	p, err := Derive(DeriveInput{
		Raw:         raw,
		Declaration: decl,
		Facts:       provider.SessionFacts{Branch: "feat/x", ModifiedFiles: 2},
		Criteria:    fixtureCriteria(),
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	wantState, err := provider.BuildState(raw, provider.SessionFacts{Branch: "feat/x", ModifiedFiles: 2}, fixtureCriteria())
	if err != nil {
		t.Fatalf("BuildState: %v", err)
	}
	if string(p.State) != string(wantState) {
		t.Fatalf("state = %s\nwant    %s", p.State, wantState)
	}
	if len(p.Questions) != 2 || p.Questions[1].Key != "method" {
		t.Fatalf("questions = %+v, want effort_tier and method", p.Questions)
	}

	sum := sha256.Sum256([]byte(validDeclaration))
	if p.PayloadHash != hex.EncodeToString(sum[:]) {
		t.Fatalf("payload hash = %q, want the SHA-256 of the exact declaration bytes", p.PayloadHash)
	}
}

func TestDeriveRejectsAMissingDeclaration(t *testing.T) {
	if _, err := Derive(DeriveInput{Facts: provider.SessionFacts{Branch: "x"}}); err == nil ||
		!strings.Contains(err.Error(), "declaration") {
		t.Fatalf("Derive = %v, want a declaration error", err)
	}
	decl, _ := Parse([]byte(validDeclaration))
	if _, err := Derive(DeriveInput{Declaration: decl, Facts: provider.SessionFacts{Branch: "x"}}); err == nil ||
		!strings.Contains(err.Error(), "pre-flight") {
		t.Fatalf("Derive = %v, want a missing-bytes error", err)
	}
}

func TestModifiedFileCount(t *testing.T) {
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", "-A")
	run("commit", "-q", "-m", "init")

	n, err := ModifiedFileCount(root)
	if err != nil {
		t.Fatalf("ModifiedFileCount: %v", err)
	}
	if n != 0 {
		t.Fatalf("clean tree = %d, want 0", n)
	}

	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	n, err = ModifiedFileCount(root)
	if err != nil {
		t.Fatalf("ModifiedFileCount: %v", err)
	}
	if n != 1 {
		t.Fatalf("modified tree = %d, want 1", n)
	}
}
