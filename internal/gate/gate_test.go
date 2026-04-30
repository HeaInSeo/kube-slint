package gate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/internal/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const categoryThreshold = "threshold"

// --- fixtures ---

type policyFixture struct {
	Thresholds  []map[string]any `yaml:"thresholds"`
	Regression  map[string]any   `yaml:"regression"`
	Reliability map[string]any   `yaml:"reliability"`
	FailOn      []string         `yaml:"fail_on"`
}

func defaultPolicy() policyFixture {
	return policyFixture{
		Thresholds: []map[string]any{
			{"name": "reconcile_min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
			{"name": "workqueue_max", "metric": "workqueue_depth_end", "operator": "<=", "value": 5},
		},
		Regression:  map[string]any{"enabled": true, "tolerance_percent": 5},
		Reliability: map[string]any{"required": false, "min_level": "partial"},
		FailOn:      []string{"threshold_miss", "regression_detected"},
	}
}

func writePolicyFile(t *testing.T, dir string, p policyFixture) string {
	t.Helper()
	data, err := yaml.Marshal(p)
	require.NoError(t, err)
	path := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

func makeMeasurement(values map[string]float64, collectionStatus string) summary.Summary {
	results := make([]summary.SLIResult, 0, len(values))
	for id, v := range values {
		results = append(results, summary.SLIResult{ID: id, Value: &v, Status: summary.StatusPass})
	}
	return summary.Summary{
		SchemaVersion: "slo.v3",
		GeneratedAt:   time.Now(),
		Results:       results,
		Reliability:   &summary.Reliability{CollectionStatus: collectionStatus},
	}
}

func writeMeasurementFile(t *testing.T, dir, name string, s summary.Summary) string {
	t.Helper()
	data, err := json.Marshal(s)
	require.NoError(t, err)
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// --- tests ---

func TestEvaluate_PolicyMissing(t *testing.T) {
	dir := t.TempDir()
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{"reconcile_total_delta": 3}, "Complete"))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      filepath.Join(dir, "nonexistent.yaml"),
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "missing", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_MISSING")
}

func TestEvaluate_MeasurementMissing(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())

	result := gate.Evaluate(gate.Request{
		MeasurementPath: filepath.Join(dir, "nonexistent.json"),
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "missing", result.MeasurementStatus)
	assert.Contains(t, result.Reasons, "MEASUREMENT_INPUT_MISSING")
}

func TestEvaluate_MeasurementCorrupt(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	measPath := filepath.Join(dir, "meas.json")
	require.NoError(t, os.WriteFile(measPath, []byte("not json {{{"), 0o644))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: measPath,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "corrupt", result.MeasurementStatus)
	assert.Contains(t, result.Reasons, "MEASUREMENT_INPUT_CORRUPT")
}

func TestEvaluate_FirstRun_ThresholdPass(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		// no baseline = first run
	})

	// regression enabled but no baseline → WARN (not FAIL)
	assert.Equal(t, gate.GateWarn, result.GateResult)
	assert.Contains(t, result.Reasons, "BASELINE_ABSENT_FIRST_RUN")

	// threshold checks themselves still pass
	var thresholdChecks []gate.Check
	for _, c := range result.Checks {
		if c.Category == categoryThreshold {
			thresholdChecks = append(thresholdChecks, c)
		}
	}
	require.Len(t, thresholdChecks, 2)
	for _, c := range thresholdChecks {
		assert.Equal(t, "pass", c.Status)
	}
}

func TestEvaluate_FirstRun_ThresholdFail(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		// reconcile_total_delta = 0 → fails ">= 1"
		map[string]float64{"reconcile_total_delta": 0, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "THRESHOLD_MISS")
}

func TestEvaluate_WithBaseline_AllPass(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
	assert.Equal(t, "evaluated", result.EvaluationStatus)
	for _, c := range result.Checks {
		assert.Equal(t, "pass", c.Status, "check %q should pass", c.Name)
	}
}

func TestEvaluate_WithBaseline_RegressionFail(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	// current: reconcile_total_delta doubled (100% increase → exceeds 5% tolerance)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 6, "workqueue_depth_end": 0},
		"Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "REGRESSION_DETECTED")
}

func TestEvaluate_WithBaseline_WithinTolerance(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	// 3% change — within 5% tolerance
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3.09, "workqueue_depth_end": 0},
		"Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3.0, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
}

func TestEvaluate_RegressionDisabled_NoBaseline(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false, "tolerance_percent": 5}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	// regression disabled and threshold passes → PASS (no WARN for missing baseline)
	assert.Equal(t, gate.GatePass, result.GateResult)
}

func TestEvaluate_ReliabilityRequired_BelowMinimum(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false}
	p.Reliability = map[string]any{"required": true, "min_level": "complete"}
	policy := writePolicyFile(t, dir, p)
	// collectionStatus "Partial" is below required "complete"
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Partial",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateWarn, result.GateResult)
	assert.Contains(t, result.Reasons, "RELIABILITY_INSUFFICIENT")
}

