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
	measurementPath := flag.String("measurement-summary", "artifacts/sli-summary.json", "Path to measurement summary JSON")
	policyPath := flag.String("policy", ".slint/policy.yaml", "Path to policy YAML")
	baselinePath := flag.String("baseline", "", "Optional path to baseline summary JSON")
	outputPath := flag.String("output", "slint-gate-summary.json", "Output path for gate summary JSON")
	githubStepSummary := flag.Bool("github-step-summary", false, "Append markdown result to $GITHUB_STEP_SUMMARY")
	flag.Parse()

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

	if *githubStepSummary {
		if err := renderGitHubStepSummary(result); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write GitHub step summary: %v\n", err)
		}
	}
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
				c.Name, c.Category, c.Status, c.Metric, obs, c.Expected, c.Message)
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
