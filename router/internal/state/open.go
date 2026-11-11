// Package state owns the on-disk routing state: the open records under
// .router/open/<branch-encoded>/<routing_id>.json and the append-only history
// under .router/history/<routing_id>.jsonl.
//
// Layout rule (design S14): every writer path funnels through this package and
// appends to the history file; no other component writes .router/.
package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SchemaVersion is the record schema this binary writes and the only one it
// validates as current.
const SchemaVersion = 1

// Status is the lifecycle of one routing record.
type Status string

const (
	StatusOpen       Status = "open"
	StatusClosed     Status = "closed"
	StatusSuperseded Status = "superseded"
)

// DecidedBy distinguishes how a routing decision came about. It is the
// stratification key for the phase-1 agreement rate, so the set is closed.
type DecidedBy string

const (
	// DecidedByClef is the classifier's own suggestion, accepted by the user.
	DecidedByClef DecidedBy = "clef"
	// DecidedByUserGateOverride is the user overriding a Clef suggestion at
	// the confirmation gate.
	DecidedByUserGateOverride DecidedBy = "user_gate_override"
	// DecidedByJointFallback is a decision made by the user and agent together
	// after a provider failure.
	DecidedByJointFallback DecidedBy = "joint_fallback"
	// DecidedByOverrideReRoute is an agent-initiated re-route the user approved.
	DecidedByOverrideReRoute DecidedBy = "override_re_route"
)

// Valid reports whether the value is one of the four declared origins.
func (d DecidedBy) Valid() bool {
	switch d {
	case DecidedByClef, DecidedByUserGateOverride, DecidedByJointFallback, DecidedByOverrideReRoute:
		return true
	}
	return false
}

// Record is one routing decision, as persisted under .router/open/. The field
// set is the v1 schema; unknown fields are rejected on read so a stale binary
// never silently reinterprets a newer record.
type Record struct {
	SchemaVersion   int       `json:"schema_version"`
	RoutingID       string    `json:"routing_id"`
	Slug            string    `json:"slug"`
	Branch          string    `json:"branch"`
	BranchEncoded   string    `json:"branch_encoded"`
	Method          string    `json:"method"`
	Tier            string    `json:"tier"`
	DecidedAt       time.Time `json:"decided_at"`
	RequestID       string    `json:"request_id"`
	Provider        string    `json:"provider"`
	Model           string    `json:"model"`
	CriteriaHash    string    `json:"criteria_hash"`
	DecidedBy       DecidedBy `json:"decided_by"`
	HumanApproved   bool      `json:"human_approved"`
	SuggestedMethod string    `json:"suggested_method"`
	SuggestedTier   string    `json:"suggested_tier"`
	UserChoice      string    `json:"user_choice,omitempty"`
	PayloadHash     string    `json:"payload_hash"`
	Status          Status    `json:"status"`
	SupersededBy    string    `json:"superseded_by,omitempty"`
	RequestedBy     string    `json:"requested_by,omitempty"`
}

// Validate checks the invariants detect relies on. It is intentionally strict:
// a record that fails here is reported, never repaired.
func (r *Record) Validate() error {
	switch {
	case r.SchemaVersion == 0:
		return fmt.Errorf("missing schema_version")
	case r.SchemaVersion != SchemaVersion:
		return fmt.Errorf("unknown schema_version %d (want %d)", r.SchemaVersion, SchemaVersion)
	case r.RoutingID == "":
		return fmt.Errorf("missing routing_id")
	case r.RequestID == "":
		return fmt.Errorf("missing request_id")
	case r.Slug == "":
		return fmt.Errorf("missing slug")
	case r.Branch == "":
		return fmt.Errorf("missing branch")
	case r.BranchEncoded == "":
		return fmt.Errorf("missing branch_encoded")
	case r.Method == "":
		return fmt.Errorf("missing method")
	case r.Tier == "":
		return fmt.Errorf("missing tier")
	case r.DecidedAt.IsZero():
		return fmt.Errorf("missing decided_at")
	case !r.DecidedBy.Valid():
		return fmt.Errorf("invalid decided_by %q", r.DecidedBy)
	case r.Status != StatusOpen && r.Status != StatusClosed && r.Status != StatusSuperseded:
		return fmt.Errorf("invalid status %q", r.Status)
	}
	return nil
}