func TestEvaluate_MetricMissing_NoGrade(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false}
	policy := writePolicyFile(t, dir, p)
	// workqueue_depth_end is missing from measurement
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "partially_evaluated", result.EvaluationStatus)
	assert.Contains(t, result.Reasons, "MEASUREMENT_INPUT_MISSING")
}

func TestEvaluate_BaselineCorrupt(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))
	baselinePath := filepath.Join(dir, "baseline.json")
	require.NoError(t, os.WriteFile(baselinePath, []byte("{bad json"), 0o644))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baselinePath,
	})

	assert.Equal(t, "corrupt", result.BaselineStatus)
	// regression enabled but baseline corrupt → no regression checks → NO_GRADE
	assert.Equal(t, gate.GateNoGrade, result.GateResult)
}

func TestEvaluate_AllOperators(t *testing.T) {
	cases := []struct {
		op     string
		value  float64
		target float64
		pass   bool
	}{
		{"<", 0, 5, true},
		{"<", 5, 5, false},
		{">", 6, 5, true},
		{">", 5, 5, false},
		{"==", 5, 5, true},
		{"==", 4, 5, false},
		{"<=", 5, 5, true},
		{">=", 5, 5, true},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			dir := t.TempDir()
			p := defaultPolicy()
			p.Regression = map[string]any{"enabled": false}
			p.Thresholds = []map[string]any{
				{"name": "op-test", "metric": "m", "operator": tc.op, "value": tc.target},
			}
			policy := writePolicyFile(t, dir, p)
			meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
				map[string]float64{"m": tc.value}, "Complete",
			))

			result := gate.Evaluate(gate.Request{
				MeasurementPath: meas,
				PolicyPath:      policy,
			})

			var check gate.Check
			for _, c := range result.Checks {
				if c.Category == categoryThreshold {
					check = c
				}
			}
			if tc.pass {
				assert.Equal(t, "pass", check.Status)
			} else {
				assert.Equal(t, "fail", check.Status)
			}
		})
	}
}

func TestEvaluate_InvalidOperator_NoGrade(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false}
	p.Thresholds = []map[string]any{
		{"name": "bad-op", "metric": "m", "operator": "!=", "value": 1},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"m": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

func TestEvaluate_EmptyBaselinePath_TreatedAsFirstRun(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    "   ", // whitespace only — cmd strips it; gate itself won't see ""
	})

	// Whitespace baseline path → gate tries to open " " → "missing"
	assert.Equal(t, "unavailable", result.BaselineStatus)
}

func TestEvaluate_OutputSchema(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, "slint.gate.v1", result.SchemaVersion)
	assert.NotEmpty(t, result.EvaluatedAt)
	assert.NotNil(t, result.Reasons)
	assert.NotNil(t, result.Checks)
	assert.Equal(t, meas, result.InputRefs.MeasurementSummary)
	assert.Equal(t, policy, result.InputRefs.PolicyFile)
	assert.Nil(t, result.InputRefs.BaselineFile)
}

func TestEvaluate_PolicyInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(":\tinvalid: yaml: :::"), 0o644))
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policyPath,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "invalid", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

func TestEvaluate_BaselinePath_Set_InputRefs(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	require.NotNil(t, result.InputRefs.BaselineFile)
	assert.Equal(t, baseline, *result.InputRefs.BaselineFile)
}

func TestEvaluate_RegressionEnabled_BaselineAbsent_NoGrade(t *testing.T) {
	// regression enabled + no baseline → WARN (hasWarn=true from runRegression)
	// but hasNoGrade also true → computeGateResult: failed=false, hasWarn=true → WARN wins
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": true, "tolerance_percent": 5}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateWarn, result.GateResult)
}

func TestEvaluate_NoThresholds_RegressionDisabled_Pass(t *testing.T) {
	// 정책에 threshold 없고 regression disabled → reliability check만 통과하면 PASS
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
}

func TestEvaluate_ReliabilityPartial_NotRequired_Pass(t *testing.T) {
	// reliability.required=false → reliability check는 warn 발생 안 함
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false}
	p.Reliability = map[string]any{"required": false, "min_level": "complete"}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Partial",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
}

func TestEvaluate_DefaultFailOn_Applied(t *testing.T) {
	// FailOn 필드 비어있으면 기본값(threshold_miss, regression_detected) 적용 → FAIL
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 10},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{}, // 비어있음 → 기본값 적용
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateFail, result.GateResult)
}

func TestEvaluate_UnnamedThreshold(t *testing.T) {
	// threshold name이 비어있으면 "unnamed-threshold"로 대체
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 5}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
	for _, c := range result.Checks {
		if c.Category == categoryThreshold {
			assert.Equal(t, "unnamed-threshold", c.Name)
		}
	}
}

func TestEvaluate_RegressionMetricMissingInBaseline(t *testing.T) {
	// 현재에는 metric이 있지만 baseline에 없으면 regression check → no_grade
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))
	// baseline에 workqueue_depth_end 없음
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	// regression check 일부가 no_grade → partially_evaluated
	assert.Equal(t, "partially_evaluated", result.EvaluationStatus)
}
