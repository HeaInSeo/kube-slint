package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const ciGithubActionsUsage = `Usage: slint-gate ci github-actions [flags]

Prints a GitHub Actions step snippet wired to the local summary/policy/
baseline paths, ready to paste into a workflow.

Flags:
`

func runCIGithubActions(args []string) error {
	fs := flag.NewFlagSet("ci github-actions", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, ciGithubActionsUsage)
		fs.PrintDefaults()
	}
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to the policy YAML")
	baselinePath := fs.String("baseline", "", "Optional path to a baseline summary JSON")
	actionRef := fs.String("action-ref", Version, "Tag or SHA to pin the slint-gate composite action to")
	exitOnMode := fs.String("exit-on-mode", "FAIL_OR_NOGRADE", "exit-on value for the generated step: NEVER | FAIL | FAIL_OR_WARN | FAIL_OR_NOGRADE | FAIL_WARN_OR_NOGRADE")

	if err := fs.Parse(args); err != nil {
		return err
	}

	mode := strings.ToUpper(strings.TrimSpace(*exitOnMode))
	if !isValidExitOn(mode) {
		return fmt.Errorf("invalid --exit-on-mode: %s", *exitOnMode)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "- name: Run kube-slint shift-left SLI gate\n")
	fmt.Fprintf(&b, "  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@%s\n", *actionRef)
	fmt.Fprintf(&b, "  with:\n")
	fmt.Fprintf(&b, "    measurement-summary: %s\n", *summaryPath)
	fmt.Fprintf(&b, "    policy: %s\n", *policyPath)
	if baseline := strings.TrimSpace(*baselinePath); baseline != "" {
		fmt.Fprintf(&b, "    baseline: %s\n", baseline)
	}
	fmt.Fprintf(&b, "    exit-on: %s\n", mode)

	fmt.Print(b.String())
	return nil
}
