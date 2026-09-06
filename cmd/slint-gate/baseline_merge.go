package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

const baselineMergeUsage = `Usage: slint-gate baseline merge [flags]

Merges newly-measured SLIs into an existing baseline. Modes:

  append-new-only  (default) New SLIs are appended; an existing SLI's
                    value is never changed, even if the new value would
                    be an improvement. Changed existing values are
                    reported as rejected.
  review-existing   Like append-new-only, but an existing SLI's value is
                    updated when the current measurement is a genuine
                    improvement in the direction implied by policy.yaml's
                    threshold operator for that metric. A change with no
                    recognized direction, or a regression, is still
                    rejected and left unchanged.
  force-replace     New SLIs are appended and every existing SLI with a
                    differing current value is unconditionally overwritten,
                    regardless of direction. An explicit escape hatch for
                    deliberate rebaselining — not a default-safe mode.

Flags:
`

var supportedMergeModes = map[string]bool{
	"append-new-only": true,
	"review-existing": true,
	"force-replace":   true,
}

// mergeUpdate records an existing baseline SLI value that was changed by
// review-existing (a confirmed improvement) or force-replace (unconditional).
type mergeUpdate struct {
	ID             string
	OldVal, NewVal float64
	// Cur is the full CURRENT result whose record replaces the baseline result.
	// Replacing only the value would leave stale comparability (KSL-T4) and stale
	// per-SLI evidence facts — InputsMissing, and via the top-level skipped set,
	// SkippedSLIs (KSL-T3) — that newEvidenceIndex(baseline) would later misjudge.
	Cur summary.SLIResult
}

func runBaselineMerge(args []string) error {
	fs := flag.NewFlagSet("baseline merge", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, baselineMergeUsage)
		fs.PrintDefaults()
	}
	baselinePath := fs.String("baseline", "", "Path to the existing baseline summary JSON (required)")
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the current measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML the current summary must pass")
	mode := fs.String("mode", "append-new-only", "Merge mode: append-new-only | review-existing | force-replace")
	output := fs.String("output", "", "Output path (defaults to --baseline, i.e. merge in place)")
	force := fs.Bool("force", false, "Overwrite --output if it already exists and differs from --baseline")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*baselinePath) == "" {
		return fmt.Errorf("--baseline is required")
	}
	if !supportedMergeModes[*mode] {
		return fmt.Errorf("unsupported --mode %q; supported: append-new-only, review-existing, force-replace", *mode)
	}
	outPath := *output
	if strings.TrimSpace(outPath) == "" {
		outPath = *baselinePath
	}

	if _, statErr := os.Stat(*baselinePath); statErr != nil {
		return fmt.Errorf("--baseline %s does not exist; run 'slint-gate baseline approve' first", *baselinePath)
	}

	// In-place merge (outPath == baselinePath, the default) is expected to
	// overwrite the baseline it just read from — that's the point of merge.
	// Only guard the case where --output explicitly points somewhere else,
	// so a typo'd or pre-existing unrelated path isn't silently clobbered.
	if outPath != *baselinePath && !*force {
		if _, statErr := os.Stat(outPath); statErr == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite", outPath)
		}
	}

	result := gate.Evaluate(gate.Request{
		MeasurementPath: *summaryPath,
		PolicyPath:      *policyPath,
	})
	if result.GateResult != gate.GatePass {
		return fmt.Errorf("the current summary must pass its policy before merging (gate_result=%s); run 'slint-gate inspect --summary %s'", result.GateResult, *summaryPath)
	}

	baseline, err := summary.LoadFile(*baselinePath)
	if err != nil {
		return fmt.Errorf("could not load baseline: %w", err)
	}
	cur, err := summary.LoadFile(*summaryPath)
	if err != nil {
		return fmt.Errorf("could not load summary: %w", err)
	}

	// Reject a cross-contract merge rather than write a hybrid baseline (e.g. an
	// slo.v3 top-level carrying slo.v4 comparability), which a later regression
	// would reject as legacy. Rebaselining across contracts is a migration concern,
	// not a merge; regenerate the baseline from a matching-contract summary instead.
	if baseline.SchemaVersion != cur.SchemaVersion {
		return fmt.Errorf("cannot merge across measurement contracts: baseline is %q but the current summary is %q; regenerate the baseline from a matching-contract summary",
			baseline.SchemaVersion, cur.SchemaVersion)
	}

	baseValues := baseline.ResultValues()

	var directions map[string]string
	if *mode == "review-existing" {
		directions = loadMetricDirections(*policyPath)
	}
	appended, updated, rejected := computeMergePlan(*mode, baseline, cur, directions)
	applyMergePlan(&baseline, appended, updated, cur, *mode)

	// kube-slint-no-stat-before-write: the os.Stat calls above are a
	// --baseline existence precondition and a --force overwrite guard on
	// outPath; this is a single-user local CLI artifact write, not a
	// shared/multi-tenant race.
	// nosemgrep
	if err := summary.WriteFile(outPath, baseline); err != nil {
		return fmt.Errorf("write merged baseline: %w", err)
	}

	printMergeReview(*mode, appended, updated, baseValues, rejected, outPath)
	return nil
}

