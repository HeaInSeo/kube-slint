package main

import (
	"flag"
	"fmt"
	"os"
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

	baseValues := baseline.ResultValues()

	var directions map[string]string
	if *mode == "review-existing" {
		directions = loadMetricDirections(*policyPath)
	}
	appended, updated, rejected := computeMergePlan(*mode, baseline, cur, directions)
	applyMergePlan(&baseline, appended, updated)

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
		if !ok || curVal == baseVal {
			continue
		}
		if mergeChangeApplies(mode, directions[id], baseVal, curVal) {
			updated = append(updated, mergeUpdate{id, baseVal, curVal})
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

// applyMergePlan mutates baseline in place: appends new SLIs and overwrites
// the Value of any existing SLI in updated.
func applyMergePlan(baseline *summary.Summary, appended []summary.SLIResult, updated []mergeUpdate) {
	baseline.Results = append(baseline.Results, appended...)
	if len(updated) == 0 {
		return
	}
	newValByID := make(map[string]float64, len(updated))
	for _, u := range updated {
		newValByID[u.ID] = u.NewVal
	}
	for i := range baseline.Results {
		if v, ok := newValByID[baseline.Results[i].ID]; ok {
			vCopy := v
			baseline.Results[i].Value = &vCopy
		}
	}
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
