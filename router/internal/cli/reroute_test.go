package cli

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"loom/router/internal/state"
)

func recordOne(t *testing.T, root, input string) map[string]any {
	t.Helper()
	env, out, _ := newEnv(t, root, input)
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	var rec map[string]any
	if err := json.Unmarshal(out.Bytes(), &rec); err != nil {
		t.Fatalf("record output: %v", err)
	}
	return rec
}

func TestReRouteSupersedesAndOpensTheNewMethod(t *testing.T) {
	root := routerRepo(t)
	old := recordOne(t, root, validRecordInput)
	oldID, _ := old["routing_id"].(string)

	env, out, _ := newEnv(t, root, "")
	err := Handle(context.Background(), env, "re-route", []string{
		"--routing-id", oldID,
		"--method", "ftd",
		"--justification", "scope grew past one subsystem",
		"--requested-by", "agent",
	})
	if err != nil {
		t.Fatalf("re-route: %v", err)
	}
	var fresh map[string]any
	if err := json.Unmarshal(out.Bytes(), &fresh); err != nil {
		t.Fatalf("re-route output: %v", err)
	}
	newID, _ := fresh["routing_id"].(string)
	if newID == "" || newID == oldID {
		t.Fatalf("new routing id = %q, want a fresh one", newID)
	}
	if fresh["method"] != "ftd" || fresh["decided_by"] != "override_re_route" ||
		fresh["requested_by"] != "agent" || fresh["status"] != "open" {
		t.Fatalf("new record = %+v", fresh)
	}
	if fresh["human_approved"] != false {
		t.Fatal("a re-route is not a human-approved clef decision")
	}
	if fresh["tier"] != old["tier"] {
		t.Fatalf("tier = %v, want the inherited %v", fresh["tier"], old["tier"])
	}

	superseded, _, err := state.FindRecord(root, oldID)
	if err != nil {
		t.Fatalf("find old: %v", err)
	}
	if superseded.Status != state.StatusSuperseded || superseded.SupersededBy != newID {
		t.Fatalf("old record = status %q superseded_by %q", superseded.Status, superseded.SupersededBy)
	}

	// detect now reports only the fresh record.
	detEnv, detOut, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), detEnv, "detect", nil); err != nil {
		t.Fatalf("detect: %v", err)
	}
	var det struct {
		Candidates []struct {
			RoutingID string `json:"routing_id"`
		} `json:"candidates"`
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(detOut.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	if det.Verdict != "continuation" || len(det.Candidates) != 1 || det.Candidates[0].RoutingID != newID {
		t.Fatalf("detect = %+v, want only the fresh record", det)
	}

	// The audit trail tells the whole story.
	oldHistory, err := state.ReadHistory(root, oldID)
	if err != nil {
		t.Fatalf("old history: %v", err)
	}
	if len(oldHistory) != 3 {
		t.Fatalf("old history has %d events, want 3 (decided, re_route_requested, superseded)", len(oldHistory))
	}
	if oldHistory[1].Event != state.EventReRouteRequested || oldHistory[1].Justification == "" {
		t.Fatalf("request event = %+v", oldHistory[1])
	}
	if oldHistory[2].Event != state.EventSuperseded {
		t.Fatalf("last old event = %q, want superseded", oldHistory[2].Event)
	}
	newHistory, err := state.ReadHistory(root, newID)
	if err != nil {
		t.Fatalf("new history: %v", err)
	}
	if len(newHistory) != 1 || newHistory[0].Event != state.EventReRouteRequested {
		t.Fatalf("new history = %+v", newHistory)
	}
}

func TestReRouteRejectsBadInput(t *testing.T) {
	root := routerRepo(t)
	rec := recordOne(t, root, validRecordInput)
	id, _ := rec["routing_id"].(string)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown record", []string{"--routing-id", "nope", "--method", "ftd"}, "no open record"},
		{"missing method", []string{"--routing-id", id}, "missing --method"},
		{"unavailable method", []string{"--routing-id", id, "--method", "egm-review-loop"}, "unavailable"},
		{"unknown method", []string{"--routing-id", id, "--method", "vibes"}, "unknown method"},
		{"bad requester", []string{"--routing-id", id, "--method", "ftd", "--requested-by", "robot"}, "requested-by"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, out, _ := newEnv(t, root, "")
			err := Handle(context.Background(), env, "re-route", tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want %q", err, tt.want)
			}
			if ExitCode(err) != ExitValidation {
				t.Fatalf("exit = %d, want %d", ExitCode(err), ExitValidation)
			}
			if out.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", out.String())
			}
		})
	}

	// A closed record cannot be superseded.
	envC, _, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), envC, "close", []string{"--routing-id", id}); err != nil {
		t.Fatalf("close: %v", err)
	}
	envR, _, _ := newEnv(t, root, "")
	err := Handle(context.Background(), envR, "re-route", []string{"--routing-id", id, "--method", "ftd"})
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("re-route on closed = %v, want a closed-record refusal", err)
	}
}

func TestReRouteNeverTouchesTheWorkingTree(t *testing.T) {
	root := routerRepo(t)
	rec := recordOne(t, root, validRecordInput)
	id, _ := rec["routing_id"].(string)

	before, err := state.Detect(root, currentBranch(t, root))
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	env, _, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), env, "re-route", []string{
		"--routing-id", id, "--method", "ftd", "--requested-by", "user",
	}); err != nil {
		t.Fatalf("re-route: %v", err)
	}
	after, err := state.Detect(root, currentBranch(t, root))
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(after.Candidates) != len(before.Candidates) {
		t.Fatalf("open records changed from %d to %d: re-route must only record",
			len(before.Candidates), len(after.Candidates))
	}
	// And no source file was written.
	entries, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("re-route wrote into the working tree: %v", entries)
	}
}
