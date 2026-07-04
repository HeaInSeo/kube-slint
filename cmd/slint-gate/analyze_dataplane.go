package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/dataplane/service"
	"github.com/HeaInSeo/kube-slint/pkg/report"
)

const analyzeDataplaneUsage = `Usage: slint-gate analyze-dataplane [flags] <manifest-dir>

Statically analyzes a directory of Kubernetes YAML manifests (Deployment/
StatefulSet/DaemonSet/Service/ServiceMonitor) for observability-contract
compliance: metrics port naming, /livez /readyz probe path convention,
metrics Service/ServiceMonitor wiring, and explicit
terminationGracePeriodSeconds.

This intentionally does not check for missing probes or missing resource
requests/limits — pair this with kube-linter (or similar) for that, rather
than duplicating its no-liveness-probe/no-readiness-probe/
unset-cpu-requirements/unset-memory-requirements checks here.

Note: <manifest-dir> is a positional argument and must come AFTER all flags
(standard Go flag-parsing behavior — parsing stops at the first non-flag
argument).

Flags:
`

func runAnalyzeDataplane(args []string) {
	fs := flag.NewFlagSet("analyze-dataplane", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, analyzeDataplaneUsage)
		fs.PrintDefaults()
	}

	outputJSON := fs.String("output-json", "dataplane-report.json", "Output path for JSON report")
	outputSARIF := fs.String("output-sarif", "", "Optional output path for SARIF 2.1.0 report (empty = skip)")
	githubStepSummary := fs.Bool("github-step-summary", false, "Append markdown result table to $GITHUB_STEP_SUMMARY")
	failOn := fs.String("fail-on", "error",
		"Finding severity that causes non-zero exit.\n"+
			"  none    — always exit 0\n"+
			"  error   — exit 1 if any error-severity finding exists (default)\n"+
			"  warning — exit 1 if any error OR warning-severity finding exists")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}

	failOnValue := strings.ToLower(strings.TrimSpace(*failOn))
	if !isValidDataplaneFailOn(failOnValue) {
		fmt.Fprintf(os.Stderr, "invalid --fail-on: %s\n", *failOn)
		os.Exit(2)
	}

	if fs.NArg() != 1 {
		fs.Usage()
		fmt.Fprintf(os.Stderr, "\nerror: expected exactly one manifest directory argument, got %d\n", fs.NArg())
		os.Exit(2)
	}
	manifestDir := fs.Arg(0)

	rep, warnings, err := service.Analyze(manifestDir, Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slint-gate analyze-dataplane: %v\n", err)
		os.Exit(2)
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "slint-gate analyze-dataplane: warning: %s\n", w)
	}

	if err := report.WriteJSON(*outputJSON, rep); err != nil {
		fmt.Fprintf(os.Stderr, "error writing JSON report: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(*outputJSON)

	if sarifPath := strings.TrimSpace(*outputSARIF); sarifPath != "" {
		if err := report.WriteSARIF(sarifPath, rep); err != nil {
			fmt.Fprintf(os.Stderr, "error writing SARIF report: %v\n", err)
			os.Exit(2)
		}
		fmt.Println(sarifPath)
	}

	printDataplaneDiagnostics(rep)

	if *githubStepSummary {
		if err := report.WriteGitHubStepSummary(rep); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write GitHub step summary: %v\n", err)
		}
	}

	if shouldFailOnDataplane(rep.Summary, failOnValue) {
		fmt.Fprintf(os.Stderr, "slint-gate analyze-dataplane: exit 1 (errors=%d warnings=%d, fail-on=%s)\n",
			rep.Summary.ErrorCount, rep.Summary.WarningCount, *failOn)
		os.Exit(1)
	}
}

func printDataplaneDiagnostics(rep *report.Report) {
	fmt.Printf("dataplane-service: %d error(s), %d warning(s), %d rule(s) run\n",
		rep.Summary.ErrorCount, rep.Summary.WarningCount, rep.Summary.RulesRun)
}

func isValidDataplaneFailOn(v string) bool {
	switch v {
	case "none", "error", "warning":
		return true
	default:
		return false
	}
}

// shouldFailOnDataplane returns true when s meets the failOn threshold.
func shouldFailOnDataplane(s report.Summary, failOn string) bool {
	switch failOn {
	case "error":
		return s.ErrorCount > 0
	case "warning":
		return s.ErrorCount > 0 || s.WarningCount > 0
	default: // "none"
		return false
	}
}
