package state

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newRepo creates a real git repository in a temp dir: branch keying, path
// layout and history files are exercised against the filesystem, not a mock.
func newRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func sampleRecord(t *testing.T, branch string) *Record {
	t.Helper()
	id, err := NewRoutingID()
	if err != nil {
		t.Fatalf("NewRoutingID: %v", err)
	}
	return &Record{
		SchemaVersion:   SchemaVersion,
		RoutingID:       id,
		Slug:            "339-cedula",
		Branch:          branch,
		BranchEncoded:   EncodeBranch(branch),
		Method:          "odd",
		Tier:            "medium",
		DecidedAt:       time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC),
		RequestID:       "req-1",
		Provider:        "cloudflare",
		Model:           "@cf/cloudflare/clef",
		CriteriaHash:    "deadbeef",
		DecidedBy:       DecidedByClef,
		HumanApproved:   true,
		SuggestedMethod: "odd",
		SuggestedTier:   "medium",
		PayloadHash:     "cafebabe",
		Status:          StatusOpen,
	}
}

func TestEncodeBranch(t *testing.T) {
	tests := []struct{ branch, want string }{
		{"fix/339-cedula", "fix--339-cedula"},
		{"feat/method-router-v1", "feat--method-router-v1"},
		{"main", "main"},
		{"a/b/c", "a--b--c"},
	}
	for _, tt := range tests {
		if got := EncodeBranch(tt.branch); got != tt.want {
			t.Errorf("EncodeBranch(%q) = %q, want %q", tt.branch, got, tt.want)
		}
	}
}

func TestEncodedCollisionIsResolvedByTrueBranch(t *testing.T) {
	// fix/339 and fix--339 encode to the same directory component; the record's
	// true branch is what decides membership.
	if EncodeBranch("fix/339") != EncodeBranch("fix--339") {
		t.Fatalf("expected a collision for this fixture, got %q vs %q",
			EncodeBranch("fix/339"), EncodeBranch("fix--339"))
	}
	r := sampleRecord(t, "fix--339")
	if !r.MatchesBranch("fix--339") {
		t.Error("record should match its true branch")
	}
	if r.MatchesBranch("fix/339") {
		t.Error("record must not match a colliding branch")
	}
}

func TestNewRoutingIDIsUUIDv4(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id, err := NewRoutingID()
		if err != nil {
			t.Fatalf("NewRoutingID: %v", err)
		}
		if len(id) != 36 {
			t.Fatalf("id %q length = %d, want 36", id, len(id))
		}
		if id[14] != '4' {
			t.Fatalf("id %q version nibble = %c, want 4", id, id[14])
		}
		if seen[id] {
			t.Fatalf("duplicate routing id %q", id)
		}
		seen[id] = true
	}
}

func TestRecordRoundTrip(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "fix/339-cedula")
	if err := WriteRecord(root, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	path := RecordPath(root, rec)
	wantPath := filepath.Join(root, ".router", "open", "fix--339-cedula", rec.RoutingID+".json")
	if path != wantPath {
		t.Fatalf("path = %q, want %q", path, wantPath)
	}
	got, err := ReadRecord(path)
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}
	if got.RoutingID != rec.RoutingID || got.Branch != rec.Branch || got.Method != rec.Method {
		t.Fatalf("round trip mismatch: %+v vs %+v", got, rec)
	}
}

func TestReadRecordRejectsUnknownFields(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "main")
	if err := WriteRecord(root, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	path := RecordPath(root, rec)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m["surprise"] = true
	data, _ = json.Marshal(m)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := ReadRecord(path); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("want unknown-field rejection, got %v", err)
	}
}

func TestValidateRejectsIncompleteRecords(t *testing.T) {
	base := func() *Record { return sampleRecord(t, "main") }
	tests := []struct {
		name   string
		mutate func(*Record)
		want   string
	}{
		{"schema version", func(r *Record) { r.SchemaVersion = 2 }, "unknown schema_version"},
		{"routing id", func(r *Record) { r.RoutingID = "" }, "missing routing_id"},
		{"request id", func(r *Record) { r.RequestID = "" }, "missing request_id"},
		{"decided by", func(r *Record) { r.DecidedBy = "vibes" }, "invalid decided_by"},
		{"status", func(r *Record) { r.Status = "draft" }, "invalid status"},
		{"decided at", func(r *Record) { r.DecidedAt = time.Time{} }, "missing decided_at"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := base()
			tt.mutate(r)
			err := r.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want %q", err, tt.want)
			}
			if werr := WriteRecord(t.TempDir(), r); werr == nil {
				t.Fatal("WriteRecord accepted an invalid record")
			}
		})
	}
}

