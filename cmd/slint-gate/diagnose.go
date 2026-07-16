package main

import (
	"fmt"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
)

// diagEntry is a diagnostic entry for one reason code.
type diagEntry struct {
	summary string
	hints   []string
}

// diagMessages maps reason codes to human-readable diagnostic messages.
var diagMessages = map[string]diagEntry{
	"MEASUREMENT_INPUT_MISSING": {
		summary: "sli-summary.json is missing or empty.",
		hints: []string{
			"Run slint-gate only after the E2E test (sess.End(ctx)) has completed.",
			"Check that SessionConfig.ArtifactsDir points to the right path.",
			"Check the E2E logs for a 'fetch failed' message.",
			"Check RBAC:\n    kubectl auth can-i create pods" +
				" --as=system:serviceaccount:<ns>:<sa> -n <ns>",
		},
	},
	"MEASUREMENT_INPUT_CORRUPT": {
		summary: "sli-summary.json could not be parsed.",
		hints: []string{
			"Check that the file is complete JSON: cat artifacts/sli-summary.json | python3 -m json.tool",
			"The E2E test may have been interrupted mid-run, leaving the file incomplete.",
		},
	},
	"POLICY_MISSING": {
		summary: ".slint/policy.yaml is missing.",
		hints: []string{
			"Generate a default policy.yaml:\n    slint-gate init",
			"Or point directly at a path with --policy:\n    slint-gate --policy path/to/policy.yaml",
		},
	},
	"POLICY_INVALID": {
		summary: "policy.yaml is invalid.",
		hints: []string{
			"Check for YAML syntax errors: cat .slint/policy.yaml",
			"Check that schema_version is present and exactly \"slint.policy.v1\" (missing or any other value is treated as invalid):\n    schema_version: slint.policy.v1",
			"Check for unsupported values in promote_to_fail/fail_on. Supported values: threshold_miss, regression_detected, coverage_gap",
			"Check that reliability.min_level is partial or complete.",
			"Check for an unsupported operator (e.g. !=) in an operator field.",
			"Supported operators: <=, >=, <, >, ==",
		},
	},
	"THRESHOLD_MISS": {
		summary: "One or more threshold conditions were violated.",
		hints: []string{
			"Check slint-gate-summary.json's checks entries for any in a fail status.",
			"Adjust the threshold or review the operator's behavior.",
		},
	},
	"REGRESSION_DETECTED": {
		summary: "A metric exceeded the allowed tolerance versus the baseline.",
		hints: []string{
			"Check slint-gate-summary.json's regression check entries.",
			"If the change is expected, update the baseline:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
			"If it's transient variance, adjust tolerance_percent in policy.yaml.",
		},
	},
	"BASELINE_ABSENT_FIRST_RUN": {
		summary: "No baseline exists (first run). Regression comparison is skipped.",
		hints: []string{
			"This is a warning; it's expected on the first run.",
			"To save a baseline:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
			"To proceed without a baseline, set regression.enabled: false in policy.yaml.",
		},
	},
	"BASELINE_UNAVAILABLE": {
		summary: "The file specified via --baseline could not be found.",
		hints: []string{
			"Check that the path passed to --baseline is correct.",
			"If you don't have a baseline yet, omit the --baseline flag.",
		},
	},
	"BASELINE_CORRUPT": {
		summary: "The baseline file could not be parsed.",
		hints: []string{
			"Check that the baseline file is complete JSON.",
			"Refresh the baseline:\n    make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json",
		},
	},
	"RELIABILITY_INSUFFICIENT": {
		summary: "Metric collection reliability is below policy.yaml's minimum required level.",
		hints: []string{
			"Check the E2E logs for fetch errors.",
			"To allow this, set reliability.required: false in policy.yaml.",
		},
	},
}

// printDiagnostics prints a diagnostic message to stdout when the gate
// result is not PASS.
func printDiagnostics(result *gate.Summary) {
	if result.GateResult == gate.GatePass {
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n── Diagnostics ─────────────────────────────────────────────────────────\n")
	fmt.Fprintf(&sb, "Gate Result: %s\n", result.GateResult)

	if len(result.Reasons) == 0 {
		fmt.Fprintf(&sb, "────────────────────────────────────────────────────────────────────────\n")
		fmt.Print(sb.String())
		return
	}

	for _, reason := range result.Reasons {
		entry, ok := diagMessages[reason]
		if !ok {
			fmt.Fprintf(&sb, "\n[%s]\n  (no additional diagnostic info)\n", reason)
			continue
		}
		fmt.Fprintf(&sb, "\n[%s]\n  %s\n", reason, entry.summary)
		for _, hint := range entry.hints {
			indented := "  → " + strings.ReplaceAll(hint, "\n", "\n    ")
			fmt.Fprintf(&sb, "%s\n", indented)
		}
	}
	fmt.Fprintf(&sb, "────────────────────────────────────────────────────────────────────────\n")
	fmt.Print(sb.String())
}
