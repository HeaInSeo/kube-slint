package gate

import (
	"fmt"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// runReliability checks the measurement's collection reliability against policy.
//
// A CollectionStatus of "Failed" means the measurement never actually
// completed (both fetch attempts failed or the start snapshot never
// happened) — this can never support a trustworthy PASS/WARN/FAIL decision,
// so it is promoted to NO_GRADE unconditionally, regardless of
// reliability.required. reliability.required only governs the softer
// Partial-vs-Complete minimum-level check, which remains warn-only.
func runReliability(out *Summary, policy *Policy, s *summary.Summary) (anyWarn, anyNoGrade bool) {
	minLevel := strings.ToLower(strings.TrimSpace(policy.Reliability.MinLevel))
	if minLevel == "" {
		minLevel = "partial"
	}
	requiredRank := 1
	if minLevel == "complete" {
		requiredRank = 2
	}

	collectionStatus := ""
	if s != nil && s.Reliability != nil {
		collectionStatus = s.Reliability.CollectionStatus
	}

	check := Check{
		Name:     "reliability-minimum",
		Category: "reliability",
		Status:   "pass",
		Metric:   "reliability.collectionStatus",
		Observed: nilIfEmpty(collectionStatus),
		Expected: fmt.Sprintf(">= %s", minLevel),
		Message:  "reliability requirement satisfied",
	}

	if strings.EqualFold(collectionStatus, "Failed") {
		check.Status = "no_grade"
		check.Message = "collection failed; measurement is not trustworthy"
		addReason(&out.Reasons, reasonCollectionFailed)
		out.MeasurementStatus = measInsufficient
		out.Checks = append(out.Checks, check)
		return false, true
	}

	if policy.Reliability.Required && reliabilityRank(collectionStatus) < requiredRank {
		check.Status = "warn"
		check.Message = "reliability below required level"
		addReason(&out.Reasons, reasonReliabilityInsufficient)
		out.MeasurementStatus = measInsufficient
		anyWarn = true
	}
	out.Checks = append(out.Checks, check)
	return anyWarn, false
}

func reliabilityRank(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete":
		return 2
	case "partial":
		return 1
	default:
		return 0
	}
}
