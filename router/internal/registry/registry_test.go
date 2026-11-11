package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is three levels up from router/internal/registry: the shipped
// registry and criteria files are tested as they ship, not as fixtures.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return root
}

func loadShipped(t *testing.T) *Registry {
	t.Helper()
	r, err := Load(repoRoot(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return r
}

func TestShippedRegistryEnumeratesTheFiveMethodsInOrder(t *testing.T) {
	r := loadShipped(t)
	want := []string{"odd", "ftd", "asdd", "gentle-sdd", "egm-review-loop"}
	got := r.Names()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("methods = %v, want %v", got, want)
	}
	m, ok := r.Method("gentle-sdd")
	if !ok {
		t.Fatal("gentle-sdd missing from the registry")
	}
	if m.Namespace != "sdd" || m.Criteria != "router/criteria/gentle-sdd/CRITERIA.md" {
		t.Fatalf("gentle-sdd entry = %+v", m)
	}
}

func TestAvailabilityGating(t *testing.T) {
	r := loadShipped(t)
	avail := r.Available()
	if len(avail) != 4 {
		t.Fatalf("available = %d methods, want 4", len(avail))
	}
	for _, m := range avail {
		if m.Name == "egm-review-loop" {
			t.Fatal("egm-review-loop must not be available in v1")
		}
	}
	if m, _ := r.Method("egm-review-loop"); m.Available {
		t.Fatal("egm-review-loop is marked available")
	}
}

func TestValidateChecksCriteriaFilesAndSizeBound(t *testing.T) {
	r := loadShipped(t)
	if err := r.Validate(repoRoot(t)); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// A criteria file over the 2 KB hard bound is refused.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "router", "criteria", "odd"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	big := strings.Repeat("x", MaxCriteriaBytes+1)
	if err := os.WriteFile(filepath.Join(root, "router", "criteria", "odd", "CRITERIA.md"), []byte(big), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	one := `{"schema_version":1,"methods":[{"name":"odd","criteria":"router/criteria/odd/CRITERIA.md","skill":"harness","namespace":"odd","available":true}]}`
	if err := os.MkdirAll(filepath.Join(root, "router"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "router", "registry.json"), []byte(one), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	r, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	err = r.Validate(root)
	if err == nil || !strings.Contains(err.Error(), "2048") {
		t.Fatalf("Validate = %v, want a size-bound error naming 2048", err)
	}
}

func TestCriteriaHashIsStableAndCoversUnavailableMethods(t *testing.T) {
	r := loadShipped(t)
	root := repoRoot(t)

	h1, err := r.CriteriaHash(root)
	if err != nil {
		t.Fatalf("CriteriaHash: %v", err)
	}
	if len(h1) != 64 {
		t.Fatalf("hash = %q, want 64 hex chars", h1)
	}
	h2, err := r.CriteriaHash(root)
	if err != nil {
		t.Fatalf("CriteriaHash: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hash is not stable: %q vs %q", h1, h2)
	}

	// Mutating the criteria of the unavailable method must change the hash: the
	// pin covers all five files, not only the four dispatchable ones.
	r2 := loadShipped(t)
	for i := range r2.Methods {
		if r2.Methods[i].Name == "egm-review-loop" {
			r2.Methods[i].Criteria = filepath.Join("router", "criteria", "odd", "CRITERIA.md")
		}
	}
	h3, err := r2.CriteriaHash(root)
	if err != nil {
		t.Fatalf("CriteriaHash: %v", err)
	}
	if h3 == h1 {
		t.Fatal("hash did not change with the criteria assignment: the pin does not cover every method")
	}
}

func TestCriteriaContentIsVerbatim(t *testing.T) {
	r := loadShipped(t)
	root := repoRoot(t)
	content, err := r.CriteriaContent(root, "asdd")
	if err != nil {
		t.Fatalf("CriteriaContent: %v", err)
	}
	disk, err := os.ReadFile(filepath.Join(root, "router", "criteria", "asdd", "CRITERIA.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != string(disk) {
		t.Fatal("criteria content is not byte-identical to the file")
	}
	if !strings.Contains(string(content), "does NOT serve") {
		t.Fatal("criteria must carry the NOT-statements section")
	}
	for _, name := range []string{"asdd", "gentle-sdd"} {
		c, err := r.CriteriaContent(root, name)
		if err != nil {
			t.Fatalf("CriteriaContent(%s): %v", name, err)
		}
		if !strings.Contains(string(c), "Mutual exclusion") {
			t.Fatalf("%s criteria has no mutual-exclusion section", name)
		}
	}
}

func TestLoadRejectsUnknownFieldsAndBadShape(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "router"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cases := map[string]string{
		"unknown field": `{"schema_version":1,"methods":[],"rubric":true}`,
		"bad version":   `{"schema_version":2,"methods":[]}`,
		"no methods":    `{"schema_version":1,"methods":[]}`,
		"empty name":    `{"schema_version":1,"methods":[{"name":"","criteria":"c","skill":"s","namespace":"n","available":true}]}`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(root, "router", "registry.json"), []byte(doc), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if _, err := Load(root); err == nil {
				t.Fatalf("Load accepted %s", name)
			}
		})
	}
}
