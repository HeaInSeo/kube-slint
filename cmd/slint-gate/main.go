package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-version":
			fmt.Printf("slint-gate %s\n", Version)
			return
		case "init":
			if err := runInit(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "slint-gate init: %v\n", err)
				os.Exit(1)
			}
			return
		case "analyze-dataplane":
			runAnalyzeDataplane(os.Args[2:])
			return
		case "inspect":
			if err := runInspect(os.Args[2:]); err != nil {
				os.Exit(1)
			}
			return
		case "recommend-policy":
			if err := runRecommendPolicy(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "slint-gate recommend-policy: %v\n", err)
				os.Exit(1)
			}
			return
		case "baseline":
			dispatchBaseline(os.Args[2:])
			return
		case "quickstart":
			if err := runQuickstart(os.Args[2:]); err != nil {
				os.Exit(1)
			}
			return
		case "wizard":
			if err := runWizard(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "slint-gate wizard: %v\n", err)
				os.Exit(1)
			}
			return
		case "ci":
			if len(os.Args) < 3 || os.Args[2] != "github-actions" {
				fmt.Fprintln(os.Stderr, "usage: slint-gate ci github-actions [flags]")
				os.Exit(2)
			}
			if err := runCIGithubActions(os.Args[3:]); err != nil {
				fmt.Fprintf(os.Stderr, "slint-gate ci github-actions: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}
	runGate()
}

// dispatchBaseline handles `slint-gate baseline <approve|diff|merge> [flags]`.
func dispatchBaseline(args []string) {
	usage := "usage: slint-gate baseline <approve|diff|merge> [flags]"
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch args[0] {
	case "approve":
		err = runBaselineApprove(args[1:])
	case "diff":
		err = runBaselineDiff(args[1:])
	case "merge":
		err = runBaselineMerge(args[1:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "slint-gate baseline %s: %v\n", args[0], err)
		os.Exit(1)
	}
}

func runGate() {
	summaryPath := flag.String("summary", "", "Path to measurement summary JSON")
	// Deprecated: every onboarding subcommand (inspect, recommend-policy,
	// baseline diff/approve/merge, ci github-actions) uses --summary for
	// this same artifact — this flag predates that convention. Kept as a
	// working alias so existing invocations don't break.
	measurementPath := flag.String("measurement-summary", "", "Deprecated: use --summary instead. Same meaning.")
	policyPath := flag.String("policy", ".slint/policy.yaml", "Path to policy YAML")
	baselinePath := flag.String("baseline", "", "Optional path to baseline summary JSON")
	outputPath := flag.String("output", "slint-gate-summary.json", "Output path for gate summary JSON")
	githubStepSummary := flag.Bool("github-step-summary", false, "Append markdown result to $GITHUB_STEP_SUMMARY")
	exitOn := flag.String("exit-on", "",
		"Gate result level that causes non-zero exit.\n"+
			"  NEVER               — always exit 0; let the caller inspect gate_result (default)\n"+
			"  FAIL                — exit 1 on hard policy violations\n"+
			"  FAIL_OR_WARN        — treat WARN as failure too\n"+
			"  FAIL_OR_NOGRADE     — treat NO_GRADE as failure too\n"+
			"  FAIL_WARN_OR_NOGRADE — treat WARN and NO_GRADE as failures")
	failOn := flag.String("fail-on", "NEVER", "Deprecated: use --exit-on instead. Same values as --exit-on.")
	flag.Parse()

	exitOnSet, failOnSet := false, false
	summarySet, measurementSet := false, false
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "exit-on":
			exitOnSet = true
		case "fail-on":
			failOnSet = true
		case "summary":
			summarySet = true
		case "measurement-summary":
			measurementSet = true
		}
	})

	resolved, deprecated := resolveExitOn(exitOnSet, *exitOn, failOnSet, *failOn)
	if deprecated {
		fmt.Fprintln(os.Stderr, "slint-gate: --fail-on is deprecated; use --exit-on instead (still honored)")
	}

	exitOnValue := strings.ToUpper(strings.TrimSpace(resolved))
	if !isValidExitOn(exitOnValue) {
		fmt.Fprintf(os.Stderr, "invalid --exit-on: %s\n", resolved)
		os.Exit(2)
	}

	resolvedSummary, summaryDeprecated := resolveSummaryPath(summarySet, *summaryPath, measurementSet, *measurementPath)
	if summaryDeprecated {
		fmt.Fprintln(os.Stderr, "slint-gate: --measurement-summary is deprecated; use --summary instead (still honored)")
	}

	result := gate.Evaluate(gate.Request{
		MeasurementPath: resolvedSummary,
		PolicyPath:      *policyPath,
		BaselinePath:    strings.TrimSpace(*baselinePath),
	})

	if err := writeJSON(*outputPath, result); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(*outputPath)

	printDiagnostics(result)

	if *githubStepSummary {
		if err := renderGitHubStepSummary(result); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write GitHub step summary: %v\n", err)
		}
	}

	if shouldExitOn(result.GateResult, exitOnValue) {
		fmt.Fprintf(os.Stderr, "slint-gate: exit 1 (gate_result=%s, exit-on=%s)\n", result.GateResult, resolved)
		os.Exit(1)
	}
}

