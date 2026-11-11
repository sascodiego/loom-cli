package state

import (
	"fmt"
	"path/filepath"
)

// Candidate is one open routing record reported by detect. The field set is the
// minimum the harness needs to echo the record back to the user.
type Candidate struct {
	RoutingID string `json:"routing_id"`
	Slug      string `json:"slug"`
	Method    string `json:"method"`
	Tier      string `json:"tier"`
	DecidedAt string `json:"decided_at"`
}

// Verdict values are part of the CLI contract.
const (
	VerdictContinuation = "continuation"
	VerdictNew          = "new"
)

// DetectResult is the JSON document `router detect` prints.
type DetectResult struct {
	Candidates []Candidate `json:"candidates"`
	Verdict    string      `json:"verdict"`
	// Warnings surface malformed or foreign records without hiding the valid
	// ones. They are evidence for the user, never a repair path.
	Warnings []string `json:"warnings,omitempty"`
}

// Detect lists the open records belonging to branch. It reads the working tree
// only: no network, no git history, no reachability inference.
func Detect(root, branch string) (*DetectResult, error) {
	if branch == "" {
		return nil, fmt.Errorf("detect: missing branch")
	}
	found, err := ListOpen(root, branch)
	if err != nil {
		return nil, err
	}
	res := &DetectResult{Candidates: []Candidate{}, Verdict: VerdictNew}
	for _, item := range found {
		name := filepath.Base(item.Path)
		if item.Err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", name, item.Err))
			continue
		}
		rec := item.Record
		if !rec.MatchesBranch(branch) {
			// Same encoded directory, different true branch: an encoding
			// collision, not a candidate for this branch.
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s: belongs to branch %q, not %q", name, rec.Branch, branch))
			continue
		}
		if err := rec.Validate(); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if rec.Status != StatusOpen {
			continue
		}
		res.Candidates = append(res.Candidates, Candidate{
			RoutingID: rec.RoutingID,
			Slug:      rec.Slug,
			Method:    rec.Method,
			Tier:      rec.Tier,
			DecidedAt: rec.DecidedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	if len(res.Candidates) > 0 {
		res.Verdict = VerdictContinuation
	}
	return res, nil
}
