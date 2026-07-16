package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"gopkg.in/yaml.v3"
)

const inspectUsage = `Usage: slint-gate inspect [flags]

Explains which shift-left SLIs were measured in a sli-summary.json, which
expected SLIs are missing, and whether the measurement is ready for a
threshold policy and a regression baseline.

Flags:
`

func runInspect(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, inspectUsage)
		fs.PrintDefaults()
	}
	summaryPath := fs.String("summary", "artifacts/sli-summary.json", "Path to the measurement summary JSON")
	policyPath := fs.String("policy", ".slint/policy.yaml", "Path to policy YAML for advisory coverage diagnostics")
	profile := fs.String("profile", "kubebuilder-operator", "Onboarding profile to check measured SLIs against")
	profileFile := fs.String("profile-file", "", "Path to a local custom profile file (overrides --profile)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	candidates, _, err := resolveProfileCandidates(*profileFile, strings.TrimSpace(*profile))
	if err != nil {
		return err
	}

	s, err := summary.LoadFile(*summaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, `kube-slint could not inspect this summary.

Reason:
  %s

What this means:
  Your E2E test may have passed, but kube-slint could not read a trustworthy
  shift-left SLI measurement from %q.

How to fix:
  1. Confirm the path is correct and the file was written by Session.End().
  2. Re-run the E2E test with kube-slint attached.

Result:
  NO_GRADE
`, err, *summaryPath)
		return err
	}

	measured := make(map[string]*summary.SLIResult, len(s.Results))
	for i := range s.Results {
		measured[s.Results[i].ID] = &s.Results[i]
	}

	fmt.Println("Measured shift-left SLIs:")
	var missing []profileCandidate
	measuredCount := 0
	for _, c := range candidates {
		r, ok := measured[c.ID]
		if !ok || r.Value == nil {
			missing = append(missing, c)
			continue
		}
		measuredCount++
		usable := "usable for threshold + regression"
		switch c.Tier {
		case tierNoisy:
			usable = "usable, but may be CI-environment sensitive"
		case tierInformational:
			usable = "measured, informational only (no default threshold)"
		}
		fmt.Printf("  %-32s %-12v %s\n", c.ID, *r.Value, usable)
	}

	fmt.Println("\nMissing profile SLIs:")
	if len(missing) == 0 {
		fmt.Println("  (none)")
	}
	for _, c := range missing {
		fmt.Printf("  %-32s missing metric\n", c.ID)
		fmt.Println("    Recommendation: keep this SLI commented out for now.")
	}

	printPolicyCoverage(*policyPath, s)

	collectionStatus := "unknown"
	if s.Reliability != nil && s.Reliability.CollectionStatus != "" {
		collectionStatus = s.Reliability.CollectionStatus
	}

	fmt.Println("\nReadiness:")
	if measuredCount > 0 {
		fmt.Println("  Threshold policy: ready")
	} else {
		fmt.Println("  Threshold policy: not ready — no profile SLIs were measured")
	}
	if measuredCount > 0 && !strings.EqualFold(collectionStatus, "Failed") {
		fmt.Println("  Baseline approval: ready")
	} else {
		fmt.Println("  Baseline approval: not ready")
	}
	fmt.Println("  Regression gate: not enabled yet")
	fmt.Printf("  Measurement confidence: %s\n", strings.ToLower(collectionStatus))

	fmt.Println(`
What this means:
  kube-slint collected enough development-time operational signal to build
  or refine a CI gate for this test run.

Next:
  slint-gate recommend-policy --summary ` + *summaryPath + ` --profile ` + *profile)

	return nil
}

func printPolicyCoverage(policyPath string, s summary.Summary) {
	coverage, err := loadInspectPolicyCoverage(policyPath)

	fmt.Println("\nPolicy coverage:")
	if err != nil {
		fmt.Printf("  policy not loaded: %s\n", err)
		fmt.Println("  Recommendation: run recommend-policy or pass --policy to inspect coverage.")
		return
	}

	measured := measuredValueIDs(s)
	uncovered := difference(measured, coverage.metrics)
	policyMissing := difference(coverage.metrics, measured)

	if len(uncovered) == 0 {
		fmt.Println("  Measured but not covered by policy: (none)")
	} else {
		fmt.Println("  Measured but not covered by policy:")
		for _, id := range uncovered {
			fmt.Printf("    %-32s advisory: add a threshold/regression rule or mark informational\n", id)
		}
	}

	if len(policyMissing) == 0 {
		fmt.Println("  Policy-covered but missing from summary: (none)")
	} else {
		fmt.Println("  Policy-covered but missing from summary:")
		for _, id := range policyMissing {
			fmt.Printf("    %-32s policy references an SLI not measured in this run\n", id)
		}
	}

	if coverage.regressionEnabled {
		fmt.Println("  Regression coverage: enabled for policy-covered metrics when a baseline is supplied.")
	} else {
		fmt.Println("  Regression coverage: disabled in policy.")
	}
}

type inspectPolicyCoverage struct {
	metrics           map[string]bool
	regressionEnabled bool
}

func loadInspectPolicyCoverage(path string) (inspectPolicyCoverage, error) {
	if strings.TrimSpace(path) == "" {
		return inspectPolicyCoverage{}, fmt.Errorf("empty --policy path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return inspectPolicyCoverage{}, fmt.Errorf("%s does not exist", path)
		}
		return inspectPolicyCoverage{}, fmt.Errorf("could not read %s: %w", path, err)
	}
	var policy gate.Policy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return inspectPolicyCoverage{}, fmt.Errorf("could not parse %s: %w", path, err)
	}
	if strings.TrimSpace(policy.SchemaVersion) != "slint.policy.v1" {
		return inspectPolicyCoverage{}, fmt.Errorf("%s has unsupported schema_version %q", path, policy.SchemaVersion)
	}
	metrics := make(map[string]bool, len(policy.Thresholds))
	for _, rule := range policy.Thresholds {
		metric := strings.TrimSpace(rule.Metric)
		if metric != "" {
			metrics[metric] = true
		}
	}
	if len(metrics) == 0 {
		return inspectPolicyCoverage{}, fmt.Errorf("%s has no threshold metrics", path)
	}
	return inspectPolicyCoverage{metrics: metrics, regressionEnabled: policy.Regression.Enabled}, nil
}

func measuredValueIDs(s summary.Summary) map[string]bool {
	out := make(map[string]bool, len(s.Results))
	for _, r := range s.Results {
		if r.Value == nil {
			continue
		}
		id := strings.TrimSpace(r.ID)
		if id != "" {
			out[id] = true
		}
	}
	return out
}

func difference(left, right map[string]bool) []string {
	var out []string
	for id := range left {
		if !right[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
