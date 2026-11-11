package preflight

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A declaration fixture with every required field, used as the base for the
// negative cases below.
const validDeclaration = `{
  "schema_version": 1,
  "request_summary": "The user asked to segment the endpoint by tenant tier.",
  "intent": "Change how quota is enforced for paid tenants.",
  "scope_summary": "Touches the quota middleware and its config.",
  "subsystems": ["router", "docs"],
  "scope_estimate": {"steps": 4, "sessions": 1}
}`

func newRepo(t *testing.T, files ...string) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q")
	for _, f := range files {
		path := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	run("add", "-A")
	return root
}

func TestParseAcceptsAValidDeclaration(t *testing.T) {
	d, err := Parse([]byte(validDeclaration))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", d.SchemaVersion, SchemaVersion)
	}
	if len(d.Subsystems) != 2 || d.ScopeEstimate.Steps != 4 || d.ScopeEstimate.Sessions != 1 {
		t.Fatalf("declaration = %+v", d)
	}
}

func TestParseRejectsOutputSlotsAndUnknownFields(t *testing.T) {
	// effort_tier and method are Clef outputs: they must never travel in.
	cases := map[string]string{
		"effort_tier": strings.Replace(validDeclaration, `"schema_version": 1,`,
			`"schema_version": 1, "effort_tier": "high",`, 1),
		"method": strings.Replace(validDeclaration, `"schema_version": 1,`,
			`"schema_version": 1, "method": "asdd",`, 1),
		"unknown": strings.Replace(validDeclaration, `"schema_version": 1,`,
			`"schema_version": 1, "rubric": [],`, 1),
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(doc)); err == nil ||
				!strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("Parse = %v, want unknown-field rejection", err)
			}
		})
	}
}

func TestParseRejectsEmptyProse(t *testing.T) {
	for _, field := range []string{"request_summary", "intent", "scope_summary"} {
		t.Run(field, func(t *testing.T) {
			empty := "   \n\t "
			doc := replaceJSONString(t, validDeclaration, field, empty)
			if _, err := Parse([]byte(doc)); err == nil ||
				!strings.Contains(err.Error(), field) {
				t.Fatalf("Parse = %v, want a %s error", err, field)
			}
		})
	}
}

func TestParseRejectsBadSchemaVersionAndScopeEstimate(t *testing.T) {
	tests := []struct{ name, doc string }{
		{"schema version", strings.Replace(validDeclaration, `"schema_version": 1`, `"schema_version": 2`, 1)},
		{"zero steps", strings.Replace(validDeclaration, `"steps": 4`, `"steps": 0`, 1)},
		{"negative sessions", strings.Replace(validDeclaration, `"sessions": 1`, `"sessions": -1`, 1)},
		{"malformed json", `{"schema_version": 1,`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Parse([]byte(tt.doc)); err == nil {
				t.Fatalf("Parse accepted %s", tt.name)
			}
		})
	}
}

func TestValidateSubsystemsAcceptsTrackedTopLevelDirs(t *testing.T) {
	root := newRepo(t, "router/main.go", "docs/a.md", "README.md")
	d, err := Parse([]byte(validDeclaration))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if err := d.Validate(root); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateSubsystemsRejectsUnknown(t *testing.T) {
	root := newRepo(t, "router/main.go", "docs/a.md")
	doc := strings.Replace(validDeclaration, `"subsystems": ["router", "docs"]`,
		`"subsystems": ["router", "billing"]`, 1)
	d, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	err = d.Validate(root)
	if err == nil || !strings.Contains(err.Error(), "billing") {
		t.Fatalf("Validate = %v, want the unknown subsystem named", err)
	}
}

func TestValidateAllowsAnEmptySubsystemList(t *testing.T) {
	root := newRepo(t, "router/main.go")
	doc := strings.Replace(validDeclaration, `"subsystems": ["router", "docs"]`, `"subsystems": []`, 1)
	d, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// A root-file-only change declares no subsystem; that is not an error, it
	// only means Clef receives less breadth signal.
	if err := d.Validate(root); err != nil {
		t.Fatalf("Validate = %v, want nil for an empty declaration", err)
	}
}

func TestTopLevelDirsIgnoresRootFiles(t *testing.T) {
	root := newRepo(t, "router/main.go", "docs/a.md", "README.md", "odd/tasks/x.md")
	dirs, err := TopLevelDirs(root)
	if err != nil {
		t.Fatalf("TopLevelDirs: %v", err)
	}
	want := []string{"docs", "odd", "router"}
	if strings.Join(dirs, ",") != strings.Join(want, ",") {
		t.Fatalf("dirs = %v, want %v", dirs, want)
	}
}

// replaceJSONString rewrites one top-level string field, keeping the fixture
// readable instead of hand-editing escaped JSON per case.
func replaceJSONString(t *testing.T, doc, field, value string) string {
	t.Helper()
	marker := `"` + field + `": "`
	start := strings.Index(doc, marker)
	if start < 0 {
		t.Fatalf("fixture has no field %q", field)
	}
	rest := doc[start+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("fixture field %q is not a string", field)
	}
	quoted, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("quote %q: %v", field, err)
	}
	return doc[:start+len(marker)] + string(quoted[1:len(quoted)-1]) + rest[end:]
}
