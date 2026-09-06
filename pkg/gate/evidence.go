package gate

import (
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// evidenceIndex answers positive per-SLI evidence-sufficiency questions from
// TYPED measurement facts only — value presence, the skipped-SLI set, and
// per-SLI missing inputs — never from the producer's diagnostic status verdict
// (KSL-T2/T3). A check may grade an SLI only when this index proves the evidence
// is present and reliable; otherwise the check is NO_GRADE. Because the judgment
// is per-SLI, an unrelated insufficient SLI cannot poison an independently
// sufficient one.
type evidenceIndex struct {
	byID    map[string]summary.SLIResult
	skipped map[string]bool
	// collectionFailed is true when the measurement's reliability record reports a
	// failed collection: the whole run's values are untrustworthy, so NO SLI has
	// sufficient evidence to grade, regardless of an individual value being present.
	collectionFailed bool
}

func newEvidenceIndex(s *summary.Summary) evidenceIndex {
	idx := evidenceIndex{byID: map[string]summary.SLIResult{}, skipped: map[string]bool{}}
	if s == nil {
		return idx
	}
	for _, r := range s.Results {
		idx.byID[r.ID] = r
	}
	if s.Reliability != nil {
		for _, id := range s.Reliability.SkippedSLIs {
			idx.skipped[id] = true
		}
		idx.collectionFailed = strings.EqualFold(strings.TrimSpace(s.Reliability.CollectionStatus), "Failed")
	}
	return idx
}

// valueSufficient reports whether the SLI referenced by id has positively
// sufficient, reliable evidence to support a graded numeric comparison. It
// returns (value, true, "") when sufficient and (0, false, reason) otherwise.
// The judgment uses only typed facts, so a producer status verdict can neither
// promote unreliable evidence into a grade nor demote reliable evidence.
func (e evidenceIndex) valueSufficient(id string) (float64, bool, string) {
	if e.collectionFailed {
		// The whole collection failed: every value is untrustworthy, so no check
		// may grade on it (matches the collection-wide reliability NO_GRADE).
		return 0, false, reasonEvidenceInsufficient
	}
	r, ok := e.byID[id]
	if !ok {
		// The referenced SLI was not produced at all: nothing to grade against.
		return 0, false, reasonMeasInputMissing
	}
	if e.skipped[id] {
		// Positively marked as skipped by the measurement's own reliability record.
		return 0, false, reasonEvidenceInsufficient
	}
	if len(r.InputsMissing) > 0 {
		// The producer recorded that a required input for this SLI was missing,
		// so its value is not a trustworthy basis for a grade.
		return 0, false, reasonEvidenceInsufficient
	}
	if r.Value == nil {
		// No numeric value was recorded (e.g. a skip): a numeric check cannot grade.
		return 0, false, reasonEvidenceInsufficient
	}
	return *r.Value, true, ""
}
