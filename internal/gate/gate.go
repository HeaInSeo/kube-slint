package gate

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"gopkg.in/yaml.v3"
)

const schemaVersion = "slint.gate.v1"

// Gate result values.
const (
	// GatePass indicates all policy checks passed.
	GatePass = "PASS"
	// GateWarn indicates non-blocking warnings (e.g. first-run, reliability).
	GateWarn = "WARN"
	// GateFail indicates a policy violation that may fail CI.
	GateFail = "FAIL"
	// GateNoGrade indicates the gate could not be evaluated.
	GateNoGrade = "NO_GRADE"
)

const (
	evalEvaluated = "evaluated"
	evalPartial   = "partially_evaluated"
	evalNot       = "not_evaluated"
)

const (
	measOK           = "ok"
	measMissing      = "missing"
	measCorrupt      = "corrupt"
	measInsufficient = "insufficient"
)

const (
	basePresent     = "present"
	baseAbsentFirst = "absent_first_run"
	baseUnavailable = "unavailable"
	baseCorrupt     = "corrupt"
)

const (
	policyOK      = "ok"
	policyMissing = measMissing
	policyInvalid = "invalid"
)

const (
	reasonThresholdMiss           = "THRESHOLD_MISS"
	reasonRegressionDetected      = "REGRESSION_DETECTED"
	reasonBaselineAbsentFirstRun  = "BASELINE_ABSENT_FIRST_RUN"
	reasonBaselineUnavailable     = "BASELINE_UNAVAILABLE"
	reasonBaselineCorrupt         = "BASELINE_CORRUPT"
	reasonMeasInputMissing        = "MEASUREMENT_INPUT_MISSING"
	reasonMeasInputCorrupt        = "MEASUREMENT_INPUT_CORRUPT"
	reasonPolicyMissing           = "POLICY_MISSING"
	reasonPolicyInvalid           = "POLICY_INVALID"
	reasonReliabilityInsufficient = "RELIABILITY_INSUFFICIENT"
)

// Policy is the deserialized .slint/policy.yaml.
type Policy struct {
	Thresholds  []ThresholdRule `yaml:"thresholds"`
	Regression  RegressionCfg   `yaml:"regression"`
	Reliability ReliabilityCfg  `yaml:"reliability"`
	FailOn      []string        `yaml:"fail_on"`
}

// ThresholdRule is a single absolute threshold check.
type ThresholdRule struct {
	Name     string  `yaml:"name"`
	Metric   string  `yaml:"metric"`
	Operator string  `yaml:"operator"`
	Value    float64 `yaml:"value"`
}

// RegressionCfg holds regression comparison configuration.
type RegressionCfg struct {
	Enabled          bool    `yaml:"enabled"`
	TolerancePercent float64 `yaml:"tolerance_percent"`
}

// ReliabilityCfg holds reliability requirement configuration.
type ReliabilityCfg struct {
	Required bool   `yaml:"required"`
	MinLevel string `yaml:"min_level"`
}

// Check is a single policy check result.
type Check struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Metric   string `json:"metric"`
	Observed any    `json:"observed"`
	Expected string `json:"expected"`
	Message  string `json:"message"`
}

// Summary is the slint-gate output schema (slint-gate-summary.json).
type Summary struct {
	SchemaVersion     string    `json:"schema_version"`
	GateResult        string    `json:"gate_result"`
	EvaluationStatus  string    `json:"evaluation_status"`
	MeasurementStatus string    `json:"measurement_status"`
	BaselineStatus    string    `json:"baseline_status"`
	PolicyStatus      string    `json:"policy_status"`
	Reasons           []string  `json:"reasons"`
	EvaluatedAt       string    `json:"evaluated_at"`
	InputRefs         InputRefs `json:"input_refs"`
	Checks            []Check   `json:"checks"`
	OverallMessage    string    `json:"overall_message"`
}

// InputRefs records which files were used as inputs.
type InputRefs struct {
	MeasurementSummary string  `json:"measurement_summary"`
	PolicyFile         string  `json:"policy_file"`
	BaselineFile       *string `json:"baseline_file"`
}

// Request carries input paths for Evaluate.
type Request struct {
	MeasurementPath string
	PolicyPath      string
	BaselinePath    string // empty = first-run, no baseline
}

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
	failOn := makeFailOn(policy)

	tFailed, tNoGrade := runThresholds(out, policy.Thresholds, cur, failOn)
	rFailed, rWarn, rNoGrade := runRegression(out, policy, cur, base)
	relWarn := runReliability(out, policy, measurement)

	anyFailed := tFailed || rFailed
	anyNoGrade := tNoGrade || rNoGrade
	hasWarn := rWarn || relWarn

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
	policy, state := loadPolicy(path)
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
	case measCorrupt:
		out.BaselineStatus = baseCorrupt
		addReason(&out.Reasons, reasonBaselineCorrupt)
	default:
		out.BaselineStatus = basePresent
	}
	return s
}

