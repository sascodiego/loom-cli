package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

const fixtureDeclaration = `{"schema_version":1,"request_summary":"segment quota by tier","intent":"change enforcement","scope_summary":"quota middleware","subsystems":["router"],"scope_estimate":{"steps":2,"sessions":1}}`

func fixtureCriteria() []Criterion {
	return []Criterion{
		{Method: "odd", Content: "odd criteria"},
		{Method: "ftd", Content: "ftd criteria"},
	}
}

func TestBuildStateIsTheDeclarationPlusSessionFactsPlusCriteria(t *testing.T) {
	state, err := BuildState(json.RawMessage(fixtureDeclaration), SessionFacts{
		Branch:        "feat/x",
		ModifiedFiles: 3,
	}, fixtureCriteria())
	if err != nil {
		t.Fatalf("BuildState: %v", err)
	}
	want := `{"schema_version":1,` +
		`"declaration":` + fixtureDeclaration + `,` +
		`"session":{"branch":"feat/x","modified_files":3},` +
		`"criteria":[{"method":"odd","content":"odd criteria"},` +
		`{"method":"ftd","content":"ftd criteria"}]}`
	if string(state) != want {
		t.Fatalf("state = %s\nwant    %s", state, want)
	}
}

func TestBuildStateRejectsAnEmptyDeclaration(t *testing.T) {
	if _, err := BuildState(nil, SessionFacts{Branch: "x"}, fixtureCriteria()); err == nil ||
		!strings.Contains(err.Error(), "declaration") {
		t.Fatalf("BuildState = %v, want a declaration error", err)
	}
	if _, err := BuildState(json.RawMessage(`not json`), SessionFacts{Branch: "x"}, fixtureCriteria()); err == nil {
		t.Fatal("BuildState accepted a non-JSON declaration")
	}
}

func TestBuildQuestionsCarriesTypedEnumsAndVerbatimDescriptions(t *testing.T) {
	qs := BuildQuestions(fixtureCriteria())
	if len(qs) != 2 {
		t.Fatalf("got %d questions, want 2 (effort_tier, method)", len(qs))
	}
	if qs[0].Key != "effort_tier" {
		t.Fatalf("first question = %q, want effort_tier", qs[0].Key)
	}
	var tiers []string
	for _, o := range qs[0].Options {
		tiers = append(tiers, o.Value)
		if o.Description == "" {
			t.Fatalf("tier option %q has no description", o.Value)
		}
	}
	if strings.Join(tiers, ",") != "low,medium,high" {
		t.Fatalf("tiers = %v, want low,medium,high", tiers)
	}

	if qs[1].Key != "method" {
		t.Fatalf("second question = %q, want method", qs[1].Key)
	}
	if len(qs[1].Options) != 2 {
		t.Fatalf("method options = %d, want 2 (criteria given)", len(qs[1].Options))
	}
	if qs[1].Options[0].Value != "odd" || qs[1].Options[0].Description != "odd criteria" {
		t.Fatalf("method option = %+v, want the criteria content verbatim", qs[1].Options[0])
	}
}

func TestBuildQuestionsRejectsEmptyCriteria(t *testing.T) {
	if qs := BuildQuestions(nil); len(qs) != 1 {
		t.Fatalf("questions = %v, want only effort_tier when no method is dispatchable", qs)
	}
}
