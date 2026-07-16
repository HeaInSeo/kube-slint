package gate

import (
	"fmt"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

const reasonCoverageGap = "COVERAGE_GAP"

func runCoverage(out *Summary, policy *Policy, s *summary.Summary, promote map[string]bool) (failed, warn bool) {
	if policy == nil || s == nil || !policy.Coverage.Required {
		return false, false
	}
	covered := map[string]bool{}
	for _, rule := range policy.Thresholds {
		metric := strings.TrimSpace(rule.Metric)
		if metric != "" {
			covered[metric] = true
		}
	}
	informational := map[string]bool{}
	for _, id := range policy.Coverage.Informational {
		id = strings.TrimSpace(id)
		if id != "" {
			informational[id] = true
		}
	}

	for _, r := range s.Results {
		if r.Value == nil || r.ID == "" || covered[r.ID] || informational[r.ID] {
			continue
		}
		check := Check{
			Name:     fmt.Sprintf("coverage:%s", r.ID),
			Category: "coverage",
			Metric:   r.ID,
			Observed: *r.Value,
			Expected: "threshold rule or coverage.informational entry",
			Message:  "measured SLI is not covered by policy",
		}
		addReason(&out.Reasons, reasonCoverageGap)
		if promote["coverage_gap"] {
			check.Status = "fail"
			failed = true
		} else {
			check.Status = "warn"
			warn = true
		}
		out.Checks = append(out.Checks, check)
	}
	return failed, warn
}
