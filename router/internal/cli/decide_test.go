package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"loom/router/internal/provider"
	"loom/router/internal/state"
)

const decideDeclaration = `{
  "schema_version": 1,
  "request_summary": "The user asked to segment the quota by tenant tier.",
  "intent": "Change how quota is enforced for paid tenants.",
  "scope_summary": "Touches the quota middleware and its config.",
  "subsystems": ["router"],
  "scope_estimate": {"steps": 4, "sessions": 1}
}`

// fakeProvider returns a canned answer and records what it received.
type fakeProvider struct {
	answer      map[string]map[string]float64
	err         error
	gotState    json.RawMessage
	gotQuests   []provider.Question
	answerCalls int
}

func (f *fakeProvider) Decide(ctx context.Context, state json.RawMessage, questions []provider.Question) (*provider.Decision, error) {
	f.answerCalls++
	f.gotState = state
	f.gotQuests = questions
	if f.err != nil {
		return nil, f.err
	}
	return &provider.Decision{RequestID: "trace-42", Probabilities: f.answer, Attempts: 1}, nil
}

// routerRepo builds a temp repository holding the registry, three criteria files
// and the config decide needs.
func routerRepo(t *testing.T) string {
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

	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("router/registry.json", `{
  "schema_version": 1,
  "methods": [
    {"name":"odd","criteria":"router/criteria/odd/CRITERIA.md","skill":"harness","namespace":"odd","available":true},
    {"name":"ftd","criteria":"router/criteria/ftd/CRITERIA.md","skill":"ftd","namespace":"ftd","available":true},
    {"name":"egm-review-loop","criteria":"router/criteria/egm-review-loop/CRITERIA.md","skill":"egm","namespace":"egm","available":false}
  ]
}`)
	write("router/criteria/odd/CRITERIA.md", "odd criteria\n")
	write("router/criteria/ftd/CRITERIA.md", "ftd criteria\n")
	write("router/criteria/egm-review-loop/CRITERIA.md", "egm criteria\n")
	write(".router/config.json", `{"provider":"cloudflare","model":"@cf/cloudflare/clef","endpoint":"https://example.invalid/run"}`)
	run("add", "-A")
	return root
}

func withFakeProvider(t *testing.T, root string, fake *fakeProvider) {
	t.Helper()
	prev := providerFactory
	providerFactory = func(cfg *state.Config) (provider.Provider, error) { return fake, nil }
	t.Cleanup(func() { providerFactory = prev })
}

func runDecide(t *testing.T, root, declaration string) (map[string]any, error, string) {
	t.Helper()
	env, out, errb := newEnv(t, root, declaration)
	err := Handle(context.Background(), env, "decide", []string{"--input", "-"})
	var got map[string]any
	if err == nil {
		if jerr := json.Unmarshal(out.Bytes(), &got); jerr != nil {
			t.Fatalf("decide stdout is not JSON: %v (%q)", jerr, out.String())
		}
	}
	return got, err, errb.String()
}

func TestDecideHappyPath(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.75, "high": 0.15},
		"method":      {"odd": 0.8, "ftd": 0.15, "egm-review-loop": 0.05},
	}}
	withFakeProvider(t, root, fake)

	got, err, _ := runDecide(t, root, decideDeclaration)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if got["tier"] != "medium" {
		t.Fatalf("tier = %v, want medium (argmax)", got["tier"])
	}
	if got["method"] != "odd" {
		t.Fatalf("method = %v, want odd", got["method"])
	}
	if got["request_id"] != "trace-42" {
		t.Fatalf("request_id = %v, want the provider trace id", got["request_id"])
	}
	if got["provider"] != "cloudflare" || got["model"] != "@cf/cloudflare/clef" {
		t.Fatalf("provider/model = %v/%v", got["provider"], got["model"])
	}
	if got["ambiguous"] != false {
		t.Fatalf("ambiguous = %v, want false", got["ambiguous"])
	}
	if _, present := got["substituted"]; present {
		t.Fatalf("substituted present on a clean decision: %v", got["substituted"])
	}
	if got["payload_hash"] == nil || got["payload_hash"] == "" {
		t.Fatal("decide must report the payload hash record needs")
	}
	sum := sha256.Sum256([]byte(decideDeclaration))
	if got["payload_hash"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("payload_hash = %v, want SHA-256 of the exact declaration", got["payload_hash"])
	}

	// criteria_hash covers the three fixture criteria in registry order.
	h := sha256.New()
	for _, part := range []string{"odd criteria\n", "ftd criteria\n", "egm criteria\n"} {
		h.Write([]byte(part))
	}
	if got["criteria_hash"] != hex.EncodeToString(h.Sum(nil)) {
		t.Fatalf("criteria_hash = %v, want the registry-order hash", got["criteria_hash"])
	}

	// The unavailable method never reaches the question's option list…
	for _, q := range fake.gotQuests {
		if q.Key != "method" {
			continue
		}
		for _, o := range q.Options {
			if o.Value == "egm-review-loop" {
				t.Fatal("unavailable method was offered as an option")
			}
		}
	}
	// …and the state carries the declaration plus session facts.
	if !strings.Contains(string(fake.gotState), `"branch"`) ||
		!strings.Contains(string(fake.gotState), `"modified_files"`) {
		t.Fatalf("state missing session facts: %s", fake.gotState)
	}
}

