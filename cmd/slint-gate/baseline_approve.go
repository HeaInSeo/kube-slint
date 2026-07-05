package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

const baselineApproveUsage = `Usage: slint-gate baseline approve [flags]

Evaluates a measurement summary against a policy and, only if it passes,
approves it as the known-good development baseline for future regression
checks. FAIL and NO_GRADE results are never approved.

Flags:
`

func runBaselineApprove(args []string) error {
	fs := flag.NewFlagSet("baseline approve", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, baselineApproveUsage)
		fs.PrintDefaults()
	}
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML")
	output := fs.String("output", "", "Output path for the approved baseline JSON (required)")
	allowWarn := fs.Bool("allow-warn", false, "Also approve a WARN gate result as a baseline")
	force := fs.Bool("force", false, "Overwrite --output if it already exists")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*output) == "" {
		return fmt.Errorf("--output is required")
	}

	result := gate.Evaluate(gate.Request{
		MeasurementPath: *summaryPath,
		PolicyPath:      *policyPath,
	})

	approved := result.GateResult == gate.GatePass || (result.GateResult == gate.GateWarn && *allowWarn)
	if !approved {
		printBaselineRejection(result, *summaryPath, *allowWarn)
		return fmt.Errorf("baseline not approved: gate_result=%s", result.GateResult)
	}

	if _, statErr := os.Stat(*output); statErr == nil && !*force {
		return fmt.Errorf("%s already exists; pass --force to overwrite", *output)
	}

	s, err := summary.LoadFile(*summaryPath)
	if err != nil {
		return fmt.Errorf("could not re-read summary for baseline write: %w", err)
	}
	// EvidencePaths point at local temp-file artifacts from the original test
	// run; once committed as a baseline, those paths are stale and meaningless.
	s.Config.EvidencePaths = nil

	if err := summary.WriteFile(*output, s); err != nil {
		return fmt.Errorf("write baseline file: %w", err)
	}

	printBaselineApproval(result, *summaryPath, *policyPath, *output, s)
	return nil
}

func printBaselineRejection(result *gate.Summary, summaryPath string, allowWarn bool) {
	reason := fmt.Sprintf("The summary produced %s.", result.GateResult)
	howToFix := fmt.Sprintf("  1. Run:\n       slint-gate inspect --summary %s\n"+
		"  2. Review the failed threshold, regression, or reliability check.\n"+
		"  3. Fix the test environment or application behavior and re-run.", summaryPath)
	if result.GateResult == gate.GateWarn && !allowWarn {
		reason = "The summary produced WARN, which is not approved as a baseline by default."
		howToFix = "  1. Review the WARN reason below.\n" +
			"  2. If it's expected (e.g. no prior baseline yet), pass --allow-warn to approve anyway.\n" +
			"  3. Otherwise, fix the underlying cause and re-run."
	}

	fmt.Printf(`Baseline was not approved.

Reason:
  %s

What this means:
  Your E2E test may have passed, but kube-slint did not collect enough
  trustworthy development-time operational signal to create a regression
  baseline from this run.

How to fix:
%s

Result:
  %s
`, reason, howToFix, result.GateResult)
}

func printBaselineApproval(result *gate.Summary, summaryPath, policyPath, output string, s summary.Summary) {
	fmt.Println("Baseline approval review:")
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  %s\n\n", summaryPath)
	fmt.Println("Policy:")
	fmt.Printf("  %s\n\n", policyPath)
	fmt.Println("Gate evaluation:")
	fmt.Printf("  Result: %s\n\n", result.GateResult)

	fmt.Println("Measured shift-left SLIs:")
	for _, r := range s.Results {
		if r.Value == nil {
			continue
		}
		fmt.Printf("  %-32s %v\n", r.ID, *r.Value)
	}

	fmt.Println("\nBaseline output:")
	fmt.Printf("  %s\n\n", output)
	fmt.Println("Approved.")
	fmt.Println(`
What this means:
  This E2E run is now the known-good development baseline for future
  operational regression checks.`)
}
