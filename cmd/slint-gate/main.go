package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/internal/gate"
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
		}
	}
	runGate()
}

func runGate() {
	measurementPath := flag.String("measurement-summary", "artifacts/sli-summary.json", "Path to measurement summary JSON")
	policyPath := flag.String("policy", ".slint/policy.yaml", "Path to policy YAML")
	baselinePath := flag.String("baseline", "", "Optional path to baseline summary JSON")
	outputPath := flag.String("output", "slint-gate-summary.json", "Output path for gate summary JSON")
	githubStepSummary := flag.Bool("github-step-summary", false, "Append markdown result to $GITHUB_STEP_SUMMARY")
	failOn := flag.String("fail-on", "NEVER",
		"Gate result level that causes non-zero exit.\n"+
			"  NEVER               — always exit 0; let the caller inspect gate_result (default)\n"+
			"  FAIL                — exit 1 on hard policy violations\n"+
			"  FAIL_OR_WARN        — treat WARN as failure too\n"+
			"  FAIL_OR_NOGRADE     — treat NO_GRADE as failure too\n"+
			"  FAIL_WARN_OR_NOGRADE — treat WARN and NO_GRADE as failures")
	flag.Parse()

	failOnValue := strings.ToUpper(strings.TrimSpace(*failOn))
	if !isValidFailOn(failOnValue) {
		fmt.Fprintf(os.Stderr, "invalid --fail-on: %s\n", *failOn)
		os.Exit(2)
	}

	result := gate.Evaluate(gate.Request{
		MeasurementPath: *measurementPath,
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

	if shouldFailOn(result.GateResult, failOnValue) {
		fmt.Fprintf(os.Stderr, "slint-gate: exit 1 (gate_result=%s, fail-on=%s)\n", result.GateResult, *failOn)
		os.Exit(1)
	}
}

func isValidFailOn(failOn string) bool {
	switch failOn {
	case "NEVER", "", "FAIL", "FAIL_OR_WARN", "FAIL_OR_NOGRADE", "FAIL_WARN_OR_NOGRADE":
		return true
	default:
		return false
	}
}

// shouldFailOn returns true when gateResult meets the failOn threshold.
func shouldFailOn(gateResult, failOn string) bool {
	switch failOn {
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
