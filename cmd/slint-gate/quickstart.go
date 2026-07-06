package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
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

	hasPolicy := fileExists(*policyPath)

	summaryFileExists := fileExists(*summaryPath)
	s, summaryErr := summary.LoadFile(*summaryPath)
	// A summary that exists but fails to load is a different problem
	// ("it's broken, go inspect it") than one that was never produced yet
	// ("go run your E2E test") -- both are hasSummary=false, but nextCommand
	// needs to tell them apart.
	summaryInvalid := summaryFileExists && summaryErr != nil
	hasSummary := summaryErr == nil

	var gateResult string
	if hasPolicy && hasSummary {
		result := gate.Evaluate(gate.Request{
			MeasurementPath: *summaryPath,
			PolicyPath:      *policyPath,
		})
		gateResult = result.GateResult
	}

	baselineRequested := strings.TrimSpace(*baselinePath) != ""
	hasBaseline := baselineRequested && fileExists(*baselinePath)

	fmt.Println("Status:")
	fmt.Printf("  Policy:      %s\n", statusLine(hasPolicy, *policyPath))
	fmt.Printf("  Measurement: %s\n", measurementStatusLine(hasSummary, summaryErr, s, *summaryPath))
	if hasPolicy && hasSummary {
		fmt.Printf("  Gate:        %s\n", gateResult)
	} else {
		fmt.Println("  Gate:        not evaluated yet")
	}
	if baselineRequested {
		fmt.Printf("  Baseline:    %s\n", statusLine(hasBaseline, *baselinePath))
	} else {
		fmt.Println("  Baseline:    not yet approved (or --baseline not given)")
	}

	fmt.Println("\nNext:")
	fmt.Println("  " + nextCommand(hasPolicy, hasSummary, summaryInvalid, gateResult, hasBaseline, *summaryPath, *policyPath, *baselinePath))

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

// nextCommand picks a single next step, following the precedence: get a
// policy in place; get a measurement in place; make it PASS; approve a
// baseline; wire CI.
func nextCommand(hasPolicy, hasSummary, summaryInvalid bool, gateResult string, hasBaseline bool, summaryPath, policyPath, baselinePath string) string {
	switch {
	case summaryInvalid:
		return fmt.Sprintf("slint-gate inspect --summary %s", summaryPath)
	case !hasPolicy && !hasSummary:
		return "slint-gate init --profile kubebuilder-operator"
	case !hasPolicy:
		return fmt.Sprintf("slint-gate recommend-policy --summary %s --output %s", summaryPath, policyPath)
	case !hasSummary:
		return "run your E2E test with kube-slint attached, then re-run 'slint-gate quickstart'"
	case gateResult != gate.GatePass:
		return fmt.Sprintf("slint-gate inspect --summary %s", summaryPath)
	case !hasBaseline:
		if baselinePath == "" {
			return fmt.Sprintf("slint-gate baseline approve --summary %s --policy %s --output docs/baselines/<name>-sli-summary.json", summaryPath, policyPath)
		}
		return fmt.Sprintf("slint-gate baseline approve --summary %s --policy %s --output %s", summaryPath, policyPath, baselinePath)
	default:
		return fmt.Sprintf("slint-gate ci github-actions --summary %s --policy %s --baseline %s", summaryPath, policyPath, baselinePath)
	}
}