// computeMergePlan decides, per mode, which current-summary SLIs are new
// (to append), which existing baseline SLIs should have their value
// changed (to update, per mode's rules), and which existing-value changes
// are left untouched (rejected). It does not mutate baseline or cur.
func computeMergePlan(mode string, baseline, cur summary.Summary, directions map[string]string) (appended []summary.SLIResult, updated []mergeUpdate, rejected []string) {
	baseValues := baseline.ResultValues()
	curValues := cur.ResultValues()
	curByID := make(map[string]summary.SLIResult, len(cur.Results))
	for _, r := range cur.Results {
		curByID[r.ID] = r
	}
	baseByID := make(map[string]summary.SLIResult, len(baseline.Results))
	for _, r := range baseline.Results {
		baseByID[r.ID] = r
	}

	for _, r := range cur.Results {
		if r.Value == nil {
			continue
		}
		if _, ok := baseValues[r.ID]; !ok {
			appended = append(appended, r)
		}
	}

	for id, baseVal := range baseValues {
		curVal, ok := curValues[id]
		if !ok {
			continue
		}
		if curVal == baseVal {
			// Value unchanged: force-replace still refreshes the record when ANY part
			// of the current evidence differs (comparability, InputsMissing, status,
			// …), so the baseline fully matches the current artifact and a later
			// newEvidenceIndex(baseline)/regression is not misled by stale evidence.
			// Other modes leave an unchanged value untouched.
			if mode == "force-replace" && !reflect.DeepEqual(baseByID[id], curByID[id]) {
				updated = append(updated, mergeUpdate{ID: id, OldVal: baseVal, NewVal: curVal, Cur: curByID[id]})
			}
			continue
		}
		if mergeChangeApplies(mode, directions[id], baseVal, curVal) {
			updated = append(updated, mergeUpdate{ID: id, OldVal: baseVal, NewVal: curVal, Cur: curByID[id]})
		} else {
			rejected = append(rejected, fmt.Sprintf("%s: current summary has %v, baseline has %v", id, curVal, baseVal))
		}
	}

	sort.Slice(appended, func(i, j int) bool { return appended[i].ID < appended[j].ID })
	sort.Slice(updated, func(i, j int) bool { return updated[i].ID < updated[j].ID })
	sort.Strings(rejected)
	return appended, updated, rejected
}

// mergeChangeApplies reports whether an existing SLI's changed value should
// be applied to the baseline, per mode: force-replace always applies;
// review-existing applies only a confirmed improvement in direction;
// append-new-only never applies (existing values are immutable).
func mergeChangeApplies(mode, direction string, oldVal, newVal float64) bool {
	switch mode {
	case "force-replace":
		return true
	case "review-existing":
		return isImprovement(direction, oldVal, newVal)
	default: // append-new-only
		return false
	}
}

// skippedSet returns the set of SLI IDs a summary marks skipped.
func skippedSet(s summary.Summary) map[string]bool {
	set := map[string]bool{}
	if s.Reliability != nil {
		for _, id := range s.Reliability.SkippedSLIs {
			set[id] = true
		}
	}
	return set
}

// applyMergePlan mutates baseline in place: appends new SLIs, replaces the full
// record of any existing SLI in updated, and reconciles the top-level skipped set
// against the current run.
func applyMergePlan(baseline *summary.Summary, appended []summary.SLIResult, updated []mergeUpdate, cur summary.Summary, mode string) {
	baseline.Results = append(baseline.Results, appended...)
	if len(updated) > 0 {
		updateByID := make(map[string]summary.SLIResult, len(updated))
		for _, u := range updated {
			updateByID[u.ID] = u.Cur
		}
		for i := range baseline.Results {
			if curRec, ok := updateByID[baseline.Results[i].ID]; ok {
				// Replace the FULL evidence record (value, comparability, inputs, status)
				// so no stale baseline evidence fact survives alongside the new value and
				// misleads a later newEvidenceIndex(baseline) (KSL-T3/T4).
				baseline.Results[i] = curRec
			}
		}
	}

	// Determine which SLIs' skipped membership is redecided from the current run.
	// force-replace rebaselines to current, so EVERY shared/appended SLI adopts the
	// current skipped membership — including one whose record was byte-identical (so
	// it never entered `updated`) but whose skipped membership changed. Other modes
	// only reconcile the SLIs they actually merged (appended/updated).
	mergedIDs := make(map[string]bool, len(baseline.Results))
	for _, r := range baseline.Results {
		mergedIDs[r.ID] = true
	}
	reconcileFor := map[string]bool{}
	if mode == "force-replace" {
		for _, r := range cur.Results {
			if mergedIDs[r.ID] {
				reconcileFor[r.ID] = true
			}
		}
	} else {
		for _, r := range appended {
			reconcileFor[r.ID] = true
		}
		for _, u := range updated {
			reconcileFor[u.ID] = true
		}
	}
	reconcileSkippedSLIs(baseline, reconcileFor, skippedSet(cur))
}