func TestListOpenReportsMalformedAndEmptyDir(t *testing.T) {
	root := newRepo(t)
	recs, err := ListOpen(root, "main")
	if err != nil || len(recs) != 0 {
		t.Fatalf("empty tree: recs=%v err=%v", recs, err)
	}

	good := sampleRecord(t, "main")
	if err := WriteRecord(root, good); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	bad := filepath.Join(OpenDir(root, "main"), "broken.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write broken: %v", err)
	}

	recs, err = ListOpen(root, "main")
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (one valid, one malformed)", len(recs))
	}
	var valid, malformed int
	for _, r := range recs {
		if r.Err != nil {
			malformed++
		} else {
			valid++
		}
	}
	if valid != 1 || malformed != 1 {
		t.Fatalf("valid=%d malformed=%d, want 1/1", valid, malformed)
	}
}

func TestFindRecord(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "feat/x")
	if err := WriteRecord(root, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	got, path, err := FindRecord(root, rec.RoutingID)
	if err != nil {
		t.Fatalf("FindRecord: %v", err)
	}
	if got.RoutingID != rec.RoutingID || path != RecordPath(root, rec) {
		t.Fatalf("FindRecord mismatch: %+v %q", got, path)
	}
	if _, _, err := FindRecord(root, "nope"); err == nil {
		t.Fatal("FindRecord found a record that does not exist")
	}
}

func TestHistoryAppendAndRead(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "main")
	if err := WriteRecord(root, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	events, err := ReadHistory(root, rec.RoutingID)
	if err != nil || len(events) != 0 {
		t.Fatalf("missing history: %v %v", events, err)
	}

	if err := Append(root, Event{Event: EventDecided, DecidedBy: rec.DecidedBy, HumanApproved: true, Record: rec}); err != nil {
		t.Fatalf("Append decided: %v", err)
	}
	rec.Status = StatusClosed
	if err := Append(root, Event{Event: EventClosed, DecidedBy: rec.DecidedBy, HumanApproved: true, Record: rec}); err != nil {
		t.Fatalf("Append closed: %v", err)
	}

	events, err = ReadHistory(root, rec.RoutingID)
	if err != nil {
		t.Fatalf("ReadHistory: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Event != EventDecided || events[1].Event != EventClosed {
		t.Fatalf("order = %q,%q, want decided,closed", events[0].Event, events[1].Event)
	}
	if events[1].Record == nil || events[1].Record.Status != StatusClosed {
		t.Fatal("history snapshot did not carry the record state")
	}
	if events[0].Timestamp.IsZero() {
		t.Fatal("Append did not stamp a timestamp")
	}

	ids, err := ListHistoryIDs(root)
	if err != nil || len(ids) != 1 || ids[0] != rec.RoutingID {
		t.Fatalf("ListHistoryIDs = %v %v", ids, err)
	}
}

func TestHistoryRejectsReservedAndSnapshotlessEvents(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "main")
	if err := Append(root, Event{Event: EventReconciled, Record: rec}); err == nil {
		t.Fatal("Append accepted the v2-reserved reconciled event")
	}
	if err := Append(root, Event{Event: EventDecided}); err == nil {
		t.Fatal("Append accepted an event without a record snapshot")
	}
}

func TestDeleteRecordIsTheRollbackHalfOfASupersede(t *testing.T) {
	root := newRepo(t)
	rec := sampleRecord(t, "main")
	if err := WriteRecord(root, rec); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}
	if _, err := ReadRecord(RecordPath(root, rec)); err != nil {
		t.Fatalf("record should exist: %v", err)
	}
	if err := DeleteRecord(root, rec); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if _, err := os.Stat(RecordPath(root, rec)); !os.IsNotExist(err) {
		t.Fatalf("record still on disk after delete: %v", err)
	}
	// Deleting an already-absent record is idempotent: a rollback must not fail
	// because the first rollback already ran.
	if err := DeleteRecord(root, rec); err != nil {
		t.Fatalf("second DeleteRecord: %v", err)
	}
}