// resolveExitOn applies --exit-on/--fail-on precedence: an explicitly-passed
// --exit-on always wins; otherwise an explicitly-passed --fail-on is honored
// (and flagged as deprecated); otherwise the default is "NEVER".
func resolveExitOn(exitOnSet bool, exitOnVal string, failOnSet bool, failOnVal string) (resolved string, deprecated bool) {
	if exitOnSet {
		return exitOnVal, false
	}
	if failOnSet {
		return failOnVal, true
	}
	return failOnVal, false
}

// resolveSummaryPath applies --summary/--measurement-summary precedence: an
// explicitly-passed --summary always wins; otherwise an explicitly-passed
// --measurement-summary is honored (and flagged as deprecated); otherwise
// the default is "artifacts/sli-summary.json".
func resolveSummaryPath(summarySet bool, summaryVal string, measurementSet bool, measurementVal string) (resolved string, deprecated bool) {
	if summarySet {
		return summaryVal, false
	}
	if measurementSet {
		return measurementVal, true
	}
	return "artifacts/sli-summary.json", false
}

func isValidExitOn(exitOn string) bool {
	switch exitOn {
	case "NEVER", "", "FAIL", "FAIL_OR_WARN", "FAIL_OR_NOGRADE", "FAIL_WARN_OR_NOGRADE":
		return true
	default:
		return false
	}
}

// shouldExitOn returns true when gateResult meets the exitOn threshold.
func shouldExitOn(gateResult, exitOn string) bool {
	switch exitOn {
	case "NEVER", "":
		return false
	case "FAIL":
		return gateResult == gate.GateFail
	case "FAIL_OR_WARN":
		return gateResult == gate.GateFail || gateResult == gate.GateWarn
	case "FAIL_OR_NOGRADE":
		return gateResult == gate.GateFail || gateResult == gate.GateNoGrade
	case "FAIL_WARN_OR_NOGRADE":
		return gateResult == gate.GateFail || gateResult == gate.GateWarn || gateResult == gate.GateNoGrade
	default:
		return false
	}
}

func mdCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}
	return f.Close()
}

func renderGitHubStepSummary(result *gate.Summary) error {
	sumPath := os.Getenv("GITHUB_STEP_SUMMARY")
	if sumPath == "" {
		return nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# slint-gate Result\n\n")
	fmt.Fprintf(&sb, "- **Gate Result:** %s\n", result.GateResult)
	fmt.Fprintf(&sb, "- **Evaluation Status:** %s\n", result.EvaluationStatus)
	fmt.Fprintf(&sb, "- **Measurement Status:** %s\n", result.MeasurementStatus)
	fmt.Fprintf(&sb, "- **Baseline Status:** %s\n", result.BaselineStatus)
	fmt.Fprintf(&sb, "- **Policy Status:** %s\n", result.PolicyStatus)
	fmt.Fprintf(&sb, "- **Overall Message:** %s\n\n", result.OverallMessage)

	fmt.Fprintf(&sb, "## Reasons\n")
	if len(result.Reasons) == 0 {
		fmt.Fprintf(&sb, "- (none)\n")
	} else {
		for _, r := range result.Reasons {
			fmt.Fprintf(&sb, "- `%s`\n", r)
		}
	}

	fmt.Fprintf(&sb, "\n## Checks\n")
	if len(result.Checks) == 0 {
		fmt.Fprintf(&sb, "- (no checks)\n")
	} else {
		fmt.Fprintf(&sb, "| Name | Category | Status | Metric | Observed | Expected | Message |\n")
		fmt.Fprintf(&sb, "|---|---|---|---|---|---|---|\n")
		for _, c := range result.Checks {
			obs := ""
			if c.Observed != nil {
				obs = fmt.Sprintf("%v", c.Observed)
			}
			fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s | %s |\n",
				mdCell(c.Name), mdCell(c.Category), mdCell(c.Status), mdCell(c.Metric), mdCell(obs), mdCell(c.Expected), mdCell(c.Message))
		}
	}

	f, err := os.OpenFile(sumPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(sb.String())
	if err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
