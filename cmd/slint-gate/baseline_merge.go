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

Safely appends newly-measured SLIs into an existing baseline without
weakening or deleting anything already there. Only --mode append-new-only
is currently supported.

Flags:
`

func runBaselineMerge(args []string) error {
	fs := flag.NewFlagSet("baseline merge", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, baselineMergeUsage)
		fs.PrintDefaults()
	}
	baselinePath := fs.String("baseline", "", "Path to the existing baseline summary JSON (required)")
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the current measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML the current summary must pass")
	mode := fs.String("mode", "append-new-only", "Merge mode (only append-new-only is currently supported)")
	output := fs.String("output", "", "Output path (defaults to --baseline, i.e. merge in place)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*baselinePath) == "" {
		return fmt.Errorf("--baseline is required")
	}
	if *mode != "append-new-only" {
		return fmt.Errorf("unsupported --mode %q; only 'append-new-only' is currently supported (review-existing and force-replace are planned for a future release)", *mode)
	}
	outPath := *output
	if strings.TrimSpace(outPath) == "" {
		outPath = *baselinePath
	}

	if _, statErr := os.Stat(*baselinePath); statErr != nil {
		return fmt.Errorf("--baseline %s does not exist; run 'slint-gate baseline approve' first", *baselinePath)
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

	baseValues := resultValues(baseline)
	curValues := resultValues(cur)

	var appended []summary.SLIResult
	var rejected []string
	for _, r := range cur.Results {
		if r.Value == nil {
			continue
		}
		if _, ok := baseValues[r.ID]; !ok {
			appended = append(appended, r)
		}
	}
	for id, baseVal := range baseValues {
		if curVal, ok := curValues[id]; ok && curVal != baseVal {
			rejected = append(rejected, fmt.Sprintf("%s: current summary has %v, baseline has %v", id, curVal, baseVal))
		}
	}
	sort.Slice(appended, func(i, j int) bool { return appended[i].ID < appended[j].ID })
	sort.Strings(rejected)

	baseline.Results = append(baseline.Results, appended...)

	// kube-slint-no-stat-before-write: the earlier os.Stat only checks that
	// --baseline already exists (a precondition, not an overwrite guard);
	// this is a single-user local CLI artifact write, not a shared/multi-tenant race.
	// nosemgrep
	if err := summary.WriteFile(outPath, baseline); err != nil {
		return fmt.Errorf("write merged baseline: %w", err)
	}

	printMergeReview(*mode, appended, baseValues, rejected, outPath)
	return nil
}

func printMergeReview(mode string, appended []summary.SLIResult, baseValues map[string]float64, rejected []string, output string) {
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

	unchangedIDs := make([]string, 0, len(baseValues))
	for id := range baseValues {
		unchangedIDs = append(unchangedIDs, id)
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
		fmt.Println("    Reason: append-new-only does not weaken existing baseline values.")
	}

	result := "MERGED"
	switch {
	case len(rejected) > 0:
		result = "MERGED_WITH_REJECTIONS"
	case len(appended) == 0:
		result = "NO_CHANGE"
	}
	fmt.Printf("\nResult:\n  %s\n\n", result)
	fmt.Println("Output:")
	fmt.Printf("  %s\n", output)
}
