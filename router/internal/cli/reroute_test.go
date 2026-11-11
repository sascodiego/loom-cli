package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loom/router/internal/provider"
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

// TestReRouteClassifiesFreshProvenance pins S2: the fresh record's provenance
// comes from a real classification of the re-declared pre-flight, not from the
// predecessor. The fake provider answers high/odd while the predecessor was
// recorded medium/odd, so an inherited field could never satisfy the assertions.
func TestReRouteClassifiesFreshProvenance(t *testing.T) {
	root := routerRepo(t)
	old := recordOne(t, root, validRecordInput)
	oldID, _ := old["routing_id"].(string)

	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.15, "high": 0.75},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}}
	withFakeProvider(t, root, fake)

	env, out, _ := newEnv(t, root, decideDeclaration)
	err := Handle(context.Background(), env, "re-route", []string{
		"--routing-id", oldID,
		"--method", "ftd",
		"--input", "-",
	})
	if err != nil {
		t.Fatalf("re-route: %v", err)
	}
	var fresh map[string]any
	if err := json.Unmarshal(out.Bytes(), &fresh); err != nil {
		t.Fatalf("re-route output: %v", err)
	}

	if fresh["tier"] != "high" || fresh["tier"] == old["tier"] {
		t.Fatalf("tier = %v, want the classification's high, not the inherited %v",
			fresh["tier"], old["tier"])
	}
	if fresh["suggested_tier"] != "high" {
		t.Fatalf("suggested_tier = %v, want the classification's tier", fresh["suggested_tier"])
	}
	if fresh["request_id"] != "trace-42" || fresh["request_id"] == old["request_id"] {
		t.Fatalf("request_id = %v, want the provider's fresh trace id", fresh["request_id"])
	}
	sum := sha256.Sum256([]byte(decideDeclaration))
	if fresh["payload_hash"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("payload_hash = %v, want the hash of the re-declared declaration", fresh["payload_hash"])
	}
	h := sha256.New()
	for _, part := range []string{"odd criteria\n", "ftd criteria\n", "egm criteria\n"} {
		h.Write([]byte(part))
	}
	if fresh["criteria_hash"] != hex.EncodeToString(h.Sum(nil)) {
		t.Fatalf("criteria_hash = %v, want the registry-order hash", fresh["criteria_hash"])
	}
	// The method stays the user-approved choice; Clef's answer is preserved
	// beside it as the suggestion.
	if fresh["method"] != "ftd" {
		t.Fatalf("method = %v, want the --method target", fresh["method"])
	}
	if fresh["suggested_method"] != "odd" {
		t.Fatalf("suggested_method = %v, want Clef's answer", fresh["suggested_method"])
	}
	if fresh["provider"] != "cloudflare" || fresh["model"] != "@cf/cloudflare/clef" {
		t.Fatalf("provider/model = %v/%v", fresh["provider"], fresh["model"])
	}
}

