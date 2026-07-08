package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

const baselineDiffUsage = `Usage: slint-gate baseline diff [flags]

Compares a stored baseline against a current measurement summary: which
SLIs are new, which changed, and which are missing from the current run.
Read-only — never fails the process except on an unreadable baseline or
summary file.

Flags:
`

// diffThresholdRule is the minimal shape baseline diff needs from a
// policy.yaml threshold rule (metric + operator), read directly via yaml.v3
// rather than importing pkg/gate, since diff only needs direction, not
// full policy validation/evaluation.
type diffThresholdRule struct {
	Metric   string `yaml:"metric"`
	Operator string `yaml:"operator"`
}

type diffPolicyDoc struct {
	Thresholds []diffThresholdRule `yaml:"thresholds"`
}

func runBaselineDiff(args []string) error {
	fs := flag.NewFlagSet("baseline diff", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, baselineDiffUsage)
		fs.PrintDefaults()
	}
	baselinePath := fs.String("baseline", "", "Path to the stored baseline summary JSON (required)")
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the current measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Optional path to policy YAML, used only to infer improve/weaken direction")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*baselinePath) == "" {
		return fmt.Errorf("--baseline is required")
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
	curValues := cur.ResultValues()
	directions := loadMetricDirections(*policyPath)

	var newIDs, changedIDs, missingIDs []string
	for id := range curValues {
		if _, ok := baseValues[id]; !ok {
			newIDs = append(newIDs, id)
		}
	}
	for id, baseVal := range baseValues {
		curVal, ok := curValues[id]
		if !ok {
			missingIDs = append(missingIDs, id)
			continue
		}
		if curVal != baseVal {
			changedIDs = append(changedIDs, id)
		}
	}
	sort.Strings(newIDs)
	sort.Strings(changedIDs)
	sort.Strings(missingIDs)

	existingIDs := make([]string, 0, len(baseValues))
	for id := range baseValues {
		existingIDs = append(existingIDs, id)
	}
	sort.Strings(existingIDs)

	fmt.Println("Baseline diff:")
	fmt.Println()
	fmt.Println("Existing SLIs:")
	for _, id := range existingIDs {
		fmt.Printf("  %s\n", id)
	}

	fmt.Println("\nNew measured SLIs:")
	if len(newIDs) == 0 {
		fmt.Println("  (none)")
	}
	for _, id := range newIDs {
		fmt.Printf("  %s\n", id)
		fmt.Println("    Recommendation: review and consider an append-new-only merge.")
	}

	fmt.Println("\nChanged existing SLIs:")
	if len(changedIDs) == 0 {
		fmt.Println("  (none)")
	}
	for _, id := range changedIDs {
		fmt.Printf("  %s: %v → %v\n", id, baseValues[id], curValues[id])
		printChangeGuidance(id, baseValues[id], curValues[id], directions[id])
	}

	fmt.Println("\nRemoved or missing SLIs:")
	if len(missingIDs) == 0 {
		fmt.Println("  (none)")
	}
	for _, id := range missingIDs {
		fmt.Printf("  %s\n", id)
		fmt.Println("    Recommendation: mark stale, do not delete automatically.")
	}

	result := "OK"
	if len(changedIDs) > 0 || len(missingIDs) > 0 {
		result = "REVIEW_REQUIRED"
	}
	fmt.Printf("\nResult:\n  %s\n", result)

	return nil
}

// loadMetricDirections best-effort loads a policy.yaml and returns, for each
// metric with a threshold rule, whether lower or higher values are better.
// A missing or invalid policy file simply yields an empty map — diff still
// runs, just without direction-aware wording.
func loadMetricDirections(policyPath string) map[string]string {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil
	}
	var doc diffPolicyDoc
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil
	}
	directions := make(map[string]string, len(doc.Thresholds))
	for _, t := range doc.Thresholds {
		if gate.LowerIsBetter(t.Operator) {
			directions[t.Metric] = "lower"
		} else if gate.HigherIsBetter(t.Operator) {
			directions[t.Metric] = "higher"
		}
	}
	return directions
}

func printChangeGuidance(id string, baseVal, curVal float64, direction string) {
	switch direction {
	case "lower":
		if curVal > baseVal {
			fmt.Println("    Recommendation: do not merge automatically.")
			fmt.Println("    Reason: this weakens the known-good baseline.")
			return
		}
		fmt.Println("    Recommendation: an improvement — 'baseline merge --mode review-existing' will apply it.")
	case "higher":
		if curVal < baseVal {
			fmt.Println("    Recommendation: do not merge automatically.")
			fmt.Println("    Reason: this weakens the known-good baseline.")
			return
		}
		fmt.Println("    Recommendation: an improvement — 'baseline merge --mode review-existing' will apply it.")
	default:
		fmt.Printf("    Recommendation: review before merging (improve/weaken direction for %q is unknown — no policy threshold for it, or its operator doesn't imply a direction).\n", id)
	}
}
