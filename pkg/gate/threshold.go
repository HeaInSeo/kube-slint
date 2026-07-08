package gate

import (
	"fmt"
	"strings"
)

func runThresholds(out *Summary, rules []ThresholdRule, cur map[string]float64, promote map[string]bool) (failed, warn, anyNoGrade bool) {
	for _, rule := range rules {
		check, ruleFailed, ruleWarn, ruleNoGrade := evalThreshold(rule, cur, promote)
		if ruleFailed {
			failed = true
		}
		if ruleWarn {
			warn = true
		}
		if ruleNoGrade {
			anyNoGrade = true
		}
		for _, r := range check.pendingReasons {
			addReason(&out.Reasons, r)
		}
		out.Checks = append(out.Checks, check.Check)
	}
	return
}

type thresholdResult struct {
	Check
	pendingReasons []string
}

// evalThreshold returns (result, failed, warn, noGrade).
// failed=true  → threshold miss and threshold_miss is in the promotion set → gate FAIL
// warn=true    → threshold miss but threshold_miss not in the promotion set → gate WARN (never PASS)
func evalThreshold(rule ThresholdRule, cur map[string]float64, promote map[string]bool) (thresholdResult, bool, bool, bool) {
	name := rule.Name
	if name == "" {
		name = "unnamed-threshold"
	}
	c := thresholdResult{
		Check: Check{
			Name:     name,
			Category: "threshold",
			Status:   "no_grade",
			Metric:   rule.Metric,
			Expected: fmt.Sprintf("%s %v", rule.Operator, rule.Value),
		},
	}

	observed, ok := cur[rule.Metric]
	if rule.Metric == "" || !ok {
		c.Message = "metric missing or invalid threshold target"
		c.pendingReasons = []string{reasonMeasInputMissing}
		return c, false, false, true
	}

	c.Observed = observed
	matched, err := CompareOp(observed, rule.Operator, rule.Value)
	if err != nil {
		c.Message = "invalid operator"
		c.pendingReasons = []string{reasonPolicyInvalid}
		return c, false, false, true
	}

	if matched {
		c.Status = "pass"
		c.Message = "within threshold"
		return c, false, false, false
	}

	c.Status = "fail"
	c.Message = "threshold miss"
	c.pendingReasons = []string{reasonThresholdMiss}
	if promote["threshold_miss"] {
		return c, true, false, false
	}
	return c, false, true, false
}

// CompareOp evaluates "v op target" for the operators supported in
// policy.yaml threshold rules (<=, >=, <, >, ==, and the =</=> aliases).
// Exported so CLI-only consumers (e.g. recommend-policy's default-threshold
// mismatch check) share the same operator semantics instead of
// reimplementing them.
func CompareOp(v float64, op string, target float64) (bool, error) {
	switch strings.TrimSpace(op) {
	case "<=", "=<":
		return v <= target, nil
	case ">=", "=>":
		return v >= target, nil
	case "<":
		return v < target, nil
	case ">":
		return v > target, nil
	case "==", "=":
		return v == target, nil
	default:
		return false, fmt.Errorf("unsupported operator: %q", op)
	}
}
