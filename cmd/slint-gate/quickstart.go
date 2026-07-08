package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

const quickstartUsage = `Usage: slint-gate quickstart [flags]

Read-only status check over the onboarding artifacts (policy, measurement
summary, optional baseline) and a single suggestion for what to run next.
Never gates and never exits non-zero except on a flag error.

Flags:
`

func runQuickstart(args []string) error {
	fs := flag.NewFlagSet("quickstart", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, quickstartUsage)
		fs.PrintDefaults()
	}
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML")
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the measurement summary JSON")
	baselinePath := fs.String("baseline", "", "Optional path to a baseline summary JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	st := inspectOnboardingState(*policyPath, *summaryPath, *baselinePath)

	fmt.Println("Status:")
	fmt.Printf("  Policy:      %s\n", statusLine(st.HasPolicy, *policyPath))
	fmt.Printf("  Measurement: %s\n", measurementStatusLine(st.HasSummary, st.SummaryLoadErr, st.Summary, *summaryPath))
	if st.HasPolicy && st.HasSummary {
		fmt.Printf("  Gate:        %s\n", st.GateResult)
	} else {
		fmt.Println("  Gate:        not evaluated yet")
	}
	if st.BaselineRequested {
		fmt.Printf("  Baseline:    %s\n", statusLine(st.HasBaseline, *baselinePath))
	} else {
		fmt.Println("  Baseline:    not yet approved (or --baseline not given)")
	}

	fmt.Println("\nNext:")
	fmt.Println("  " + nextCommand(st, *summaryPath, *policyPath, *baselinePath))

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func statusLine(ok bool, path string) string {
	if ok {
		return "✓ " + path
	}
	return "✗ not found (" + path + ")"
}

func measurementStatusLine(ok bool, err error, s summary.Summary, path string) string {
	if ok {
		return fmt.Sprintf("✓ %s (%d result(s))", path, len(s.Results))
	}
	return fmt.Sprintf("✗ %s (%v)", path, err)
}

// nextCommand formats nextOnboardingStep(st) into the single suggested
// command quickstart prints. wizard uses nextOnboardingStep directly to act
// on the same decision instead of printing it.
func nextCommand(st onboardingState, summaryPath, policyPath, baselinePath string) string {
	switch nextOnboardingStep(st) {
	case stepInspectInvalidSummary, stepInspectFailedGate:
		return fmt.Sprintf("slint-gate inspect --summary %s", summaryPath)
	case stepInit:
		return "slint-gate init --profile kubebuilder-operator"
	case stepRecommendPolicy:
		return fmt.Sprintf("slint-gate recommend-policy --summary %s --output %s", summaryPath, policyPath)
	case stepRunE2E:
		return "run your E2E test with kube-slint attached, then re-run 'slint-gate quickstart'"
	case stepApproveBaseline:
		if baselinePath == "" {
			return fmt.Sprintf("slint-gate baseline approve --summary %s --policy %s --output docs/baselines/<name>-sli-summary.json", summaryPath, policyPath)
		}
		return fmt.Sprintf("slint-gate baseline approve --summary %s --policy %s --output %s", summaryPath, policyPath, baselinePath)
	default: // stepWireCI
		return fmt.Sprintf("slint-gate ci github-actions --summary %s --policy %s --baseline %s", summaryPath, policyPath, baselinePath)
	}
}
