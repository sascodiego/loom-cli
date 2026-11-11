package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests drive the state-backed subcommands through Handle, the same entry
// point main uses, against real temp git repositories.

func newGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func newEnv(t *testing.T, root, stdin string) (*Env, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	return &Env{Root: root, Stdout: &out, Stderr: &errb, Stdin: strings.NewReader(stdin)}, &out, &errb
}

func execCmd(t *testing.T, env *Env, name, stdin string) error {
	t.Helper()
	env.Stdin = strings.NewReader(stdin)
	return Handle(context.Background(), env, name, inputArgs(name, stdin))
}

// inputArgs keeps the flag surface in one place: every state-backed command
// accepts --input - so tests never write temp files.
func inputArgs(name, stdin string) []string {
	switch name {
	case "detect":
		return nil
	case "record":
		return []string{"--input", "-"}
	default:
		return nil
	}
}

func currentBranch(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("branch: %v", err)
	}
	return strings.TrimSpace(string(out))
}

const validRecordInput = `{
  "slug": "339-cedula",
  "method": "odd",
  "tier": "medium",
  "request_id": "req-1",
  "provider": "cloudflare",
  "model": "@cf/cloudflare/clef",
  "criteria_hash": "deadbeef",
  "decided_by": "clef",
  "human_approved": true,
  "suggested_method": "odd",
  "suggested_tier": "medium",
  "payload_hash": "cafebabe"
}`

func decodeRecord(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	var rec map[string]any
	if err := json.Unmarshal(out.Bytes(), &rec); err != nil {
		t.Fatalf("stdout is not one JSON document: %v (%q)", err, out.String())
	}
	return rec
}

func TestDetectThenRecordThenClose(t *testing.T) {
	root := newGitRepo(t)
	env, out, errb := newEnv(t, root, "")

	if err := Handle(context.Background(), env, "detect", nil); err != nil {
		t.Fatalf("detect: %v", err)
	}
	var det struct {
		Candidates []any  `json:"candidates"`
		Verdict    string `json:"verdict"`
	}
	if err := json.Unmarshal(out.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	if det.Verdict != "new" || len(det.Candidates) != 0 {
		t.Fatalf("detect = %+v, want new with no candidates", det)
	}

	out.Reset()
	env.Stdin = strings.NewReader(validRecordInput)
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	rec := decodeRecord(t, out)
	routingID, _ := rec["routing_id"].(string)
	if routingID == "" {
		t.Fatalf("record output has no routing_id: %v", rec)
	}
	if rec["branch"] != currentBranch(t, root) {
		t.Fatalf("record branch = %v, want the checked-out branch", rec["branch"])
	}
	if rec["status"] != "open" {
		t.Fatalf("status = %v, want open", rec["status"])
	}
	if errb.Len() != 0 {
		t.Fatalf("unexpected stderr on the happy path: %q", errb.String())
	}

	out.Reset()
	if err := Handle(context.Background(), env, "detect", nil); err != nil {
		t.Fatalf("detect after record: %v", err)
	}
	if err := json.Unmarshal(out.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	if det.Verdict != "continuation" || len(det.Candidates) != 1 {
		t.Fatalf("detect = %+v, want continuation with one candidate", det)
	}

	out.Reset()
	env2, out2, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), env2, "close", []string{"--routing-id", routingID}); err != nil {
		t.Fatalf("close: %v", err)
	}
	closed := decodeRecord(t, out2)
	if closed["status"] != "closed" {
		t.Fatalf("status = %v, want closed", closed["status"])
	}

	out.Reset()
	if err := Handle(context.Background(), env, "detect", nil); err != nil {
		t.Fatalf("detect after close: %v", err)
	}
	if err := json.Unmarshal(out.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	if det.Verdict != "new" {
		t.Fatalf("verdict after close = %q, want new", det.Verdict)
	}

	history := filepath.Join(root, ".router", "history", routingID+".jsonl")
	data, err := os.ReadFile(history)
	if err != nil {
		t.Fatalf("history file: %v", err)
	}
	lines := strings.Count(strings.TrimSpace(string(data)), "\n") + 1
	if lines != 2 {
		t.Fatalf("history has %d lines, want 2 (decided, closed):\n%s", lines, data)
	}
}

func TestRecordRejectsInconsistentGateOutcomes(t *testing.T) {
	root := newGitRepo(t)
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "clef without approval",
			json: strings.Replace(validRecordInput, `"human_approved": true`, `"human_approved": false`, 1),
			want: "requires human_approved: true",
		},
		{
			name: "clef with a different method",
			json: strings.Replace(validRecordInput, `"method": "odd"`, `"method": "asdd"`, 1),
			want: "is an override",
		},
		{
			name: "override equal to suggestion",
			json: strings.Replace(validRecordInput,
				`"decided_by": "clef",
  "human_approved": true`, `"decided_by": "user_gate_override",
  "human_approved": false, "user_choice": "odd"`, 1),
			want: "is not an override",
		},
		{
			name: "re-route origin on record",
			json: strings.Replace(validRecordInput, `"decided_by": "clef"`, `"decided_by": "override_re_route"`, 1),
			want: "belongs to `router re-route`",
		},
		{
			name: "unknown field",
			json: strings.Replace(validRecordInput, `"slug"`, `"extra": 1, "slug"`, 1),
			want: "unknown field",
		},
		{
			name: "missing payload hash",
			json: strings.Replace(validRecordInput, `"payload_hash": "cafebabe"`, `"payload_hash": ""`, 1),
			want: "missing payload_hash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, out, _ := newEnv(t, root, tt.json)
			err := Handle(context.Background(), env, "record", []string{"--input", "-"})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
			if ExitCode(err) != ExitValidation {
				t.Fatalf("exit = %d, want %d", ExitCode(err), ExitValidation)
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty on failure", out.String())
			}
		})
	}
}

func TestCloseRejectsUnknownAndRepeatedClose(t *testing.T) {
	root := newGitRepo(t)
	env, _, _ := newEnv(t, root, validRecordInput)
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v", err)
	}

	envC, out, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), envC, "close", []string{"--routing-id", "does-not-exist"}); err == nil {
		t.Fatal("close accepted an unknown routing id")
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}

	out.Reset()
	if err := Handle(context.Background(), envC, "detect", nil); err != nil {
		t.Fatalf("detect: %v", err)
	}
	var det struct {
		Candidates []struct {
			RoutingID string `json:"routing_id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(out.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	id := det.Candidates[0].RoutingID

	if err := Handle(context.Background(), envC, "close", []string{"--routing-id", id}); err != nil {
		t.Fatalf("first close: %v", err)
	}
	err := Handle(context.Background(), envC, "close", []string{"--routing-id", id})
	if err == nil || !strings.Contains(err.Error(), "already closed") {
		t.Fatalf("second close err = %v, want already closed", err)
	}
}
