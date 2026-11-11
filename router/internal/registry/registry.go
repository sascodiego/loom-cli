// Package registry owns the canonical method registry and the criteria files it
// points at: the closed set of methods the router may dispatch to.
package registry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SchemaVersion is the registry schema this binary accepts.
const SchemaVersion = 1

// MaxCriteriaBytes is the hard bound on one CRITERIA.md, enforced both here and
// at decide time.
const MaxCriteriaBytes = 2048

// Method is one registry entry, fully enumerated: no field is optional and no
// value is inferred.
type Method struct {
	Name      string `json:"name"`
	Criteria  string `json:"criteria"`
	Skill     string `json:"skill"`
	Namespace string `json:"namespace"`
	Available bool   `json:"available"`
}

// Registry is the whole method set, in the order that pins criteria_hash.
type Registry struct {
	SchemaVersion int      `json:"schema_version"`
	Methods       []Method `json:"methods"`
}

// Path returns router/registry.json for a repository root.
func Path(root string) string { return filepath.Join(root, "router", "registry.json") }

// Load reads and shape-checks the registry. Criteria files are not opened here;
// Validate does that so loading stays cheap and side-effect free.
func Load(root string) (*Registry, error) {
	data, err := os.ReadFile(Path(root))
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var r Registry
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode registry: %w", err)
	}
	if err := r.validateShape(); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Registry) validateShape() error {
	switch {
	case r.SchemaVersion == 0:
		return fmt.Errorf("registry: missing schema_version")
	case r.SchemaVersion != SchemaVersion:
		return fmt.Errorf("registry: unknown schema_version %d (want %d)", r.SchemaVersion, SchemaVersion)
	case len(r.Methods) == 0:
		return fmt.Errorf("registry: no methods declared")
	}
	seen := make(map[string]bool, len(r.Methods))
	for i, m := range r.Methods {
		switch {
		case m.Name == "":
			return fmt.Errorf("registry: method %d has no name", i)
		case seen[m.Name]:
			return fmt.Errorf("registry: duplicate method %q", m.Name)
		case m.Criteria == "":
			return fmt.Errorf("registry: method %q has no criteria path", m.Name)
		case m.Skill == "":
			return fmt.Errorf("registry: method %q has no skill", m.Name)
		case m.Namespace == "":
			return fmt.Errorf("registry: method %q has no namespace", m.Name)
		}
		seen[m.Name] = true
	}
	return nil
}

// Validate checks that every criteria file exists and respects the size bound.
// The registry is unusable if a criteria file is missing: decide would send an
// empty option description and the classification would be blind.
func (r *Registry) Validate(root string) error {
	if err := r.validateShape(); err != nil {
		return err
	}
	for _, m := range r.Methods {
		info, err := os.Stat(filepath.Join(root, m.Criteria))
		if err != nil {
			return fmt.Errorf("registry: method %q criteria %s: %w", m.Name, m.Criteria, err)
		}
		if info.Size() > MaxCriteriaBytes {
			return fmt.Errorf("registry: method %q criteria %s is %d bytes, over the %d bound",
				m.Name, m.Criteria, info.Size(), MaxCriteriaBytes)
		}
	}
	return nil
}

// Names returns the method names in registry order.
func (r *Registry) Names() []string {
	names := make([]string, len(r.Methods))
	for i, m := range r.Methods {
		names[i] = m.Name
	}
	return names
}

// Method looks one method up by name.
func (r *Registry) Method(name string) (Method, bool) {
	for _, m := range r.Methods {
		if m.Name == name {
			return m, true
		}
	}
	return Method{}, false
}

// Available returns the methods v1 may dispatch to, in registry order.
func (r *Registry) Available() []Method {
	out := make([]Method, 0, len(r.Methods))
	for _, m := range r.Methods {
		if m.Available {
			out = append(out, m)
		}
	}
	return out
}

// CriteriaContent reads one method's criteria verbatim. The content is what the
// endpoint receives, so it is never transformed here.
func (r *Registry) CriteriaContent(root, name string) ([]byte, error) {
	m, ok := r.Method(name)
	if !ok {
		return nil, fmt.Errorf("registry: unknown method %q", name)
	}
	data, err := os.ReadFile(filepath.Join(root, m.Criteria))
	if err != nil {
		return nil, fmt.Errorf("registry: method %q criteria: %w", name, err)
	}
	if len(data) > MaxCriteriaBytes {
		return nil, fmt.Errorf("registry: method %q criteria is %d bytes, over the %d bound",
			name, len(data), MaxCriteriaBytes)
	}
	return data, nil
}

// CriteriaHash is SHA-256 over the concatenation of every criteria file in
// registry order — including methods marked unavailable, so the pin covers the
// whole set rather than only the dispatchable subset. It changes whenever the
// classifier's criteria material changes, which is exactly what the delegation
// criterion needs to detect.
func (r *Registry) CriteriaHash(root string) (string, error) {
	h := sha256.New()
	for _, m := range r.Methods {
		data, err := r.CriteriaContent(root, m.Name)
		if err != nil {
			return "", err
		}
		if _, err := h.Write(data); err != nil {
			return "", fmt.Errorf("hash criteria %q: %w", m.Name, err)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Describe renders the registry for human output on stderr.
func (r *Registry) Describe() string {
	lines := make([]string, 0, len(r.Methods)+1)
	for _, m := range r.Methods {
		state := "available"
		if !m.Available {
			state = "unavailable"
		}
		lines = append(lines, fmt.Sprintf("  %-15s %-12s namespace=%s", m.Name, state, m.Namespace))
	}
	return "methods:\n" + strings.Join(lines, "\n")
}
