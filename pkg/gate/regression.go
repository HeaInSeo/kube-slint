package gate

import (
	"fmt"
	"math"
	"strings"
)

func runRegression(out *Summary, policy *Policy, cur, base map[string]float64) (failed, anyWarn, anyNoGrade bool) {
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
	promote := makePromotionSet(policy)
	for _, rule := range policy.Thresholds {
		if rule.Metric == "" {
			continue
		}
		check, rFailed, rWarnCheck, rNoGrade := evalRegressionCheck(rule, cur, base, policy.Regression.TolerancePercent, promote)
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

// evalRegressionCheck returns (result, failed, warn, noGrade).
// failed=true  → regression detected and regression_detected is in the promotion set → gate FAIL
// warn=true    → regression detected but regression_detected not in the promotion set → gate WARN (never PASS)
func evalRegressionCheck(rule ThresholdRule, cur, base map[string]float64, tolerancePct float64, promote map[string]bool) (thresholdResult, bool, bool, bool) {
	c := thresholdResult{
		Check: Check{
			Name:     fmt.Sprintf("regression:%s", rule.Metric),
			Category: "regression",
			Status:   "no_grade",
			Metric:   rule.Metric,
			Expected: fmt.Sprintf("abs(delta_percent) <= %v", tolerancePct),
		},
	}

	curVal, hasCur := cur[rule.Metric]
	baseVal, hasBase := base[rule.Metric]
	if !hasCur || !hasBase {
		c.Message = "metric missing in current/baseline"
		return c, false, false, true
	}

	// baseline=0, current≠0: unquantifiable percent change from zero.
	// Guard here to prevent math.Inf(1) from reaching JSON encoding.
	if baseVal == 0 && curVal != 0 {
		if higherIsBetter(rule.Operator) {
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
	case lowerIsBetter(operator):
		return d > tolerancePct
	case higherIsBetter(operator):
		return d < -tolerancePct
	default:
		return math.Abs(d) > tolerancePct
	}
}

func lowerIsBetter(operator string) bool {
	switch strings.TrimSpace(operator) {
	case "<=", "<", "=<":
		return true
	default:
		return false
	}
}

func higherIsBetter(operator string) bool {
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
