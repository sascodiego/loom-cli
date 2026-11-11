package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"loom/router/internal/state"
)

// End-to-end coverage of the critical user journeys over one temp repository,
// driving the same Handle entry point the binary uses. Success paths assert
// that stdout stays JSON-only (exit 0, decodable documents) and exit codes
// come back through ExitCode.

// overrideRecordInput is the gate-override variant of the canonical record: the
// user rejected the odd suggestion and picked ftd at the confirmation gate.
const overrideRecordInput = `{
  "slug": "339-cedula",
  "method": "ftd",
  "tier": "medium",
  "request_id": "req-1",
  "provider": "cloudflare",
  "model": "@cf/cloudflare/clef",
  "criteria_hash": "deadbeef",
  "decided_by": "user_gate_override",
  "human_approved": false,
  "suggested_method": "odd",
  "suggested_tier": "medium",
  "user_choice": "ftd",
  "payload_hash": "cafebabe"
}`

// recordFromDecide builds the record input the harness sends after the user
// confirms a decide output: the decided facts plus the gate outcome.
func recordFromDecide(t *testing.T, decided map[string]any) string {
	t.Helper()
	payload := map[string]any{
		"slug":             "auth-ui",
		"method":           decided["method"],
		"tier":             decided["tier"],
		"request_id":       decided["request_id"],
		"provider":         decided["provider"],
		"model":            decided["model"],
		"criteria_hash":    decided["criteria_hash"],
		"decided_by":       "clef",
		"human_approved":   true,
		"suggested_method": decided["method"],
		"suggested_tier":   decided["tier"],
		"payload_hash":     decided["payload_hash"],
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal record input: %v", err)
	}
	return string(data)
}

func TestE2EDecideRecordDetectClose(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.75, "high": 0.15},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}}
	withFakeProvider(t, root, fake)

	// CUJ 1, step 1: decide.
	decided, err, _ := runDecide(t, root, decideDeclaration)
	if err != nil || ExitCode(err) != 0 {
		t.Fatalf("decide: %v (exit %d)", err, ExitCode(err))
	}

	// CUJ 1, step 2: the user confirms; record persists the decision.
	env, out, errb := newEnv(t, root, recordFromDecide(t, decided))
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v (exit %d)", err, ExitCode(err))
	}
	if errb.Len() != 0 {
		t.Fatalf("record stderr = %q, want quiet on a fresh branch", errb.String())
	}
	rec := decodeRecord(t, out)
	if rec["decided_by"] != "clef" || rec["human_approved"] != true || rec["status"] != "open" {
		t.Fatalf("record = %+v", rec)
	}
	routingID, _ := rec["routing_id"].(string)

	// CUJ 2: a continuation is recognized from state alone, no provider call.
	if fake.answerCalls != 1 {
		t.Fatalf("provider called %d times, want exactly one (the decide)", fake.answerCalls)
	}
	envD, outD, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), envD, "detect", nil); err != nil {
		t.Fatalf("detect: %v", err)
	}
	var det struct {
		Candidates []struct {
			RoutingID string `json:"routing_id"`
		} `json:"candidates"`
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(outD.Bytes(), &det); err != nil {
		t.Fatalf("detect output: %v", err)
	}
	if det.Verdict != "continuation" || len(det.Candidates) != 1 || det.Candidates[0].RoutingID != routingID {
		t.Fatalf("detect = %+v, want the recorded candidate", det)
	}

	// Ship: close marks the record closed and appends the closing event.
	envC, outC, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), envC, "close", []string{"--routing-id", routingID}); err != nil {
		t.Fatalf("close: %v (exit %d)", err, ExitCode(err))
	}
	if closed := decodeRecord(t, outC); closed["status"] != "closed" {
		t.Fatalf("status = %v, want closed", closed["status"])
	}
	events, err := state.ReadHistory(root, routingID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(events) != 2 || events[0].Event != state.EventDecided || events[1].Event != state.EventClosed {
		t.Fatalf("history = %+v, want decided then closed", events)
	}
}

