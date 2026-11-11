package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// StateSchemaVersion is the schema of the state document sent to the endpoint.
const StateSchemaVersion = 1

// SessionFacts are the observable facts that exist at decision time. Nothing
// here is derived from a diff: a diff usually does not exist yet when the
// routing decision is taken.
type SessionFacts struct {
	// Branch is the checked-out branch, the strongest continuation signal.
	Branch string `json:"branch"`
	// ModifiedFiles is the count of modified tracked files, a cleanliness
	// summary only. Paths are deliberately not sent.
	ModifiedFiles int `json:"modified_files"`
}

// Criterion is one method's criteria content, in registry order.
type Criterion struct {
	Method  string `json:"method"`
	Content string `json:"content"`
}

// StateDoc is the state half of the request.
type StateDoc struct {
	SchemaVersion int             `json:"schema_version"`
	Declaration   json.RawMessage `json:"declaration"`
	Session       SessionFacts    `json:"session"`
	Criteria      []Criterion     `json:"criteria"`
}

// Effort tier option descriptions. The method question takes its descriptions
// verbatim from CRITERIA.md; the tier question has no criteria file, so its
// wording lives here and stays on the complexity axis only.
var tierOptions = []Option{
	{Value: "low", Description: "one bounded change: one subsystem, one session, a handful of steps, no written specification needed"},
	{Value: "medium", Description: "several steps or files in one subsystem, still one session, still a reviewable diff"},
	{Value: "high", Description: "several subsystems, more than one session, or work that must follow a written artifact chain"},
}

// BuildState assembles the declared pre-flight, the session facts and the
// criteria excerpts. It validates that the declaration is present and is JSON,
// because an unparseable state would make the classification meaningless.
func BuildState(declaration json.RawMessage, facts SessionFacts, criteria []Criterion) (json.RawMessage, error) {
	if len(bytes.TrimSpace(declaration)) == 0 {
		return nil, fmt.Errorf("build state: missing declaration")
	}
	if !json.Valid(declaration) {
		return nil, fmt.Errorf("build state: declaration is not valid JSON")
	}
	if facts.Branch == "" {
		return nil, fmt.Errorf("build state: missing branch")
	}
	if criteria == nil {
		criteria = []Criterion{}
	}
	doc := StateDoc{
		SchemaVersion: StateSchemaVersion,
		Declaration:   declaration,
		Session:       facts,
		Criteria:      criteria,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("build state: %w", err)
	}
	return data, nil
}

// BuildQuestions returns the typed questions: effort_tier always, method only
// for the methods that are dispatchable. An unavailable method contributes no
// option, so the endpoint cannot suggest it.
func BuildQuestions(criteria []Criterion) []Question {
	questions := []Question{{Key: "effort_tier", Options: tierOptions}}
	if len(criteria) == 0 {
		return questions
	}
	options := make([]Option, 0, len(criteria))
	for _, c := range criteria {
		if strings.TrimSpace(c.Method) == "" || strings.TrimSpace(c.Content) == "" {
			continue
		}
		options = append(options, Option{Value: c.Method, Description: c.Content})
	}
	if len(options) == 0 {
		return questions
	}
	return append(questions, Question{Key: "method", Options: options})
}
