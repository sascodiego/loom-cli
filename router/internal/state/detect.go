package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Config is the per-project router configuration. Credentials never live here:
// they come from the environment.
type Config struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
}

// Environment variables the Cloudflare provider reads at call time.
const (
	EnvCloudflareToken   = "CLOUDFLARE_API_TOKEN"
	EnvCloudflareAccount = "CLOUDFLARE_ACCOUNT_ID"
)

// ConfigPath returns .router/config.json for a repository root.
func ConfigPath(root string) string {
	return filepath.Join(RouterDir(root), "config.json")
}

// LoadConfig reads and validates .router/config.json. A missing file is a
// validation error that names the path, because decide cannot run without it.
func LoadConfig(root string) (*Config, error) {
	path := ConfigPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no router config at %s (write %s)", path, filepath.Join(".router", "config.json"))
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate requires every field the provider needs.
func (c *Config) Validate() error {
	switch {
	case c.Provider == "":
		return fmt.Errorf("config: missing provider")
	case c.Model == "":
		return fmt.Errorf("config: missing model")
	case c.Endpoint == "":
		return fmt.Errorf("config: missing endpoint")
	}
	if strings.Contains(c.Endpoint, "\n") || strings.Contains(c.Endpoint, " ") {
		return fmt.Errorf("config: endpoint must be a single URL")
	}
	return nil
}
