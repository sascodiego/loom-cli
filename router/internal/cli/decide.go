package cli

import (
	"context"
	"fmt"
	"os"
	"sort"

	"loom/router/internal/preflight"
	"loom/router/internal/provider"
	"loom/router/internal/registry"
	"loom/router/internal/state"
)

func init() { Register("decide", runDecideCmd) }

// providerFactory builds the classifier from the project config. It is a
// package variable so tests can substitute a provider without a network.
var providerFactory = func(cfg *state.Config) (provider.Provider, error) {
	if cfg.Provider != "cloudflare" {
		return nil, fmt.Errorf("unsupported provider %q (v1 implements cloudflare)", cfg.Provider)
	}
	token := os.Getenv(state.EnvCloudflareToken)
	if token == "" {
		return nil, fmt.Errorf("missing %s in the environment", state.EnvCloudflareToken)
	}
	return &provider.Cloudflare{
		Endpoint: cfg.Endpoint,
		Model:    cfg.Model,
		Token:    token,
	}, nil
}

// DecideOutput is the JSON document `router decide` prints. Everything here is
// what the confirmation gate must be able to show, and nothing else: the gate
// never displays a fact the record cannot capture.
type DecideOutput struct {
	Tier          string                        `json:"tier"`
	Method        string                        `json:"method"`
	Probabilities map[string]map[string]float64 `json:"probabilities"`
	RequestID     string                        `json:"request_id"`
	Provider      string                        `json:"provider"`
	Model         string                        `json:"model"`
	CriteriaHash  string                        `json:"criteria_hash"`
	// PayloadHash pins the exact declaration that was classified, so `record`
	// can carry it and a later audit can prove what was sent.
	PayloadHash string `json:"payload_hash"`
	// Ambiguous means the top two method probabilities are within 0.05: the
	// gate must show both options rather than imply a clear winner.
	Ambiguous bool `json:"ambiguous"`
	// Substituted means the classifier's top choice was not dispatchable and
	// the next-best available method was taken instead.
	Substituted bool `json:"substituted,omitempty"`
}

// AmbiguityThreshold is the confidence gap below which a decision is ambiguous.
const AmbiguityThreshold = 0.05

// runDecideCmd classifies a declared pre-flight. The sequence is fixed:
// declaration -> subsystem validation -> derive -> request -> provider ->
// output. No diff is read and nothing is written.
func runDecideCmd(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("decide")
	input := fs.String("input", "", "path to the pre-flight declaration JSON, or - for stdin")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}
	data, err := ReadInput(*input, env.Stdin)
	if err != nil {
		return err
	}
	decl, err := preflight.Parse(data)
	if err != nil {
		return Fail("invalid pre-flight: %v", err)
	}
	if err := decl.Validate(env.Root); err != nil {
		return Fail("invalid pre-flight: %v", err)
	}

	reg, err := registry.Load(env.Root)
	if err != nil {
		return Fail("%v", err)
	}
	if err := reg.Validate(env.Root); err != nil {
		return Fail("%v", err)
	}
	criteriaHash, err := reg.CriteriaHash(env.Root)
	if err != nil {
		return Fail("%v", err)
	}
	criteria, err := dispatchableCriteria(env.Root, reg)
	if err != nil {
		return Fail("%v", err)
	}

	cfg, err := state.LoadConfig(env.Root)
	if err != nil {
		return Fail("%v", err)
	}
	branch, err := CurrentBranch(env.Root)
	if err != nil {
		return err
	}
	modified, err := preflight.ModifiedFileCount(env.Root)
	if err != nil {
		return Fail("%v", err)
	}

	payload, err := preflight.Derive(preflight.DeriveInput{
		Raw:         data,
		Declaration: decl,
		Facts:       provider.SessionFacts{Branch: branch, ModifiedFiles: modified},
		Criteria:    criteria,
	})
	if err != nil {
		return Fail("%v", err)
	}

	p, err := providerFactory(cfg)
	if err != nil {
		return Fail("%v", err)
	}
	decision, err := p.Decide(ctx, payload.State, payload.Questions)
	if err != nil {
		// Provider failures are exit 3: the harness switches to the joint
		// fallback conversation and shows this message as evidence.
		return FailProvider("%v", err)
	}

	out, err := resolve(decision, reg, cfg, criteriaHash, payload.PayloadHash, env)
	if err != nil {
		return err
	}
	return WriteJSON(env.Stdout, out)
}

// resolve turns the raw probabilities into the single decision the gate shows:
// argmax tier, argmax available method, ambiguity flag, substitution flag.
func resolve(decision *provider.Decision, reg *registry.Registry, cfg *state.Config,
	criteriaHash, payloadHash string, env *Env) (*DecideOutput, error) {

	tier, err := argmax(decision.Probabilities["effort_tier"])
	if err != nil {
		return nil, FailProvider("effort_tier: %v", err)
	}
	methodProbs := decision.Probabilities["method"]
	ranked := rank(methodProbs)
	if len(ranked) == 0 {
		return nil, FailProvider("the provider returned no method probabilities")
	}

	method := ""
	substituted := false
	for _, cand := range ranked {
		m, ok := reg.Method(cand.name)
		if ok && m.Available {
			method = cand.name
			break
		}
	}
	if method == "" {
		return nil, FailProvider("the provider suggested no dispatchable method (options: %v)", rankedNames(ranked))
	}
	if method != ranked[0].name {
		substituted = true
		_, _ = fmt.Fprintf(env.Stderr,
			"router decide: %q is unavailable in v1; substituted the next-best available method %q\n",
			ranked[0].name, method)
	}

	ambiguous := false
	if len(ranked) >= 2 {
		ambiguous = ranked[0].probability-ranked[1].probability <= AmbiguityThreshold
	}

	return &DecideOutput{
		Tier:          tier,
		Method:        method,
		Probabilities: decision.Probabilities,
		RequestID:     decision.RequestID,
		Provider:      cfg.Provider,
		Model:         cfg.Model,
		CriteriaHash:  criteriaHash,
		PayloadHash:   payloadHash,
		Ambiguous:     ambiguous,
		Substituted:   substituted,
	}, nil
}

// dispatchableCriteria collects the criteria of the available methods in
// registry order. An unavailable method contributes nothing, so the endpoint
// cannot suggest it.
func dispatchableCriteria(root string, reg *registry.Registry) ([]provider.Criterion, error) {
	out := make([]provider.Criterion, 0, len(reg.Methods))
	for _, m := range reg.Available() {
		data, err := reg.CriteriaContent(root, m.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, provider.Criterion{Method: m.Name, Content: string(data)})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no dispatchable method in the registry")
	}
	return out, nil
}

type rankedOption struct {
	name        string
	probability float64
}

// rank orders the method probabilities descending, with the option name as a
// deterministic tiebreak so two runs never disagree on a tie.
func rank(probs map[string]float64) []rankedOption {
	out := make([]rankedOption, 0, len(probs))
	for name, p := range probs {
		out = append(out, rankedOption{name: name, probability: p})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].probability == out[j].probability {
			return out[i].name < out[j].name
		}
		return out[i].probability > out[j].probability
	})
	return out
}

func rankedNames(ranked []rankedOption) []string {
	names := make([]string, len(ranked))
	for i, r := range ranked {
		names[i] = r.name
	}
	return names
}

// argmax returns the highest-probability option, with a deterministic tiebreak.
func argmax(probs map[string]float64) (string, error) {
	ranked := rank(probs)
	if len(ranked) == 0 {
		return "", fmt.Errorf("no probabilities returned")
	}
	return ranked[0].name, nil
}
