// Package gate evaluates slint policy.yaml rules (thresholds, regression,
// reliability) against a measured SLI summary and an optional baseline,
// producing a Summary with a PASS/WARN/FAIL/NO_GRADE gate result.
//
// File layout:
//   - types.go: the Policy/Summary/Check schema and gate result constants
//   - policy.go: policy.yaml loading, validation, and fail-promotion rules
//   - measurement.go: measurement/baseline summary JSON loading
//   - threshold.go: absolute threshold rule evaluation
//   - regression.go: baseline-relative regression evaluation
//   - reliability.go: collection-reliability evaluation
//   - gate.go (this file): orchestration (Evaluate) and result aggregation
package gate

import (
	"fmt"
	"os"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// Evaluate runs slint-gate policy evaluation and returns a Summary.
func Evaluate(req Request) *Summary {
	out := newSummary(req)

	policy := initPolicy(out, req.PolicyPath)
	measurement := initMeasurement(out, req.MeasurementPath)
	baseline := initBaseline(out, req.BaselinePath)

	if out.PolicyStatus != policyOK || out.MeasurementStatus != measOK {
		out.OverallMessage = "Policy or measurement input unavailable; gate not evaluated."
		return out
	}

	cur := resultValueMap(measurement)
	base := resultValueMap(baseline)
	promote := makePromotionSet(policy)

	tFailed, tWarn, tNoGrade := runThresholds(out, policy.Thresholds, cur, promote)
	rFailed, rWarn, rNoGrade := runRegression(out, policy, cur, base)
	relWarn, relNoGrade := runReliability(out, policy, measurement)
	rsFailed, rsWarn, rsNoGrade := runResultStatus(out, measurement)

	anyFailed := tFailed || rFailed || rsFailed
	anyNoGrade := tNoGrade || rNoGrade || rsNoGrade || relNoGrade
	hasWarn := rWarn || relWarn || tWarn || rsWarn

	out.EvaluationStatus = computeEvalStatus(out.Checks, anyNoGrade)
	out.GateResult = computeGateResult(anyFailed, hasWarn, anyNoGrade, out.BaselineStatus, policy.Regression.Enabled)
	out.OverallMessage = gateMessage(out.GateResult)
	return out
}

// --- initializers ---

func newSummary(req Request) *Summary {
	var baselineRef *string
	if req.BaselinePath != "" {
		s := req.BaselinePath
		baselineRef = &s
	}
	return &Summary{
		SchemaVersion:     schemaVersion,
		GateResult:        GateNoGrade,
		EvaluationStatus:  evalNot,
		MeasurementStatus: measOK,
		BaselineStatus:    baseAbsentFirst,
		PolicyStatus:      policyOK,
		Reasons:           []string{},
		EvaluatedAt:       time.Now().UTC().Format(time.RFC3339),
		InputRefs: InputRefs{
			MeasurementSummary: req.MeasurementPath,
			PolicyFile:         req.PolicyPath,
			BaselineFile:       baselineRef,
		},
		Checks: []Check{},
	}
}

func initPolicy(out *Summary, path string) *Policy {
	policy, state, warnings := loadPolicy(path)
	if len(warnings) > 0 {
		out.PolicyWarnings = warnings
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "slint-gate: policy warning: %s\n", w)
		}
	}
	switch state {
	case policyMissing:
		out.PolicyStatus = policyMissing
		addReason(&out.Reasons, reasonPolicyMissing)
	case policyInvalid:
		out.PolicyStatus = policyInvalid
		addReason(&out.Reasons, reasonPolicyInvalid)
	}
	return policy
}

func initMeasurement(out *Summary, path string) *summary.Summary {
	s, state := loadMeasurement(path)
	switch state {
	case measMissing:
		out.MeasurementStatus = measMissing
		addReason(&out.Reasons, reasonMeasInputMissing)
	case measCorrupt:
		out.MeasurementStatus = measCorrupt
		addReason(&out.Reasons, reasonMeasInputCorrupt)
	case measUnsupportedSchema:
		out.MeasurementStatus = measUnsupportedSchema
		addReason(&out.Reasons, reasonMeasSchemaUnsupported)
	}
	return s
}

func initBaseline(out *Summary, path string) *summary.Summary {
	if path == "" {
		return nil
	}
	s, state := loadMeasurement(path)
	switch state {
	case measMissing:
		out.BaselineStatus = baseUnavailable
		addReason(&out.Reasons, reasonBaselineUnavailable)
	case measCorrupt, measUnsupportedSchema:
		out.BaselineStatus = baseCorrupt
		addReason(&out.Reasons, reasonBaselineCorrupt)
	default:
		out.BaselineStatus = basePresent
	}
	return s
}

// runResultStatus propagates per-SLI status from the measurement into gate checks.
//
// Rules (no policy override in MVP):
//
//	fail / block → check "fail", failed=true, reason RESULT_STATUS_FAIL
//	warn         → check "warn", anyWarn=true
//	skip (value==nil) → check "no_grade", anyNoGrade=true
//	pass         → no check added
func runResultStatus(out *Summary, s *summary.Summary) (failed, anyWarn, anyNoGrade bool) {
	if s == nil {
		return
	}
	for _, r := range s.Results {
		check := Check{
			Name:     fmt.Sprintf("result-status:%s", r.ID),
			Category: "result_status",
			Metric:   r.ID,
			Message:  r.Reason,
		}
		if r.Value != nil {
			check.Observed = *r.Value
		}

		switch r.Status {
		case summary.StatusFail, summary.StatusBlock:
			check.Status = "fail"
			addReason(&out.Reasons, reasonResultStatusFail)
			failed = true
		case summary.StatusWarn:
			check.Status = "warn"
			anyWarn = true
		case summary.StatusSkip:
			if r.Value != nil {
				continue // skip with a value: threshold check handles it
			}
			check.Status = "no_grade"
			anyNoGrade = true
		default:
			continue // pass: no check needed
		}
		out.Checks = append(out.Checks, check)
	}
	return
}

// --- result computation ---

func computeEvalStatus(checks []Check, anyNoGrade bool) string {
	if len(checks) == 0 {
		return evalNot
	}
	if anyNoGrade {
		return evalPartial
	}
	for _, c := range checks {
		if c.Status == "no_grade" {
			return evalPartial
		}
	}
	return evalEvaluated
}

func computeGateResult(failed, hasWarn, hasNoGrade bool, baselineStatus string, regrEnabled bool) string {
	switch {
	case failed:
		return GateFail
	case hasNoGrade:
		return GateNoGrade
	case hasWarn:
		return GateWarn
	case baselineStatus == baseAbsentFirst && regrEnabled:
		return GateWarn
	default:
		return GatePass
	}
}

func gateMessage(result string) string {
	switch result {
	case GateFail:
		return "Policy violation detected (threshold/regression)."
	case GateWarn:
		return "Policy evaluated with non-blocking warnings."
	case GateNoGrade:
		return "Policy could not be fully evaluated."
	default:
		return "Policy checks passed."
	}
}

// --- helpers ---

func addReason(reasons *[]string, code string) {
	for _, r := range *reasons {
		if r == code {
			return
		}
	}
	*reasons = append(*reasons, code)
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