func TestDecideMarksAmbiguityWithinFivePoints(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.2, "medium": 0.4, "high": 0.4},
		"method":      {"odd": 0.50, "ftd": 0.47, "egm-review-loop": 0.03},
	}}
	withFakeProvider(t, root, fake)

	got, err, _ := runDecide(t, root, decideDeclaration)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if got["ambiguous"] != true {
		t.Fatalf("ambiguous = %v, want true for a 0.03 gap", got["ambiguous"])
	}
}

func TestDecideSubstitutesAnUnavailableWinner(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{answer: map[string]map[string]float64{
		"effort_tier": {"low": 0.1, "medium": 0.2, "high": 0.7},
		"method":      {"egm-review-loop": 0.9, "odd": 0.07, "ftd": 0.03},
	}}
	withFakeProvider(t, root, fake)

	got, err, stderr := runDecide(t, root, decideDeclaration)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if got["method"] != "odd" {
		t.Fatalf("method = %v, want the next-best available method", got["method"])
	}
	if got["substituted"] != true {
		t.Fatalf("substituted = %v, want true", got["substituted"])
	}
	if !strings.Contains(stderr, "unavailable") {
		t.Fatalf("stderr = %q, want the substitution explained", stderr)
	}
}

func TestDecideReportsProviderFailureAsExitThree(t *testing.T) {
	root := routerRepo(t)
	fake := &fakeProvider{err: &provider.Failure{Kind: provider.FailureQuota, Status: 429, Attempts: 3}}
	withFakeProvider(t, root, fake)

	got, err, _ := runDecide(t, root, decideDeclaration)
	if err == nil {
		t.Fatalf("decide succeeded despite a provider failure: %v", got)
	}
	if code := ExitCode(err); code != ExitProvider {
		t.Fatalf("exit = %d, want %d", code, ExitProvider)
	}
	if got != nil {
		t.Fatal("decide printed JSON on a provider failure")
	}
}

func TestDecideRejectsAnInvalidDeclarationAsExitTwo(t *testing.T) {
	root := routerRepo(t)
	withFakeProvider(t, root, &fakeProvider{})

	bad := strings.Replace(decideDeclaration, `"schema_version": 1,`, `"schema_version": 1, "method": "asdd",`, 1)
	_, err, _ := runDecide(t, root, bad)
	if err == nil || ExitCode(err) != ExitValidation {
		t.Fatalf("err = %v, exit = %d, want a validation failure", err, ExitCode(err))
	}

	unknownSubsystem := strings.Replace(decideDeclaration, `"subsystems": ["router"]`, `"subsystems": ["billing"]`, 1)
	_, err, _ = runDecide(t, root, unknownSubsystem)
	if err == nil || ExitCode(err) != ExitValidation {
		t.Fatalf("err = %v, want a validation failure for an unknown subsystem", err)
	}
}

func TestDecideRequiresConfig(t *testing.T) {
	root := routerRepo(t)
	if err := os.Remove(filepath.Join(root, ".router", "config.json")); err != nil {
		t.Fatalf("remove config: %v", err)
	}
	withFakeProvider(t, root, &fakeProvider{})

	_, err, _ := runDecide(t, root, decideDeclaration)
	if err == nil || ExitCode(err) != ExitValidation {
		t.Fatalf("err = %v, want a validation failure naming the config", err)
	}
	var stdout bytes.Buffer
	_ = stdout // config errors never write JSON
}

// The endpoint may name the account instead of embedding it. The real provider
// factory is exercised here (not the test override), because the substitution
// happens while building the provider.
func TestProviderFactorySubstitutesTheAccountPlaceholder(t *testing.T) {
	cfg := &state.Config{
		Provider: "cloudflare",
		Model:    "@cf/cloudflare/clef",
		Endpoint: "https://api.example/client/v4/accounts/{account_id}/ai/run/@cf/cloudflare/clef",
	}
	t.Setenv(state.EnvCloudflareToken, "test-token")
	t.Setenv(state.EnvCloudflareAccount, "")

	_, err := providerFactory(cfg)
	if err == nil || !strings.Contains(err.Error(), state.EnvCloudflareAccount) {
		t.Fatalf("providerFactory = %v, want an error naming %s", err, state.EnvCloudflareAccount)
	}

	t.Setenv(state.EnvCloudflareAccount, "abc123")
	p, err := providerFactory(cfg)
	if err != nil {
		t.Fatalf("providerFactory: %v", err)
	}
	cf, ok := p.(*provider.Cloudflare)
	if !ok {
		t.Fatalf("provider = %T, want *provider.Cloudflare", p)
	}
	if !strings.Contains(cf.Endpoint, "/accounts/abc123/") {
		t.Fatalf("endpoint = %q, want the account id substituted", cf.Endpoint)
	}
	if strings.Contains(cf.Endpoint, "{account_id}") {
		t.Fatalf("endpoint = %q, placeholder left in place", cf.Endpoint)
	}
}