// --- check runners ---

func runThresholds(out *Summary, rules []ThresholdRule, cur map[string]float64, failOn map[string]bool) (failed, anyNoGrade bool) {
	for _, rule := range rules {
		check, ruleFailed, ruleNoGrade := evalThreshold(rule, cur, failOn)
		if ruleFailed {
			failed = true
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

func evalThreshold(rule ThresholdRule, cur map[string]float64, failOn map[string]bool) (thresholdResult, bool, bool) {
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
		return c, false, true
	}

	c.Observed = observed
	matched, err := compareOp(observed, rule.Operator, rule.Value)
	if err != nil {
		c.Message = "invalid operator"
		c.pendingReasons = []string{reasonPolicyInvalid}
		return c, false, true
	}

	if matched {
		c.Status = "pass"
		c.Message = "within threshold"
		return c, false, false
	}

	c.Status = "fail"
	c.Message = "threshold miss"
	c.pendingReasons = []string{reasonThresholdMiss}
	return c, failOn["threshold_miss"], false
}

func runRegression(out *Summary, policy *Policy, cur, base map[string]float64) (failed, anyWarn, anyNoGrade bool) {
	if !policy.Regression.Enabled {
		return false, false, false
	}
	switch out.BaselineStatus {
	case baseAbsentFirst:
		addReason(&out.Reasons, reasonBaselineAbsentFirstRun)
		return false, true, true
	case baseUnavailable, baseCorrupt:
		return false, false, true
	}
	failOn := makeFailOn(policy)
	for _, rule := range policy.Thresholds {
		if rule.Metric == "" {
			continue
		}
		check, rFailed, rNoGrade := evalRegressionCheck(rule, cur, base, policy.Regression.TolerancePercent, failOn)
		if rFailed {
			failed = true
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

func evalRegressionCheck(rule ThresholdRule, cur, base map[string]float64, tolerancePct float64, failOn map[string]bool) (thresholdResult, bool, bool) {
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
		return c, false, true
	}

	d := deltaPct(curVal, baseVal)
	c.Observed = d

	if math.Abs(d) > tolerancePct {
		c.Status = "fail"
		c.Message = "regression detected"
		c.pendingReasons = []string{reasonRegressionDetected}
		return c, failOn["regression_detected"], false
	}

	c.Status = "pass"
	c.Message = "within regression tolerance"
	return c, false, false
}

func runReliability(out *Summary, policy *Policy, s *summary.Summary) (anyWarn bool) {
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

	if policy.Reliability.Required && reliabilityRank(collectionStatus) < requiredRank {
		check.Status = "warn"
		check.Message = "reliability below required level"
		addReason(&out.Reasons, reasonReliabilityInsufficient)
		out.MeasurementStatus = measInsufficient
		anyWarn = true
	}
	out.Checks = append(out.Checks, check)
	return anyWarn
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
	case hasWarn:
		return GateWarn
	case baselineStatus == baseAbsentFirst && regrEnabled:
		return GateWarn
	case hasNoGrade:
		return GateNoGrade
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

// --- file loaders ---

func loadPolicy(path string) (*Policy, string) {
	if path == "" {
		return nil, policyMissing
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, policyMissing
		}
		return nil, policyInvalid
	}
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, policyInvalid
	}
	return &p, policyOK
}

func loadMeasurement(path string) (*summary.Summary, string) {
	if path == "" {
		return nil, measMissing
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, measMissing
		}
		return nil, measCorrupt
	}
	var s summary.Summary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, measCorrupt
	}
	return &s, measOK
}

// --- helpers ---

func resultValueMap(s *summary.Summary) map[string]float64 {
	m := map[string]float64{}
	if s == nil {
		return m
	}
	for _, r := range s.Results {
		if r.Value != nil {
			m[r.ID] = *r.Value
		}
	}
	return m
}

func makeFailOn(policy *Policy) map[string]bool {
	result := map[string]bool{}
	for _, item := range policy.FailOn {
		result[item] = true
	}
	if len(result) == 0 {
		result["threshold_miss"] = true
		result["regression_detected"] = true
	}
	return result
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

func deltaPct(cur, base float64) float64 {
	if base == 0 {
		if cur != 0 {
			return math.Inf(1)
		}
		return 0
	}
	return ((cur - base) / math.Abs(base)) * 100.0
}

func addReason(reasons *[]string, code string) {
	for _, r := range *reasons {
		if r == code {
			return
		}
	}
	*reasons = append(*reasons, code)
}

func compareOp(v float64, op string, target float64) (bool, error) {
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

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