// EncodeBranch turns a branch name into its filesystem path component. '/' maps
// to "--" (fix/339-cedula becomes fix--339-cedula).
//
// The encoding is not injective on its own: a branch whose own name contains
// "--" can encode to the same component. Losslessness therefore comes from the
// record, which stores the true Branch, and readers must compare it (see
// MatchesBranch) instead of trusting the directory name alone.
func EncodeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "--")
}

// MatchesBranch reports whether a record belongs to branch, guarding against
// encoded-name collisions.
func (r *Record) MatchesBranch(branch string) bool {
	return r.Branch == branch
}

// RouterDir returns the tracked state directory for a repository root.
func RouterDir(root string) string { return filepath.Join(root, ".router") }

// OpenDir returns .router/open/<branch-encoded> for a branch.
func OpenDir(root, branch string) string {
	return filepath.Join(RouterDir(root), "open", EncodeBranch(branch))
}

// HistoryDir returns .router/history.
func HistoryDir(root string) string { return filepath.Join(RouterDir(root), "history") }

// RecordPath returns the canonical path for a record.
func RecordPath(root string, r *Record) string {
	return filepath.Join(OpenDir(root, r.Branch), r.RoutingID+".json")
}

// HistoryPath returns the canonical history file for a routing id.
func HistoryPath(root, routingID string) string {
	return filepath.Join(HistoryDir(root), routingID+".jsonl")
}

// NewRoutingID mints a UUIDv4 using crypto/rand, so routing ids never collide
// across sessions or machines.
func NewRoutingID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("mint routing_id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	dst := make([]byte, 36)
	hex.Encode(dst[0:8], b[0:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], b[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], b[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], b[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:36], b[10:16])
	return string(dst), nil
}

// WriteRecord writes a record atomically (temp file plus rename) inside its
// branch directory, creating the directory when missing.
func WriteRecord(root string, r *Record) error {
	if err := r.Validate(); err != nil {
		return fmt.Errorf("refusing to write invalid record: %w", err)
	}
	path := RecordPath(root, r)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create open dir: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encode record: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data)
}

// ReadRecord decodes one record and rejects unknown fields.
func ReadRecord(path string) (*Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Record
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return &r, nil
}

// OpenRecord is one record found on disk, together with the parse error when it
// could not be read. Detect reports both: a malformed file is evidence, not a
// reason to hide the rest.
type OpenRecord struct {
	Path   string
	Record *Record
	Err    error
}

// ListOpen returns every record file under .router/open/<branch-encoded>/,
// including malformed ones (with their error) so callers can report them. A
// missing directory is not an error: it means nothing is open.
func ListOpen(root, branch string) ([]OpenRecord, error) {
	dir := OpenDir(root, branch)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	out := make([]OpenRecord, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		rec, err := ReadRecord(path)
		out = append(out, OpenRecord{Path: path, Record: rec, Err: err})
	}
	return out, nil
}

// FindRecord locates one record by routing id anywhere under .router/open/.
func FindRecord(root, routingID string) (*Record, string, error) {
	if routingID == "" {
		return nil, "", fmt.Errorf("missing routing_id")
	}
	openRoot := filepath.Join(RouterDir(root), "open")
	var found string
	err := filepath.WalkDir(openRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == routingID+".json" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("scan open records: %w", err)
	}
	if found == "" {
		return nil, "", fmt.Errorf("no open record with routing_id %s", routingID)
	}
	rec, err := ReadRecord(found)
	if err != nil {
		return nil, "", err
	}
	return rec, found, nil
}

// writeFileAtomic writes data to path via a temp file in the same directory so
// a crash never leaves a half-written record.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