func TestE2EGateOverrideLandsInRecordAndHistory(t *testing.T) {
	root := routerRepo(t)
	env, out, _ := newEnv(t, root, overrideRecordInput)
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v (exit %d)", err, ExitCode(err))
	}
	rec := decodeRecord(t, out)
	routingID, _ := rec["routing_id"].(string)
	if rec["decided_by"] != "user_gate_override" || rec["human_approved"] != false {
		t.Fatalf("record = %+v, want the override origin with no approval", rec)
	}
	if rec["method"] != "ftd" || rec["user_choice"] != "ftd" || rec["suggested_method"] != "odd" {
		t.Fatalf("record = %+v, want ftd chosen over the odd suggestion", rec)
	}

	events, err := state.ReadHistory(root, routingID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(events) != 1 || events[0].Event != state.EventGateOverridden {
		t.Fatalf("history = %+v, want one gate_overridden event", events)
	}
}

func TestE2EReRouteSupersedesAndRestartsUnderTheTargetMethod(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.75, "high": 0.15},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}}
	withFakeProvider(t, root, fake)

	// The change was routed to odd and recorded.
	decided, err, _ := runDecide(t, root, decideDeclaration)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	env, out, _ := newEnv(t, root, recordFromDecide(t, decided))
	if err := Handle(context.Background(), env, "record", []string{"--input", "-"}); err != nil {
		t.Fatalf("record: %v", err)
	}
	oldRec := decodeRecord(t, out)
	oldID, _ := oldRec["routing_id"].(string)

	// CUJ 4: the work grew past odd; the re-route supersedes it.
	envR, outR, errb := newEnv(t, root, "")
	if err := Handle(context.Background(), envR, "re-route", []string{
		"--routing-id", oldID,
		"--method", "ftd",
		"--justification", "the change now spans three subsystems and a feature doc",
		"--requested-by", "agent",
	}); err != nil {
		t.Fatalf("re-route: %v (exit %d)", err, ExitCode(err))
	}
	if !strings.Contains(errb.String(), "superseded") {
		t.Fatalf("re-route stderr = %q, want the supersede explained for the user", errb.String())
	}
	fresh := decodeRecord(t, outR)
	newID, _ := fresh["routing_id"].(string)
	if fresh["method"] != "ftd" || fresh["decided_by"] != "override_re_route" || fresh["status"] != "open" {
		t.Fatalf("fresh record = %+v, want ftd / override_re_route / open", fresh)
	}
	if fresh["suggested_method"] != "odd" {
		t.Fatalf("suggested_method = %v, want the original suggestion inherited", fresh["suggested_method"])
	}

	superseded, _, err := state.FindRecord(root, oldID)
	if err != nil {
		t.Fatalf("find old: %v", err)
	}
	if superseded.Status != state.StatusSuperseded || superseded.SupersededBy != newID {
		t.Fatalf("old record = %+v, want superseded by %s", superseded, newID)
	}

	oldHistory, err := state.ReadHistory(root, oldID)
	if err != nil {
		t.Fatalf("old history: %v", err)
	}
	if len(oldHistory) != 3 || oldHistory[0].Event != state.EventDecided ||
		oldHistory[1].Event != state.EventReRouteRequested ||
		oldHistory[2].Event != state.EventSuperseded {
		t.Fatalf("old history = %+v, want decided, re_route_requested, superseded", oldHistory)
	}
	if oldHistory[1].RequestedBy != "agent" || oldHistory[1].Justification == "" {
		t.Fatalf("request event = %+v, want the agent request with its justification", oldHistory[1])
	}
	newHistory, err := state.ReadHistory(root, newID)
	if err != nil {
		t.Fatalf("new history: %v", err)
	}
	if len(newHistory) != 1 || newHistory[0].Event != state.EventReRouteRequested {
		t.Fatalf("new history = %+v, want one re_route_requested event", newHistory)
	}

	// detect now offers only the fresh record: the change restarts under ftd.
	envD, outD, _ := newEnv(t, root, "")
	if err := Handle(context.Background(), envD, "detect", nil); err != nil {
		t.Fatalf("detect: %v", err)
	}
	if !strings.Contains(outD.String(), newID) || strings.Contains(outD.String(), oldID) {
		t.Fatalf("detect stdout = %q, want only the fresh record", outD.String())
	}
}
