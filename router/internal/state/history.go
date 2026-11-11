package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Event names written to the history. The v1 writer emits every one of these
// except EventReconciled, which is reserved for the v2 reconciliation engine.
const (
	EventDecided          = "decided"
	EventGateOverridden   = "gate_overridden"
	EventJointFallback    = "joint_fallback"
	EventReRouteRequested = "re_route_requested"
	EventSuperseded       = "superseded"
	EventClosed           = "closed"
)

// EventReconciled is v2-reserved: no v1 writer emits it, and readers must
// tolerate its absence until the reconciliation engine exists.
const EventReconciled = "reconciled"

// Event is one append-only history line. It carries the full record snapshot so
// the audit trail stays readable after the open record's status changes.
type Event struct {
	Timestamp     time.Time `json:"timestamp"`
	Event         string    `json:"event"`
	DecidedBy     DecidedBy `json:"decided_by"`
	HumanApproved bool      `json:"human_approved"`
	UserChoice    string    `json:"user_choice,omitempty"`
	RequestedBy   string    `json:"requested_by,omitempty"`
	Justification string    `json:"justification,omitempty"`
	Record        *Record   `json:"record"`
}

// Append writes one event as a single JSONL line. It is the only write path
// into .router/history/.
func Append(root string, ev Event) error {
	if ev.Event == "" {
		return fmt.Errorf("history event: missing event name")
	}
	if ev.Event == EventReconciled {
		return fmt.Errorf("history event: %q is v2-reserved and has no v1 writer", EventReconciled)
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Record == nil {
		return fmt.Errorf("history event %q: missing record snapshot", ev.Event)
	}
	dir := HistoryDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("encode history event: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(HistoryPath(root, ev.Record.RoutingID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("append history event: %w", err)
	}
	return f.Sync()
}

// ReadHistory returns the events recorded for one routing id, oldest first. A
// missing history file is an empty history, not an error.
func ReadHistory(root, routingID string) ([]Event, error) {
	path := HistoryPath(root, routingID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	var events []Event
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			return events, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
		}
		events = append(events, ev)
	}
	return events, nil
}

// ListHistoryIDs returns every routing id with a history file, in stable order.
func ListHistoryIDs(root string) ([]string, error) {
	entries, err := os.ReadDir(HistoryDir(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history dir: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) <= len(".jsonl") || name[len(name)-len(".jsonl"):] != ".jsonl" {
			continue
		}
		ids = append(ids, name[:len(name)-len(".jsonl")])
	}
	return ids, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
