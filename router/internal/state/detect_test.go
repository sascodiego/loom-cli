package state

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func newRepoForDetect(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root
}

func TestDetectEmptyTreeIsNew(t *testing.T) {
	root := newRepoForDetect(t)
	res, err := Detect(root, "main")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if res.Verdict != VerdictNew {
		t.Fatalf("verdict = %q, want %q", res.Verdict, VerdictNew)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates = %v, want none", res.Candidates)
	}
}

func TestDetectListsOpenRecordsOnly(t *testing.T) {
	root := newRepoForDetect(t)
	open := sampleRecord(t, "feat/x")
	closed := sampleRecord(t, "feat/x")
	closed.Status = StatusClosed
	for _, r := range []*Record{open, closed} {
		if err := WriteRecord(root, r); err != nil {
			t.Fatalf("WriteRecord: %v", err)
		}
	}
	res, err := Detect(root, "feat/x")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if res.Verdict != VerdictContinuation {
		t.Fatalf("verdict = %q, want %q", res.Verdict, VerdictContinuation)
	}
	if len(res.Candidates) != 1 || res.Candidates[0].RoutingID != open.RoutingID {
		t.Fatalf("candidates = %+v, want only %s", res.Candidates, open.RoutingID)
	}
	if res.Candidates[0].Slug != open.Slug || res.Candidates[0].Method != open.Method {
		t.Fatalf("candidate shape = %+v", res.Candidates[0])
	}
}

func TestDetectMultipleCandidatesAllListed(t *testing.T) {
	root := newRepoForDetect(t)
	a := sampleRecord(t, "main")
	b := sampleRecord(t, "main")
	for _, r := range []*Record{a, b} {
		if err := WriteRecord(root, r); err != nil {
			t.Fatalf("WriteRecord: %v", err)
		}
	}
	res, err := Detect(root, "main")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("got %d candidates, want 2 (Ruling 8: list all, human decides)", len(res.Candidates))
	}
}

func TestDetectWarnsOnMalformedAndForeignRecords(t *testing.T) {
	root := newRepoForDetect(t)

	// An encoding collision: a record for fix/339 lives in fix--339/, which is
	// also the directory for branch fix--339. The true branch inside the record
	// decides membership, so this must not become a candidate.
	foreign := sampleRecord(t, "fix/339")
	if err := WriteRecord(root, foreign); err != nil {
		t.Fatalf("WriteRecord: %v", err)
	}

	res, err := Detect(root, "fix--339")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates = %+v, want none: the record belongs to %q", res.Candidates, "fix/339")
	}
	if len(res.Warnings) == 0 || !strings.Contains(res.Warnings[0], "belongs to branch") {
		t.Fatalf("warnings = %v, want a branch-mismatch warning", res.Warnings)
	}

	// A record with an unknown schema version is reported, never accepted.
	stale := sampleRecord(t, "fix--339")
	path := RecordPath(root, stale)
	data := []byte(`{"schema_version":99,"routing_id":"x","brief":"","request_id":"r"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}
	res, err = Detect(root, "fix--339")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if res.Verdict != VerdictNew || len(res.Candidates) != 0 {
		t.Fatalf("detect = %+v, want new with no candidates", res)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("warnings = %v, want one foreign and one malformed", res.Warnings)
	}
}
