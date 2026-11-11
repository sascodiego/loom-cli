package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"loom/router/internal/state"
)

// These tests build routing history directly through the state package's own
// write paths (WriteRecord + Append), exactly as a real routing session would,
// and then assert what `router metrics` measures from it.

// metricsEnv runs `router metrics` and decodes its stdout JSON.
func runMetricsCmd(t *testing.T, root string, args []string) (map[string]any, string) {
	t.Helper()
	env, out, errb := newEnv(t, root, "")
	if err := Handle(context.Background(), env, "metrics", args); err != nil {
		t.Fatalf("metrics: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("metrics stdout is not JSON: %v (%q)", err, out.String())
	}
	return got, errb.String()
}

// decideOn writes one routing decision through the state write paths: the open
// record plus the history event a first decision produces. Each call gets its
// own fresh routing id, like a real routing session.
func decideOn(t *testing.T, root string, decidedBy state.DecidedBy, approved bool, method string) string {
	t.Helper()
	id, err := state.NewRoutingID()
	if err != nil {
		t.Fatalf("mint routing id: %v", err)
	}
	rec := &state.Record{
		SchemaVersion: state.SchemaVersion,
		RoutingID:     id,
		Slug:          "metrics-fixture",
		Branch:        "main",
		BranchEncoded: state.EncodeBranch("main"),
		Method:        method,
		Tier:          "medium",
		DecidedAt:     time.Now().UTC(),
		RequestID:     "req-" + id[:8],
		Provider:      "cloudflare",
		Model:         "@cf/cloudflare/clef",
		CriteriaHash:  "deadbeef",
		DecidedBy:     decidedBy,
		HumanApproved: approved,
		PayloadHash:   "cafebabe",
		Status:        state.StatusOpen,
	}
	if err := state.WriteRecord(root, rec); err != nil {
		t.Fatalf("write record: %v", err)
	}
	ev := state.Event{
		Event:         eventFor(decidedBy),
		DecidedBy:     decidedBy,
		HumanApproved: approved,
		UserChoice:    rec.UserChoice,
		Record:        rec,
	}
	if err := state.Append(root, ev); err != nil {
		t.Fatalf("append history: %v", err)
	}
	return id
}

func TestMetricsSuppressesBelowTheSampleFloor(t *testing.T) {
	root := newGitRepo(t)
	for i := 0; i < 5; i++ {
		decideOn(t, root, state.DecidedByClef, true, "odd")
	}

	got, _ := runMetricsCmd(t, root, nil)
	if got["sample"] != float64(5) {
		t.Fatalf("sample = %v, want 5", got["sample"])
	}
	if got["suppressed"] != true {
		t.Fatalf("suppressed = %v, want true below the floor", got["suppressed"])
	}
	rate, present := got["agreement_rate"]
	if present && rate != nil {
		t.Fatalf("agreement_rate = %v, want null while suppressed", rate)
	}
	if got["floor"] != float64(SampleFloor) {
		t.Fatalf("floor = %v, want %d", got["floor"], SampleFloor)
	}
}

func TestMetricsComputesTheRateAtAndAboveTheFloor(t *testing.T) {
	root := newGitRepo(t)
	for i := 0; i < 27; i++ {
		decideOn(t, root, state.DecidedByClef, true, "odd")
	}
	for i := 0; i < 3; i++ {
		decideOn(t, root, state.DecidedByUserGateOverride, false, "ftd")
	}

	got, _ := runMetricsCmd(t, root, nil)
	if got["suppressed"] != false {
		t.Fatalf("suppressed = %v, want false at the floor", got["suppressed"])
	}
	rate, ok := got["agreement_rate"].(float64)
	if !ok {
		t.Fatalf("agreement_rate = %v, want the number 0.9", got["agreement_rate"])
	}
	if rate != 0.9 {
		t.Fatalf("agreement_rate = %v, want 0.9 (27 approved of 30)", rate)
	}
	if got["sample"] != float64(30) {
		t.Fatalf("sample = %v, want 30 (clean + disagreements)", got["sample"])
	}

	byDecidedBy, ok := got["by_decided_by"].(map[string]any)
	if !ok || byDecidedBy["clef"] != float64(27) || byDecidedBy["user_gate_override"] != float64(3) {
		t.Fatalf("by_decided_by = %v, want clef 27 / user_gate_override 3", got["by_decided_by"])
	}
	byMethod, ok := got["by_method"].(map[string]any)
	if !ok {
		t.Fatalf("by_method = %v, want a stratified tally", got["by_method"])
	}
	odd, ok := byMethod["odd"].(map[string]any)
	if !ok || odd["decisions"] != float64(27) || odd["approved"] != float64(27) {
		t.Fatalf("by_method[odd] = %v, want 27 decisions, 27 approved", byMethod["odd"])
	}
	ftd, ok := byMethod["ftd"].(map[string]any)
	if !ok || ftd["decisions"] != float64(3) || ftd["approved"] != float64(0) {
		t.Fatalf("by_method[ftd] = %v, want 3 decisions, 0 approved", byMethod["ftd"])
	}
}

func TestMetricsExcludesFallbackAndReRouteFromTheDenominator(t *testing.T) {
	root := newGitRepo(t)
	for i := 0; i < 30; i++ {
		decideOn(t, root, state.DecidedByClef, true, "odd")
	}
	decideOn(t, root, state.DecidedByJointFallback, false, "ftd")
	decideOn(t, root, state.DecidedByJointFallback, false, "asdd")
	decideOn(t, root, state.DecidedByOverrideReRoute, false, "gentle-sdd")

	got, _ := runMetricsCmd(t, root, nil)
	if got["sample"] != float64(30) {
		t.Fatalf("sample = %v, want 30 (fallback and re-route excluded)", got["sample"])
	}
	excluded, ok := got["excluded"].(map[string]any)
	if !ok {
		t.Fatalf("excluded = %v, want the excluded tally", got["excluded"])
	}
	if excluded["joint_fallback"] != float64(2) {
		t.Fatalf("excluded[joint_fallback] = %v, want 2", excluded["joint_fallback"])
	}
	if excluded["override_re_route"] != float64(1) {
		t.Fatalf("excluded[override_re_route] = %v, want 1", excluded["override_re_route"])
	}
	if len(excluded) != 2 {
		t.Fatalf("excluded = %v, want exactly the two declared keys", excluded)
	}
	rate, ok := got["agreement_rate"].(float64)
	if !ok || rate != 1.0 {
		t.Fatalf("agreement_rate = %v, want 1.0 over the 30 clean decisions", got["agreement_rate"])
	}
}

func TestMetricsSurvivesAMissingOrMalformedHistoryDir(t *testing.T) {
	// Missing dir: no history at all is an empty measurement, not an error.
	root := newGitRepo(t)
	got, stderr := runMetricsCmd(t, root, nil)
	if got["sample"] != float64(0) || got["suppressed"] != true {
		t.Fatalf("missing dir: sample = %v suppressed = %v, want 0 / true", got["sample"], got["suppressed"])
	}
	if got["agreement_rate"] != nil {
		t.Fatalf("missing dir: agreement_rate = %v, want null", got["agreement_rate"])
	}
	if _, err := os.Stat(filepath.Join(root, ".router", "history")); err == nil {
		t.Fatal("metrics must not create the history dir")
	}

	// Malformed file: skipped with a warning on stderr, never a failure.
	root2 := newGitRepo(t)
	if err := os.MkdirAll(state.HistoryDir(root2), 0o755); err != nil {
		t.Fatalf("mkdir history: %v", err)
	}
	garbage := filepath.Join(state.HistoryDir(root2), "00000000-0000-4000-8000-00000000000f.jsonl")
	if err := os.WriteFile(garbage, []byte("{not json\n"), 0o644); err != nil {
		t.Fatalf("write garbage: %v", err)
	}
	decideOn(t, root2, state.DecidedByClef, true, "odd")

	got2, stderr2 := runMetricsCmd(t, root2, nil)
	if got2["sample"] != float64(1) {
		t.Fatalf("malformed file: sample = %v, want the one parseable decision", got2["sample"])
	}
	if got2["suppressed"] != true {
		t.Fatalf("malformed file: suppressed = %v, want true", got2["suppressed"])
	}
	if stderr == "" {
		// The missing-dir run stays quiet; only the malformed one warns.
		t.Log("stderr (informational):", stderr)
	}
	if stderr2 == "" {
		t.Fatal("a malformed history file must be reported on stderr, not skipped silently")
	}
}

func TestMetricsCountsOneDecisionPerRoutingID(t *testing.T) {
	root := newGitRepo(t)
	for i := 0; i < 30; i++ {
		id := decideOn(t, root, state.DecidedByClef, true, "odd")
		// A later lifecycle event on the same routing id must not count as a
		// second decision: metrics takes the first decision event only.
		rec, _, err := state.FindRecord(root, id)
		if err != nil {
			t.Fatalf("find record: %v", err)
		}
		rec.Status = state.StatusClosed
		if err := state.WriteRecord(root, rec); err != nil {
			t.Fatalf("close record: %v", err)
		}
		if err := state.Append(root, state.Event{
			Event:     state.EventClosed,
			DecidedBy: rec.DecidedBy,
			Record:    rec,
		}); err != nil {
			t.Fatalf("append close: %v", err)
		}
	}

	got, _ := runMetricsCmd(t, root, nil)
	if got["sample"] != float64(30) {
		t.Fatalf("sample = %v, want 30 — one decision per routing id despite closed events", got["sample"])
	}
	rate, ok := got["agreement_rate"].(float64)
	if !ok || rate != 1.0 {
		t.Fatalf("agreement_rate = %v, want 1.0", got["agreement_rate"])
	}
}