// reconcileSkippedSLIs redecides, for every SLI in reconcileFor, its membership in
// the merged baseline's top-level skipped set from the current run (curSkipped):
// importing a genuine current skip (so an unreliable value is not later treated as
// sufficient) and dropping a stale one (so a merged value is not treated as
// insufficient) — KSL-T3. SLIs outside reconcileFor keep their existing markers.
func reconcileSkippedSLIs(baseline *summary.Summary, reconcileFor, curSkipped map[string]bool) {
	var toAdd []string
	for id := range reconcileFor {
		if curSkipped[id] {
			toAdd = append(toAdd, id)
		}
	}
	if (baseline.Reliability == nil || len(baseline.Reliability.SkippedSLIs) == 0) && len(toAdd) == 0 {
		return
	}
	if baseline.Reliability == nil {
		baseline.Reliability = &summary.Reliability{}
	}
	seen := map[string]bool{}
	kept := make([]string, 0, len(baseline.Reliability.SkippedSLIs)+len(toAdd))
	for _, id := range baseline.Reliability.SkippedSLIs {
		// Drop markers for reconciled SLIs; their status is redecided below.
		if !reconcileFor[id] && !seen[id] {
			kept = append(kept, id)
			seen[id] = true
		}
	}
	for _, id := range toAdd {
		if !seen[id] {
			kept = append(kept, id)
			seen[id] = true
		}
	}
	sort.Strings(kept)
	baseline.Reliability.SkippedSLIs = kept
}

// isImprovement reports whether newVal is a genuine improvement over oldVal
// given a metric direction ("lower", "higher", or "" for unknown). An
// unrecognized direction never counts as an improvement — review-existing
// only auto-updates values it can positively confirm are better.
func isImprovement(direction string, oldVal, newVal float64) bool {
	switch direction {
	case "lower":
		return newVal < oldVal
	case "higher":
		return newVal > oldVal
	default:
		return false
	}
}

func printMergeReview(mode string, appended []summary.SLIResult, updated []mergeUpdate, baseValues map[string]float64, rejected []string, output string) {
	fmt.Println("Baseline merge review:")
	fmt.Println()
	fmt.Println("Mode:")
	fmt.Printf("  %s\n\n", mode)

	fmt.Println("New SLIs to append:")
	if len(appended) == 0 {
		fmt.Println("  (none)")
	}
	for _, r := range appended {
		fmt.Printf("  %s = %v\n", r.ID, *r.Value)
	}

	if mode != "append-new-only" {
		fmt.Println("\nExisting SLIs updated:")
		if len(updated) == 0 {
			fmt.Println("  (none)")
		}
		for _, u := range updated {
			fmt.Printf("  %s: %v → %v\n", u.ID, u.OldVal, u.NewVal)
		}
	}

	updatedIDs := make(map[string]bool, len(updated))
	for _, u := range updated {
		updatedIDs[u.ID] = true
	}
	unchangedIDs := make([]string, 0, len(baseValues))
	for id := range baseValues {
		if !updatedIDs[id] {
			unchangedIDs = append(unchangedIDs, id)
		}
	}
	sort.Strings(unchangedIDs)
	fmt.Println("\nExisting SLIs unchanged:")
	if len(unchangedIDs) == 0 {
		fmt.Println("  (none)")
	}
	for _, id := range unchangedIDs {
		fmt.Printf("  %s = %v\n", id, baseValues[id])
	}

	fmt.Println("\nRejected changes:")
	if len(rejected) == 0 {
		fmt.Println("  (none)")
	}
	for _, r := range rejected {
		fmt.Printf("  %s\n", r)
		if mode == "review-existing" {
			fmt.Println("    Reason: not a confirmed improvement for this metric's policy direction.")
		} else {
			fmt.Println("    Reason: append-new-only does not weaken existing baseline values.")
		}
	}

	result := "MERGED"
	switch {
	case len(rejected) > 0:
		result = "MERGED_WITH_REJECTIONS"
	case len(appended) == 0 && len(updated) == 0:
		result = "NO_CHANGE"
	}
	fmt.Printf("\nResult:\n  %s\n\n", result)
	fmt.Println("Output:")
	fmt.Printf("  %s\n", output)
}