// TestReRouteClassificationFailurePersistsNothing pins the S2 ordering: the
// classification runs before anything is written, so a bad declaration or a
// provider failure leaves the predecessor open, detect with its one candidate
// and the history file untouched.
func TestReRouteClassificationFailurePersistsNothing(t *testing.T) {
	root := routerRepo(t)
	old := recordOne(t, root, validRecordInput)
	oldID, _ := old["routing_id"].(string)

	historyBefore, err := state.ReadHistory(root, oldID)
	if err != nil {
		t.Fatalf("history: %v", err)
	}

	assertUntouched := func(t *testing.T) {
		t.Helper()
		after, _, err := state.FindRecord(root, oldID)
		if err != nil {
			t.Fatalf("find predecessor: %v", err)
		}
		if after.Status != state.StatusOpen {
			t.Fatalf("predecessor status = %q, want it still open", after.Status)
		}
		detEnv, detOut, _ := newEnv(t, root, "")
		if err := Handle(context.Background(), detEnv, "detect", nil); err != nil {
			t.Fatalf("detect: %v", err)
		}
		var det struct {
			Candidates []struct {
				RoutingID string `json:"routing_id"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(detOut.Bytes(), &det); err != nil {
			t.Fatalf("detect output: %v", err)
		}
		if len(det.Candidates) != 1 || det.Candidates[0].RoutingID != oldID {
			t.Fatalf("detect = %+v, want only the untouched predecessor", det)
		}
		historyAfter, err := state.ReadHistory(root, oldID)
		if err != nil {
			t.Fatalf("history: %v", err)
		}
		if len(historyAfter) != len(historyBefore) {
			t.Fatalf("history grew from %d to %d events on a failed classification",
				len(historyBefore), len(historyAfter))
		}
	}

	t.Run("invalid declaration", func(t *testing.T) {
		withFakeProvider(t, root, &fakeProvider{})
		bad := strings.Replace(decideDeclaration, `"subsystems": ["router"]`, `"subsystems": ["billing"]`, 1)
		env, out, _ := newEnv(t, root, bad)
		err := Handle(context.Background(), env, "re-route", []string{
			"--routing-id", oldID, "--method", "ftd", "--input", "-",
		})
		if err == nil || ExitCode(err) != ExitValidation {
			t.Fatalf("err = %v, exit = %d, want a validation failure", err, ExitCode(err))
		}
		if out.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", out.String())
		}
		assertUntouched(t)
	})

	t.Run("provider failure", func(t *testing.T) {
		withFakeProvider(t, root, &fakeProvider{
			err: &provider.Failure{Kind: provider.FailureQuota, Status: 429, Attempts: 3},
		})
		env, out, _ := newEnv(t, root, decideDeclaration)
		err := Handle(context.Background(), env, "re-route", []string{
			"--routing-id", oldID, "--method", "ftd", "--input", "-",
		})
		if err == nil || ExitCode(err) != ExitProvider {
			t.Fatalf("err = %v, exit = %d, want a provider failure", err, ExitCode(err))
		}
		if out.Len() != 0 {
			t.Fatalf("stdout = %q, want empty", out.String())
		}
		assertUntouched(t)
	})
}

func TestReRouteSupersedesAndOpensTheNewMethod(t *testing.T) {
	root := routerRepo(t)
	old := recordOne(t, root, validRecordInput)
	oldID, _ := old["routing_id"].(string)

	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.15, "high": 0.75},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}}
	withFakeProvider(t, root, fake)

	env, out, _ := newEnv(t, root, decideDeclaration)
	err := Handle(context.Background(), env, "re-route", []string{
		"--routing-id", oldID,
		"--method", "ftd",
		"--input", "-",
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
	if fresh["tier"] != "high" {
		t.Fatalf("tier = %v, want the classification's high, not the inherited %v",
			fresh["tier"], old["tier"])
	}
	if fresh["suggested_tier"] != "high" {
		t.Fatalf("suggested_tier = %v, want the classification's tier", fresh["suggested_tier"])
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

	withFakeProvider(t, root, &fakeProvider{})

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"unknown record", []string{"--routing-id", "nope", "--method", "ftd", "--input", "-"}, "no open record"},
		{"missing method", []string{"--routing-id", id}, "missing --method"},
		{"missing input", []string{"--routing-id", id, "--method", "ftd"}, "missing --input"},
		{"unavailable method", []string{"--routing-id", id, "--method", "egm-review-loop", "--input", "-"}, "unavailable"},
		{"unknown method", []string{"--routing-id", id, "--method", "vibes", "--input", "-"}, "unknown method"},
		{"bad requester", []string{"--routing-id", id, "--method", "ftd", "--input", "-", "--requested-by", "robot"}, "requested-by"},
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
	err := Handle(context.Background(), envR, "re-route", []string{"--routing-id", id, "--method", "ftd", "--input", "-"})
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
	withFakeProvider(t, root, &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.2, "medium": 0.6, "high": 0.2},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}})
	env, _, _ := newEnv(t, root, decideDeclaration)
	if err := Handle(context.Background(), env, "re-route", []string{
		"--routing-id", id, "--method", "ftd", "--input", "-", "--requested-by", "user",
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

// openRecord builds an in-memory record for the supersede unit tests.
func openRecord(t *testing.T, branch string) *state.Record {
	t.Helper()
	id, err := state.NewRoutingID()
	if err != nil {
		t.Fatalf("NewRoutingID: %v", err)
	}
	return &state.Record{
		SchemaVersion: state.SchemaVersion,
		RoutingID:     id,
		Slug:          "supersede-fixture",
		Branch:        branch,
		BranchEncoded: state.EncodeBranch(branch),
		Method:        "odd",
		Tier:          "medium",
		DecidedAt:     time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC),
		RequestID:     "req-" + id[:8],
		Provider:      "cloudflare",
		Model:         "@cf/cloudflare/clef",
		CriteriaHash:  "deadbeef",
		DecidedBy:     state.DecidedByClef,
		HumanApproved: true,
		PayloadHash:   "cafebabe",
		Status:        state.StatusOpen,
	}
}

// fakeRecordStore records the write order and can fail a chosen write.
type fakeRecordStore struct {
	failOn  string // routing id whose write fails
	writes  []string
	deletes []string
}

func (f *fakeRecordStore) Write(r *state.Record) error {
	f.writes = append(f.writes, r.RoutingID)
	if r.RoutingID == f.failOn {
		return errInjectedWrite
	}
	return nil
}

func (f *fakeRecordStore) Delete(r *state.Record) error {
	f.deletes = append(f.deletes, r.RoutingID)
	return nil
}

var errInjectedWrite = errors.New("injected write failure")

// The supersede must never leave the old record marked superseded without a
// successor: the successor is written first, and a failure on the old record
// rolls the successor back so the branch keeps exactly one open record.
func TestSupersedeWritesSuccessorFirstAndRollsBack(t *testing.T) {
	old := openRecord(t, "main")
	fresh := openRecord(t, "main")

	t.Run("old write fails", func(t *testing.T) {
		store := &fakeRecordStore{failOn: old.RoutingID}
		err := supersedeRecords(store, old, fresh)
		if err == nil {
			t.Fatal("supersede succeeded despite a failed write")
		}
		if len(store.writes) != 2 || store.writes[0] != fresh.RoutingID {
			t.Fatalf("write order = %v, want the successor first", store.writes)
		}
		if len(store.deletes) != 1 || store.deletes[0] != fresh.RoutingID {
			t.Fatalf("deletes = %v, want the successor rolled back", store.deletes)
		}
	})

	t.Run("successor write fails", func(t *testing.T) {
		store := &fakeRecordStore{failOn: fresh.RoutingID}
		if err := supersedeRecords(store, old, fresh); err == nil {
			t.Fatal("supersede succeeded despite a failed successor write")
		}
		if len(store.writes) != 1 {
			t.Fatalf("writes = %v, want only the failed successor attempt", store.writes)
		}
		if len(store.deletes) != 0 {
			t.Fatalf("deletes = %v, want none: nothing was written to roll back", store.deletes)
		}
	})

	t.Run("both writes succeed", func(t *testing.T) {
		store := &fakeRecordStore{}
		if err := supersedeRecords(store, old, fresh); err != nil {
			t.Fatalf("supersede: %v", err)
		}
		if len(store.deletes) != 0 {
			t.Fatalf("deletes = %v, want none", store.deletes)
		}
	})
}
