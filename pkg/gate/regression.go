package gate

import (
	"fmt"
	"math"
	"strings"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

func runRegression(out *Summary, policy *Policy, measurement, baseline *summary.Summary) (failed, anyWarn, anyNoGrade bool) {
	if !policy.Regression.Enabled {
		return false, false, false
	}
	switch out.BaselineStatus {
	case baseAbsentFirst:
		addReason(&out.Reasons, reasonBaselineAbsentFirstRun)
		return false, true, false
	case baseUnavailable, baseCorrupt:
		return false, false, true
	}

	// KSL-T4/T5: a protected baseline comparison may run only under the
	// trust-correct contracts — a trust-correct policy (slint.policy.v2) AND a
	// trust-correct measurement+baseline (slo.v4) that carry comparability
	// identity. Legacy contracts (slint.policy.v1 / slo.v3) cannot prove
	// comparability, so regression is NO_GRADE (baseline-incomparable), never a
	// silent comparison against evidence of unknown comparability.
	if !policyIsTrustCorrect(policy) || measurement == nil || baseline == nil ||
		!summary.IsTrustCorrectContract(*measurement) || !summary.IsTrustCorrectContract(*baseline) {
		addReason(&out.Reasons, reasonBaselineIncomparable)
		out.Checks = append(out.Checks, Check{
			Name:     "regression-comparability",
			Category: "regression",
			Status:   "no_grade",
			Metric:   "*",
			Expected: "trust-correct policy (slint.policy.v2) + measurement/baseline (slo.v4) with comparability identity",
			Message:  "baseline comparability cannot be proven under legacy contracts; regression not graded",
		})
		return false, false, true
	}

	curEv := newEvidenceIndex(measurement)
	baseEv := newEvidenceIndex(baseline)
	curCmp := comparabilityIndex(measurement)
	baseCmp := comparabilityIndex(baseline)
	promote := makePromotionSet(policy)
	for _, rule := range policy.Thresholds {
		if rule.Metric == "" {
			continue
		}
		check, rFailed, rWarnCheck, rNoGrade := evalRegressionCheck(
			rule, curEv, baseEv, curCmp, baseCmp, policy.Regression.TolerancePercent, promote)
		if rFailed {
			failed = true
		}
		if rWarnCheck {
			anyWarn = true
		}
		if rNoGrade {
			anyNoGrade = true
		}
		for _, r := range check.pendingReasons {
			addReason(&out.Reasons, r)
		}
		out.Checks = append(out.Checks, check.Check)
	}
	return failed, anyWarn, anyNoGrade
}

// comparabilityIndex maps each SLI ID to its recorded comparability identity (nil
// when absent) so a regression check can prove per-SLI comparability (KSL-T4).
func comparabilityIndex(s *summary.Summary) map[string]*summary.Comparability {
	m := map[string]*summary.Comparability{}
	if s == nil {
		return m
	}
	for _, r := range s.Results {
		m[r.ID] = r.Comparability
	}
	return m
}

// evalRegressionCheck returns (result, failed, warn, noGrade).
// failed=true  → regression detected and regression_detected is in the promotion set → gate FAIL
// warn=true    → regression detected but regression_detected not in the promotion set → gate WARN (never PASS)
//
// KSL-T3/T4: the check grades only when both current and baseline evidence are
// positively sufficient AND their comparability identities match on every
// coordinate; otherwise it is NO_GRADE, never a silent comparison.
func evalRegressionCheck(rule ThresholdRule, curEv, baseEv evidenceIndex, curCmp, baseCmp map[string]*summary.Comparability, tolerancePct float64, promote map[string]bool) (thresholdResult, bool, bool, bool) {
	c := thresholdResult{
		Check: Check{
			Name:     fmt.Sprintf("regression:%s", rule.Metric),
			Category: "regression",
			Status:   "no_grade",
			Metric:   rule.Metric,
			Expected: fmt.Sprintf("abs(delta_percent) <= %v", tolerancePct),
		},
	}

	curVal, curOK, _ := curEv.valueSufficient(rule.Metric)
	baseVal, baseOK, _ := baseEv.valueSufficient(rule.Metric)
	if !curOK || !baseOK {
		c.Message = "current or baseline evidence insufficient to grade regression"
		c.pendingReasons = []string{reasonEvidenceInsufficient}
		return c, false, false, true
	}
	if !curCmp[rule.Metric].Equal(baseCmp[rule.Metric]) {
		c.Message = "current and baseline are not provably comparable for this SLI"
		c.pendingReasons = []string{reasonBaselineIncomparable}
		return c, false, false, true
	}

	// baseline=0, current≠0: unquantifiable percent change from zero.
	// Guard here to prevent math.Inf(1) from reaching JSON encoding.
	if baseVal == 0 && curVal != 0 {
		if HigherIsBetter(rule.Operator) {
			// e.g. reconcile rate going from 0 to nonzero is an improvement, not a regression.
			c.Status = "pass"
			c.Observed = "baseline_zero_current_nonzero"
			c.Message = "baseline is zero; current improved from zero"
			return c, false, false, false
		}
		c.Status = "fail"
		c.Observed = "baseline_zero_current_nonzero"
		c.Message = "regression detected: baseline is zero, current is non-zero"
		c.pendingReasons = []string{reasonRegressionDetected}
		if promote["regression_detected"] {
			return c, true, false, false
		}
		return c, false, true, false
	}

	d := deltaPct(curVal, baseVal)
	c.Observed = d

	if isRegression(d, tolerancePct, rule.Operator) {
		c.Status = "fail"
		c.Message = "regression detected"
		c.pendingReasons = []string{reasonRegressionDetected}
		if promote["regression_detected"] {
			return c, true, false, false
		}
		return c, false, true, false
	}

	c.Status = "pass"
	c.Message = "within regression tolerance"
	return c, false, false, false
}

// isRegression reports whether a percent change d from baseline to current is a
// regression, given tolerancePct and the metric's improvement direction inferred
// from the paired threshold rule's operator. Metrics without a recognized
// direction (e.g. "==") fall back to a symmetric tolerance check.
func isRegression(d, tolerancePct float64, operator string) bool {
	switch {
	case LowerIsBetter(operator):
		return d > tolerancePct
	case HigherIsBetter(operator):
		return d < -tolerancePct
	default:
		return math.Abs(d) > tolerancePct
	}
}

// LowerIsBetter reports whether a threshold rule's operator implies that a
// lower metric value is an improvement (<=, <, =<). Exported so CLI-only
// direction-aware consumers (baseline diff/merge wording) share the same
// operator-to-direction mapping instead of reimplementing it.
func LowerIsBetter(operator string) bool {
	switch strings.TrimSpace(operator) {
	case "<=", "<", "=<":
		return true
	default:
		return false
	}
}

// HigherIsBetter is the higher-is-better counterpart to LowerIsBetter
// (>=, >, =>).
func HigherIsBetter(operator string) bool {
	switch strings.TrimSpace(operator) {
	case ">=", ">", "=>":
		return true
	default:
		return false
	}
}

// deltaPct returns the percentage change from base to cur.
//
// Contract: callers MUST handle base == 0 && cur != 0 before calling this
// function if the result will be serialized to JSON, because deltaPct returns
// +Inf for that case and encoding/json cannot marshal +Inf.
// See evalRegressionCheck for the required guard pattern.
func deltaPct(cur, base float64) float64 {
	if base == 0 {
		if cur != 0 {
			return math.Inf(1)
		}
		return 0
	}
	return ((cur - base) / math.Abs(base)) * 100.0
}
