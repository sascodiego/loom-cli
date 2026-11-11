package cli

import (
	"context"
	"fmt"
	"sort"

	"loom/router/internal/state"
)

// SampleFloor is the minimum number of denominator decisions below which the
// agreement rate is not reported at all (design G6/S13: the delegation
// criterion needs at least 30 clean decisions before the rate means anything).
const SampleFloor = 30

func init() { Register("metrics", runMetrics) }

// StaleSchemaKey names the exclusion bucket for clef decisions recorded under an
// obsolete pre-flight schema.
const StaleSchemaKey = "stale_schema"

// decisionEvents are the history events that carry a routing decision: the
// first one in a routing id's history fixes the decision that metrics measures.
var decisionEvents = map[string]bool{
	state.EventDecided:          true,
	state.EventGateOverridden:   true,
	state.EventJointFallback:    true,
	state.EventReRouteRequested: true,
}

// metricsReport is the JSON document `router metrics` prints. It is the same
// document with and without --json: stdout stays machine-readable either way.
type metricsReport struct {
	Sample        int                    `json:"sample"`
	Floor         int                    `json:"floor"`
	Suppressed    bool                   `json:"suppressed"`
	AgreementRate *float64               `json:"agreement_rate"`
	ByDecidedBy   map[string]int         `json:"by_decided_by"`
	ByMethod      map[string]methodTally `json:"by_method"`
	Excluded      map[string]int         `json:"excluded"`
}

// methodTally stratifies one method's decisions by approval outcome.
type methodTally struct {
	Decisions int `json:"decisions"`
	Approved  int `json:"approved"`
}

// runMetrics computes the phase-1 agreement rate over .router/history/.
// Clean = decided_by clef at the current schema version; gate overrides count
// in the denominator as disagreements; joint_fallback and override_re_route
// are excluded from the denominator and counted separately.
func runMetrics(ctx context.Context, env *Env, args []string) error {
	fs := NewFlagSet("metrics")
	asJSON := fs.Bool("json", false, "print the JSON report only (stdout is JSON either way)")
	if err := ParseFlags(fs, args); err != nil {
		return err
	}

	report, warnings, err := computeMetrics(env.Root)
	if err != nil {
		return Fail("%v", err)
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintln(env.Stderr, "router metrics: "+w)
	}

	if err := WriteJSON(env.Stdout, report); err != nil {
		return err
	}
	if *asJSON {
		return nil
	}
	if report.Suppressed {
		_, _ = fmt.Fprintf(env.Stderr,
			"router metrics: %d decisions in the denominator, below the floor of %d — agreement rate suppressed\n",
			report.Sample, report.Floor)
	} else {
		_, _ = fmt.Fprintf(env.Stderr,
			"router metrics: agreement rate %.2f over %d decisions (floor %d)\n",
			*report.AgreementRate, report.Sample, report.Floor)
	}
	return nil
}

// computeMetrics reads every history file and classifies each routing
// decision. A malformed history file is skipped with a warning, never a
// failure: metrics is a read-only report, and one corrupt file must not hide
// the rest of the measurements.
func computeMetrics(root string) (*metricsReport, []string, error) {
	var warnings []string
	report := &metricsReport{
		Floor:       SampleFloor,
		ByDecidedBy: map[string]int{},
		ByMethod:    map[string]methodTally{},
		Excluded: map[string]int{
			string(state.DecidedByJointFallback):   0,
			string(state.DecidedByOverrideReRoute): 0,
			// StaleSchema counts clef decisions recorded under an obsolete
			// pre-flight schema. They stay visible in by_decided_by (they are
			// real decisions) and are excluded here by name, so a reader can
			// reconcile the denominator instead of finding a silent gap.
			StaleSchemaKey: 0,
		},
	}

	ids, err := state.ListHistoryIDs(root)
	if err != nil {
		return nil, nil, err
	}
	var cleanApproved, denominator int
	for _, id := range ids {
		events, err := state.ReadHistory(root, id)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s.jsonl: %v", id, err))
			continue
		}
		var decision *state.Event
		for i := range events {
			if decisionEvents[events[i].Event] {
				decision = &events[i]
				break
			}
		}
		if decision == nil || decision.Record == nil {
			continue
		}
		rec := decision.Record
		report.ByDecidedBy[string(rec.DecidedBy)]++

		switch rec.DecidedBy {
		case state.DecidedByJointFallback, state.DecidedByOverrideReRoute:
			report.Excluded[string(rec.DecidedBy)]++
		case state.DecidedByClef:
			if rec.SchemaVersion != state.SchemaVersion {
				// Not comparable with the current criteria contract: excluded
				// from the denominator for a reason the report names.
				report.Excluded[StaleSchemaKey]++
				continue
			}
			denominator++
			tally := report.ByMethod[rec.Method]
			tally.Decisions++
			if rec.HumanApproved {
				cleanApproved++
				tally.Approved++
			}
			report.ByMethod[rec.Method] = tally
		case state.DecidedByUserGateOverride:
			denominator++
			tally := report.ByMethod[rec.Method]
			tally.Decisions++
			report.ByMethod[rec.Method] = tally
		}
	}

	report.Sample = denominator
	if denominator < SampleFloor {
		report.Suppressed = true
	} else {
		rate := float64(cleanApproved) / float64(denominator)
		report.AgreementRate = &rate
	}

	sort.Strings(warnings)
	return report, warnings, nil
}
