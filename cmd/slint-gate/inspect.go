package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
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
	profile := fs.String("profile", "kubebuilder-operator", "Onboarding profile to check measured SLIs against")

	if err := fs.Parse(args); err != nil {
		return err
	}

	candidates, err := profileCandidates(strings.TrimSpace(*profile))
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
		if c.Tier == tierNoisy {
			usable = "usable, but may be CI-environment sensitive"
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
