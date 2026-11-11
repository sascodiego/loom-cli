// Package preflight owns the declared pre-flight: the task definition a person
// and their agent write before classification, plus the assembly of the payload
// sent to Clef.
//
// It reads no diffs, calls no model and touches no network. The facts that exist
// at decision time are the declaration and the repository structure; nothing is
// measured that does not exist.
package preflight

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaVersion is the declaration schema this binary accepts.
const SchemaVersion = 1

// ScopeEstimate is the declarer's own size guess. It is input to Clef, never a
// substitute for the classifier's effort_tier.
type ScopeEstimate struct {
	Steps    int `json:"steps"`
	Sessions int `json:"sessions"`
}

// Declaration is the pre-flight document. Output slots (effort_tier, method)
// are deliberately absent: they are Clef's answer, never its question.
type Declaration struct {
	SchemaVersion  int           `json:"schema_version"`
	RequestSummary string        `json:"request_summary"`
	Intent         string        `json:"intent"`
	ScopeSummary   string        `json:"scope_summary"`
	Subsystems     []string      `json:"subsystems"`
	ScopeEstimate  ScopeEstimate `json:"scope_estimate"`
}

// Parse decodes and field-validates a declaration. Unknown fields are rejected,
// so a caller cannot smuggle a classification answer into the input.
func Parse(data []byte) (*Declaration, error) {
	var d Declaration
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("decode declaration: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("decode declaration: trailing content after the JSON document")
	}
	if err := d.ValidateFields(); err != nil {
		return nil, err
	}
	return &d, nil
}

// ValidateFields checks the schema version, the required prose and the scope
// estimate. Prose must carry text, not whitespace.
func (d *Declaration) ValidateFields() error {
	switch {
	case d.SchemaVersion == 0:
		return fmt.Errorf("missing schema_version")
	case d.SchemaVersion != SchemaVersion:
		return fmt.Errorf("unknown schema_version %d (want %d)", d.SchemaVersion, SchemaVersion)
	}
	for name, value := range map[string]string{
		"request_summary": d.RequestSummary,
		"intent":          d.Intent,
		"scope_summary":   d.ScopeSummary,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("missing %s: the declaration needs real prose, not whitespace", name)
		}
	}
	switch {
	case d.ScopeEstimate.Steps < 1:
		return fmt.Errorf("scope_estimate.steps must be at least 1")
	case d.ScopeEstimate.Sessions < 1:
		return fmt.Errorf("scope_estimate.sessions must be at least 1")
	}
	return nil
}
