package gate_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const categoryThreshold = "threshold"

// --- fixtures ---

type policyFixture struct {
	SchemaVersion string           `yaml:"schema_version"`
	Thresholds    []map[string]any `yaml:"thresholds"`
	Regression    map[string]any   `yaml:"regression"`
	Reliability   map[string]any   `yaml:"reliability"`
	Coverage      map[string]any   `yaml:"coverage,omitempty"`
	FailOn        []string         `yaml:"fail_on"`
	PromoteToFail []string         `yaml:"promote_to_fail,omitempty"`
}

// defaultPolicy returns the trust-correct policy contract (slint.policy.v2), so
// the shared helpers exercise the protected path by default. legacyPolicy covers
// the legacy (slint.policy.v1) fence.
func defaultPolicy() policyFixture {
	return policyFixture{
		SchemaVersion: "slint.policy.v2",
		Thresholds: []map[string]any{
			{"name": "reconcile_min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
			{"name": "workqueue_max", "metric": "workqueue_depth_end", "operator": "<=", "value": 5},
		},
		Regression:    map[string]any{"enabled": true, "tolerance_percent": 5},
		Reliability:   map[string]any{"required": false, "min_level": "partial"},
		PromoteToFail: []string{"threshold_miss", "regression_detected"},
	}
}

// legacyPolicy returns the same policy under the legacy contract (slint.policy.v1).
func legacyPolicy() policyFixture {
	p := defaultPolicy()
	p.SchemaVersion = "slint.policy.v1"
	return p
}

func writePolicyFile(t *testing.T, dir string, p policyFixture) string {
	t.Helper()
	if p.SchemaVersion == "" {
		p.SchemaVersion = "slint.policy.v2"
	}
	data, err := yaml.Marshal(p)
	require.NoError(t, err)
	path := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o644))
	return path
}

// defaultComparability is the trust-correct comparability identity the shared
// helpers stamp on every SLI, so a current/baseline pair built from the same
// helper is provably comparable by default (KSL-T4). Tests that need an
// incomparable pair override it explicitly.
func defaultComparability() *summary.Comparability {
	return &summary.Comparability{
		SLIContractID:  "contract-1",
		SubjectID:      "subject-1",
		WindowID:       "5m-avg",
		SourceConfigID: "cfg-1",
	}
}

// makeMeasurement builds a trust-correct (slo.v4) measurement with a complete,
// uniform comparability identity per SLI.
func makeMeasurement(values map[string]float64, collectionStatus string) summary.Summary {
	results := make([]summary.SLIResult, 0, len(values))
	for id, v := range values {
		results = append(results, summary.SLIResult{
			ID: id, Value: &v, Status: summary.StatusPass,
			Comparability: defaultComparability(),
		})
	}
	return summary.Summary{
		SchemaVersion: summary.SchemaVersionTrust,
		GeneratedAt:   time.Now(),
		Results:       results,
		Reliability:   &summary.Reliability{CollectionStatus: collectionStatus},
	}
}

// makeLegacyMeasurement builds a legacy (slo.v3) measurement — no comparability
// identity, so it can never satisfy protected baseline comparability.
func makeLegacyMeasurement(values map[string]float64, collectionStatus string) summary.Summary {
	s := makeMeasurement(values, collectionStatus)
	s.SchemaVersion = summary.SchemaVersionLegacy
	for i := range s.Results {
		s.Results[i].Comparability = nil
	}
	return s
}

// makeMeasurementWithComparability builds a trust-correct measurement stamping a
// specific comparability identity (or nil) on every SLI.
func makeMeasurementWithComparability(values map[string]float64, cmp *summary.Comparability) summary.Summary {
	s := makeMeasurement(values, "Complete")
	for i := range s.Results {
		s.Results[i].Comparability = cmp
	}
	return s
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

// TestEvaluate_MeasurementReadIOError_LogsRealError is a regression test for
// a finding from pre-release-adversarial-review (2026-07-08): a non-NotExist
// os.ReadFile error was discarded entirely and mapped to the generic
// measCorrupt state, indistinguishable from actually-malformed JSON.
func TestEvaluate_MeasurementReadIOError_LogsRealError(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	// A directory in place of the expected file reliably produces a
	// non-NotExist os.ReadFile error (EISDIR) on all platforms this repo
	// targets, without needing OS-specific permission manipulation.
	measPath := filepath.Join(dir, "meas-is-a-dir")
	require.NoError(t, os.Mkdir(measPath, 0o755))

	stderr := captureStderr(t, func() {
		result := gate.Evaluate(gate.Request{
			MeasurementPath: measPath,
			PolicyPath:      policy,
		})
		assert.Equal(t, "corrupt", result.MeasurementStatus)
	})

	assert.Contains(t, stderr, "could not read")
	assert.Contains(t, stderr, measPath)
}

// captureStderr redirects os.Stderr to a pipe and returns everything fn()
// wrote to it. The read end is drained concurrently in a goroutine started
// before fn() runs (not after) — see cmd/slint-gate/inspect_test.go's
// captureStdout, which has the identical shape and an explanatory comment:
// os.Pipe()'s write end has a bounded kernel buffer (commonly 64KiB on
// Linux, not a portable guarantee), so draining only after fn() returns
// risks a deadlock if fn() ever writes more than that in one call.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stderr = old
	<-done

	return buf.String()
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
	// current: reconcile_total_delta halved (50% decrease → exceeds 5% tolerance).
	// reconcile_total_delta is a ">=" (higher-is-better) metric, so a decrease is
	// the regressing direction, not an increase.
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0},
		"Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 6, "workqueue_depth_end": 0},
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

func TestEvaluate_WithBaseline_RegressionFail_ImprovementIsNotRegression(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	// current: reconcile_total_delta doubled (100% increase). reconcile_total_delta
	// is a ">=" (higher-is-better) metric, so this is an improvement, not a regression.
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

	assert.Equal(t, gate.GatePass, result.GateResult)
	assert.NotContains(t, result.Reasons, "REGRESSION_DETECTED")
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

func TestEvaluate_CollectionFailed_NoGrade_EvenWithoutReliabilityRequired(t *testing.T) {
	// Regression test for R1: a summary whose CollectionStatus is "Failed" must
	// never silently resolve to PASS, even when the policy has no threshold
	// rules and reliability.required is false — a measurement that never
	// completed cannot support a trustworthy gate decision.
	dir := t.TempDir()
	p := policyFixture{
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{}, "Failed",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Contains(t, result.Reasons, "COLLECTION_FAILED")
}

func TestEvaluate_CollectionFailed_OutranksWarn(t *testing.T) {
	// COLLECTION_FAILED (NO_GRADE) must outrank a concurrent WARN-level check.
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 10},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"regression_detected"}, // threshold_miss absent → would be WARN on its own
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 0}, "Failed",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
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

// TestEvaluate_PolicyReadIOError_SurfacesRealErrorInWarnings is a regression
// test for a finding from pre-release-adversarial-review (2026-07-08): a
// non-NotExist os.ReadFile error (permission denied, EISDIR, etc.) was
// discarded entirely and mapped to the generic policyInvalid state,
// indistinguishable from an actual YAML syntax error.
func TestEvaluate_PolicyReadIOError_SurfacesRealErrorInWarnings(t *testing.T) {
	dir := t.TempDir()
	// A directory in place of the expected file reliably produces a
	// non-NotExist os.ReadFile error (EISDIR) on all platforms this repo
	// targets, without needing OS-specific permission manipulation.
	policyPath := filepath.Join(dir, "policy-is-a-dir")
	require.NoError(t, os.Mkdir(policyPath, 0o755))
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policyPath,
	})

	assert.Equal(t, "invalid", result.PolicyStatus)
	require.NotEmpty(t, result.PolicyWarnings)
	assert.Contains(t, result.PolicyWarnings[0], "could not read policy file")
}

func TestEvaluate_PolicyUnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.SchemaVersion = "slint.policy.v0"
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "invalid", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

func TestEvaluate_PolicyInvalidFailOn(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.FailOn = []string{"threshold-miss"}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
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
		SchemaVersion: "slint.policy.v1",
		Thresholds:    []map[string]any{},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		FailOn:        []string{"threshold_miss"},
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

func TestEvaluate_DefaultPromotion_Applied(t *testing.T) {
	// Empty promotion fields apply the strict defaults:
	// threshold_miss, regression_detected, and coverage_gap.
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

func TestEvaluate_DefaultPromotion_CoverageGapFailsWhenCoverageRequired(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		Coverage:    map[string]any{"required": true},
		FailOn:      []string{},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"new_unclassified_sli": 1}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "COVERAGE_GAP")
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

// --- policy unknown field validity tests (KSL-T1) ---

// KSL-T1: an unknown top-level key makes the whole policy invalid (fail-closed),
// not a warn-and-still-evaluate condition. An invalid-but-readable policy must
// never yield a protected grade, so the gate is NO_GRADE, never PASS.
func TestEvaluate_PolicyUnknownTopLevelField_IsInvalid(t *testing.T) {
	dir := t.TempDir()

	policyYAML := `schema_version: "slint.policy.v1"
thresholds:
  - name: reconcile_min
    metric: reconcile_total_delta
    operator: ">="
    value: 1
metadata:
  author: test     # unknown top-level field — must invalidate the policy
severity: high     # another unknown top-level field
`
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(policyYAML), 0o644))

	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"reconcile_total_delta": 2,
	}, "complete"))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policyPath,
	})

	require.Equal(t, gate.GateNoGrade, result.GateResult)
	require.Equal(t, "invalid", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

// KSL-T1: an unknown NESTED key (inside a threshold rule) is equally invalid —
// strict validity applies at every semantic level, not only the top level.
func TestEvaluate_PolicyUnknownNestedField_IsInvalid(t *testing.T) {
	dir := t.TempDir()

	policyYAML := `schema_version: "slint.policy.v1"
thresholds:
  - name: reconcile_min
    metric: reconcile_total_delta
    operator: ">="
    value: 1
    severty: high    # misspelled nested key — must invalidate the policy
`
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(policyYAML), 0o644))

	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"reconcile_total_delta": 2,
	}, "complete"))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policyPath,
	})

	require.Equal(t, gate.GateNoGrade, result.GateResult)
	require.Equal(t, "invalid", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

// KSL-T1: a trailing second YAML document is invalid — a policy may not smuggle
// in a second, unevaluated definition.
func TestEvaluate_PolicyTrailingDocument_IsInvalid(t *testing.T) {
	dir := t.TempDir()

	policyYAML := `schema_version: "slint.policy.v1"
thresholds:
  - name: reconcile_min
    metric: reconcile_total_delta
    operator: ">="
    value: 1
---
schema_version: "slint.policy.v1"
`
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(policyYAML), 0o644))

	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"reconcile_total_delta": 2,
	}, "complete"))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policyPath,
	})

	require.Equal(t, gate.GateNoGrade, result.GateResult)
	require.Equal(t, "invalid", result.PolicyStatus)
	assert.Contains(t, result.Reasons, "POLICY_INVALID")
}

// KSL-T1: the policy-validity matrix — each malformed policy must invalidate the
// whole policy (NO_GRADE / POLICY_INVALID) before evaluation, never silently
// evaluate a partial policy.
func TestEvaluate_PolicyValidityMatrix(t *testing.T) {
	cases := map[string]string{
		"unsupported operator": `schema_version: "slint.policy.v1"
thresholds:
  - {name: t, metric: m, operator: "!=", value: 1}
`,
		"empty operator": `schema_version: "slint.policy.v1"
thresholds:
  - {name: t, metric: m, value: 1}
`,
		"empty metric": `schema_version: "slint.policy.v1"
thresholds:
  - {name: t, operator: ">=", value: 1}
`,
		"duplicate threshold identity": `schema_version: "slint.policy.v1"
thresholds:
  - {name: dup, metric: a, operator: ">=", value: 1}
  - {name: dup, metric: b, operator: ">=", value: 1}
`,
		"bad reliability enum": `schema_version: "slint.policy.v1"
reliability: {required: true, min_level: sometimes}
`,
		"bad promote value": `schema_version: "slint.policy.v1"
promote_to_fail: [not_a_category]
`,
		"duplicate yaml mapping key": `schema_version: "slint.policy.v1"
schema_version: "slint.policy.v1"
`,
		"negative tolerance while regression disabled": `schema_version: "slint.policy.v1"
regression: {enabled: false, tolerance_percent: -5}
`,
		"infinite threshold value": `schema_version: "slint.policy.v1"
thresholds:
  - {name: t, metric: m, operator: "<=", value: .inf}
`,
		"missing threshold value": `schema_version: "slint.policy.v1"
thresholds:
  - {name: t, metric: m, operator: ">="}
`,
	}
	for name, policyYAML := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			policyPath := filepath.Join(dir, "policy.yaml")
			require.NoError(t, os.WriteFile(policyPath, []byte(policyYAML), 0o644))
			meas := writeMeasurementFile(t, dir, "meas.json",
				makeMeasurement(map[string]float64{"m": 2, "a": 2, "b": 2}, "complete"))
			result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policyPath})
			require.Equal(t, gate.GateNoGrade, result.GateResult, "invalid policy must be NO_GRADE")
			require.Equal(t, "invalid", result.PolicyStatus)
			assert.Contains(t, result.Reasons, "POLICY_INVALID")
		})
	}
}

// KSL-T1: policy invalidity must not be maskable by an otherwise-genuine check
// FAIL. A policy that is invalid (unknown key) AND contains a threshold a valid
// policy would FAIL on must still be NO_GRADE, never FAIL.
func TestEvaluate_PolicyInvalidityNotMaskedByFail(t *testing.T) {
	dir := t.TempDir()
	policyYAML := `schema_version: "slint.policy.v1"
thresholds:
  - {name: would-fail, metric: m, operator: ">=", value: 100}
bogus_top_level: true
`
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(policyYAML), 0o644))
	// m=1 < 100 would be a genuine threshold FAIL under a valid policy.
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{"m": 1}, "complete"))
	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policyPath})
	require.Equal(t, gate.GateNoGrade, result.GateResult, "invalidity must not be masked by a would-be FAIL")
	require.Equal(t, "invalid", result.PolicyStatus)
}

// --- promote_to_fail / fail_on dual support ---

func TestEvaluate_PromoteToFailOnly_EquivalentToFailOn(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 10},
		},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		PromoteToFail: []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	// Same outcome as the equivalent fail_on-only policy in
	// TestEvaluate_DefaultFailOn_Applied's sibling cases: threshold miss
	// promoted to FAIL.
	assert.Equal(t, gate.GateFail, result.GateResult)
}

func TestEvaluate_FailOn_ProducesDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 2}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	found := false
	for _, w := range result.PolicyWarnings {
		if strings.Contains(w, "fail_on") && strings.Contains(w, "promote_to_fail") {
			found = true
		}
	}
	assert.True(t, found, "expected a fail_on deprecation warning, got: %v", result.PolicyWarnings)
}

func TestEvaluate_PromoteToFailOnly_NoDeprecationWarning(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
		},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		PromoteToFail: []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 2}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	for _, w := range result.PolicyWarnings {
		assert.NotContains(t, w, "fail_on")
	}
}

func TestEvaluate_FailOnAndPromoteToFail_UnionApplied(t *testing.T) {
	// fail_on only contains threshold_miss, promote_to_fail only contains
	// regression_detected — both conditions must still be promoted to FAIL,
	// confirming the two fields are unioned rather than one overriding the
	// other.
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
		},
		Regression:    map[string]any{"enabled": true, "tolerance_percent": 5},
		Reliability:   map[string]any{"required": false},
		FailOn:        []string{"threshold_miss"},
		PromoteToFail: []string{"regression_detected"},
	}
	policy := writePolicyFile(t, dir, p)
	// operator ">=" means higher-is-better: a large drop from baseline is the
	// regression, while the threshold itself (>= 1) still passes at 10.
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 10}, "Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 100}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	// regression_detected (from promote_to_fail) must promote to FAIL even
	// though the threshold itself passes.
	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "REGRESSION_DETECTED")
}

// --- fail_on semantics: check=fail must never produce PASS ---

func TestEvaluate_ThresholdFail_NotInFailOn_IsWarn(t *testing.T) {
	// threshold fails but fail_on only contains regression_detected → WARN, not PASS
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 10},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"regression_detected"}, // threshold_miss intentionally absent
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	// check must be fail
	require.Len(t, result.Checks, 2) // threshold + reliability
	var tCheck gate.Check
	for _, c := range result.Checks {
		if c.Category == "threshold" {
			tCheck = c
		}
	}
	assert.Equal(t, "fail", tCheck.Status)

	// gate_result must be WARN, never PASS
	assert.Equal(t, gate.GateWarn, result.GateResult)
}

func TestEvaluate_RegressionFail_NotInFailOn_IsWarn(t *testing.T) {
	// regression detected but fail_on only contains threshold_miss → WARN, not PASS
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "min", "metric": "reconcile_total_delta", "operator": ">=", "value": 1},
		},
		Regression:  map[string]any{"enabled": true, "tolerance_percent": 5},
		Reliability: map[string]any{"required": false},
		FailOn:      []string{"threshold_miss"}, // regression_detected intentionally absent
	}
	policy := writePolicyFile(t, dir, p)
	// 50% decrease → exceeds 5% tolerance. reconcile_total_delta is ">=" (higher
	// is better), so a decrease is the regressing direction.
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3}, "Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 6}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	var rCheck gate.Check
	for _, c := range result.Checks {
		if c.Category == "regression" {
			rCheck = c
		}
	}
	assert.Equal(t, "fail", rCheck.Status)
	assert.Equal(t, gate.GateWarn, result.GateResult)
}

// --- regression base=0 guard: no +Inf in JSON ---

func TestEvaluate_Regression_BaselineZero_CurrentNonzero_IsFail(t *testing.T) {
	// baseline=0, current=1 → must not produce +Inf; JSON must be valid
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 1},
		"Complete",
	))
	baseline := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, // workqueue base=0
		"Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
		BaselinePath:    baseline,
	})

	// gate must fail (regression_detected in default fail_on)
	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "REGRESSION_DETECTED")

	// regression check observed must be a string, not a float — no +Inf
	var rCheck gate.Check
	for _, c := range result.Checks {
		if c.Category == "regression" && c.Metric == "workqueue_depth_end" {
			rCheck = c
		}
	}
	assert.Equal(t, "fail", rCheck.Status)
	assert.Equal(t, "baseline_zero_current_nonzero", rCheck.Observed)

	// JSON serialization must succeed (no +Inf)
	_, err := json.Marshal(result)
	assert.NoError(t, err, "JSON marshal must not fail with baseline=0 current!=0")
}

func TestEvaluate_Regression_BothZero_IsPass(t *testing.T) {
	// baseline=0, current=0 → delta=0 → within tolerance → pass
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
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
	for _, c := range result.Checks {
		if c.Category == "regression" && c.Metric == "workqueue_depth_end" {
			assert.Equal(t, "pass", c.Status)
		}
	}
}

// --- schemaVersion validation tests ---

func TestEvaluate_MeasurementEmptySchema_NoGrade(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	s := makeMeasurement(map[string]float64{"reconcile_total_delta": 3}, "Complete")
	s.SchemaVersion = ""
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "unsupported_schema", result.MeasurementStatus)
	assert.Contains(t, result.Reasons, "MEASUREMENT_SCHEMA_UNSUPPORTED")
}

func TestEvaluate_MeasurementUnknownSchema_NoGrade(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())
	s := makeMeasurement(map[string]float64{"reconcile_total_delta": 3}, "Complete")
	s.SchemaVersion = "slint.summary.v4"
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Equal(t, "unsupported_schema", result.MeasurementStatus)
	assert.Contains(t, result.Reasons, "MEASUREMENT_SCHEMA_UNSUPPORTED")
}

func TestEvaluate_MeasurementSupportedSchema_Evaluates(t *testing.T) {
	dir := t.TempDir()
	p := defaultPolicy()
	p.Regression = map[string]any{"enabled": false}
	policy := writePolicyFile(t, dir, p)
	// makeMeasurement already sets SchemaVersion = "slo.v3"
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Equal(t, gate.GatePass, result.GateResult)
	assert.Equal(t, "ok", result.MeasurementStatus)
}

func TestEvaluate_PolicyKnownFieldsOnly_NoWarnings(t *testing.T) {
	dir := t.TempDir()

	policy := writePolicyFile(t, dir, defaultPolicy())
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"reconcile_total_delta": 3,
		"workqueue_depth_end":   1,
	}, "complete"))

	result := gate.Evaluate(gate.Request{
		MeasurementPath: meas,
		PolicyPath:      policy,
	})

	assert.Empty(t, result.PolicyWarnings, "expected no warnings for a valid policy")
}

// --- SLIResult.Status propagation tests ---

// makeResultsMeasurement builds a Summary from a slice of SLIResults directly.
func makeResultsMeasurement(results []summary.SLIResult) summary.Summary {
	return summary.Summary{
		SchemaVersion: summary.SchemaVersionTrust,
		GeneratedAt:   time.Now(),
		Results:       results,
		Reliability:   &summary.Reliability{CollectionStatus: "Complete"},
	}
}

func ptr(v float64) *float64 { return &v }

// KSL-T2: a producer status of warn/fail/block is recorded as a NON-AUTHORITATIVE
// diagnostic and never drives the protected grade. With no Gate Policy check on
// the SLI, the gate is PASS (nothing to grade), and a measurement_diagnostic check
// records the producer verdict for visibility.
func TestEvaluate_ProducerStatus_IsDiagnosticOnly(t *testing.T) {
	cases := []struct {
		name   string
		result summary.SLIResult
	}{
		{"warn", summary.SLIResult{ID: "churn_delta", Status: summary.StatusWarn, Reason: "counter reset suspected", Value: ptr(-3)}},
		{"fail", summary.SLIResult{ID: "reconcile_total", Status: summary.StatusFail, Reason: "producer rule fail", Value: ptr(50)}},
		{"block", summary.SLIResult{ID: "pipeline_blocked", Status: summary.StatusBlock, Reason: "upstream pipeline failure"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := policyFixture{
				Thresholds:  []map[string]any{},
				Regression:  map[string]any{"enabled": false},
				Reliability: map[string]any{"required": false},
				FailOn:      []string{"threshold_miss"},
			}
			policy := writePolicyFile(t, dir, p)
			meas := writeMeasurementFile(t, dir, "meas.json", makeResultsMeasurement([]summary.SLIResult{tc.result}))

			result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

			assert.Equal(t, gate.GatePass, result.GateResult, "producer verdict must not drive the grade")
			assert.NotContains(t, result.Reasons, "RESULT_STATUS_FAIL")
			var diag gate.Check
			for _, c := range result.Checks {
				if c.Category == "measurement_diagnostic" {
					diag = c
				}
			}
			assert.Equal(t, "info", diag.Status, "producer status is recorded as a diagnostic")
			assert.Equal(t, tc.result.ID, diag.Metric)
		})
	}
}

// KSL-T2/T3: value=10 with producer status=fail but a Gate Policy threshold that
// passes must be PASS — the producer verdict cannot manufacture a FAIL.
func TestEvaluate_ProducerFail_PolicyThresholdPasses_IsPass(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{{"name": "t", "metric": "m", "operator": "<=", "value": 100}},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusFail, Reason: "producer says fail", Value: ptr(10),
			Comparability: defaultComparability()},
	}))
	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})
	assert.Equal(t, gate.GatePass, result.GateResult)
}

// KSL-T2: a genuine Gate Policy violation (threshold miss) remains FAIL — the
// grade authority is Gate Policy, and it still fails on a real violation.
func TestEvaluate_PolicyThresholdViolation_IsFail(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:    []map[string]any{{"name": "t", "metric": "m", "operator": ">=", "value": 100}},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		PromoteToFail: []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusPass, Value: ptr(10), Comparability: defaultComparability()},
	}))
	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})
	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "THRESHOLD_MISS")
}

// KSL-T2/T3: producer status=pass but the required evidence is missing (no value)
// must NOT PASS — evidence insufficiency (a typed fact) makes the required check
// NO_GRADE, without relying on the producer verdict word.
func TestEvaluate_ProducerPass_EvidenceMissing_IsNoGrade(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{{"name": "t", "metric": "m", "operator": ">=", "value": 1}},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	// status=pass but Value=nil: the producer verdict claims pass, yet there is no
	// evidence to grade the required threshold on.
	meas := writeMeasurementFile(t, dir, "meas.json", makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusPass, Value: nil, Comparability: defaultComparability()},
	}))
	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})
	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Contains(t, result.Reasons, "EVIDENCE_INSUFFICIENT")
}

// KSL-T3: a required SLI whose evidence is unreliable (recorded in the skipped set
// or with missing inputs) is NO_GRADE, while an independently sufficient SLI in
// the same run still grades — an unrelated insufficient SLI must not poison it.
func TestEvaluate_EvidenceSufficiency_IsPerCheckIndependent(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "ok", "metric": "good", "operator": ">=", "value": 1},
			{"name": "bad", "metric": "skipped", "operator": ">=", "value": 1},
		},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	s := makeResultsMeasurement([]summary.SLIResult{
		{ID: "good", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
		{ID: "skipped", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
	})
	// "skipped" carries a value but is positively marked skipped by the reliability
	// record: its evidence is not sufficient, so its check must NO_GRADE.
	s.Reliability.SkippedSLIs = []string{"skipped"}
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	// One check NO_GRADE (skipped), one graded PASS (good). Run-level result is
	// NO_GRADE (a required check could not be graded), but the "good" check is
	// still evaluated rather than poisoned.
	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	assert.Contains(t, result.Reasons, "EVIDENCE_INSUFFICIENT")
	var goodCheck, skippedCheck gate.Check
	for _, c := range result.Checks {
		switch c.Metric {
		case "good":
			goodCheck = c
		case "skipped":
			skippedCheck = c
		}
	}
	assert.Equal(t, "pass", goodCheck.Status, "an independently sufficient check still grades")
	assert.Equal(t, "no_grade", skippedCheck.Status, "an insufficient check is NO_GRADE")
}

// KSL-T3: when the collection failed, every value is untrustworthy — a threshold
// check must NO_GRADE on evidence insufficiency, never grade on the recorded value.
func TestEvaluate_FailedCollection_ValuesNotGraded(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{{"name": "t", "metric": "m", "operator": ">=", "value": 1}},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	// A value is present, but the collection is Failed: it must not be graded.
	s := makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
	})
	s.Reliability.CollectionStatus = "Failed"
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	var tCheck gate.Check
	for _, c := range result.Checks {
		if c.Category == "threshold" && c.Metric == "m" {
			tCheck = c
		}
	}
	assert.Equal(t, "no_grade", tCheck.Status, "a value from a failed collection must not be graded")
}

// KSL-T3: a failed EVALUATION phase (not only a failed collection) makes the run
// untrustworthy — values must not be graded.
func TestEvaluate_FailedEvaluation_ValuesNotGraded(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{{"name": "t", "metric": "m", "operator": ">=", "value": 1}},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	s := makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
	})
	s.Reliability.CollectionStatus = "Complete"
	s.Reliability.EvaluationStatus = "Failed"
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GateNoGrade, result.GateResult)
	var tCheck gate.Check
	for _, c := range result.Checks {
		if c.Category == "threshold" && c.Metric == "m" {
			tCheck = c
		}
	}
	assert.Equal(t, "no_grade", tCheck.Status, "a value from a failed evaluation must not be graded")
}

// KSL-T3: an explicitly failed evaluation is an UNCONDITIONAL run-level NO_GRADE,
// even when the policy has no checks that would consult evidence sufficiency.
func TestEvaluate_FailedEvaluation_NoChecks_IsNoGrade(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
	}
	policy := writePolicyFile(t, dir, p)
	s := makeResultsMeasurement([]summary.SLIResult{
		{ID: "m", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
	})
	s.Reliability.CollectionStatus = "Complete"
	s.Reliability.EvaluationStatus = "Failed"
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GateNoGrade, result.GateResult, "a failed evaluation must be NO_GRADE even with no checks")
	assert.Contains(t, result.Reasons, "COLLECTION_FAILED")
}

// KSL-T3: a coverage-gap check counts an SLI as "measured" only when its evidence
// is positively sufficient; an SLI with a value but insufficient evidence (here,
// skipped) must not produce a coverage gap.
func TestEvaluate_Coverage_IgnoresInsufficientEvidence(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		Coverage:    map[string]any{"required": true},
	}
	policy := writePolicyFile(t, dir, p)
	s := makeResultsMeasurement([]summary.SLIResult{
		{ID: "uncovered", Status: summary.StatusPass, Value: ptr(5), Comparability: defaultComparability()},
	})
	s.Reliability.SkippedSLIs = []string{"uncovered"} // insufficient evidence
	meas := writeMeasurementFile(t, dir, "meas.json", s)

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.NotContains(t, result.Reasons, "COVERAGE_GAP",
		"an SLI with insufficient evidence must not produce a coverage gap")
}

// KSL-T4: regression grades only when the current and baseline comparability
// identities match on every coordinate; any single-coordinate mismatch, or a
// missing identity, yields NO_GRADE (baseline-incomparable), never a silent
// comparison.
func TestEvaluate_RegressionComparabilityMatrix(t *testing.T) {
	base := &summary.Comparability{SLIContractID: "c", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg"}
	regressionPolicy := func() policyFixture {
		return policyFixture{
			SchemaVersion: "slint.policy.v2",
			Thresholds:    []map[string]any{{"name": "t", "metric": "m", "operator": "<=", "value": 1000}},
			Regression:    map[string]any{"enabled": true, "tolerance_percent": 5},
			Reliability:   map[string]any{"required": false},
			PromoteToFail: []string{"regression_detected"},
		}
	}
	mismatches := map[string]*summary.Comparability{
		"different subject":     {SLIContractID: "c", SubjectID: "s2", WindowID: "w", SourceConfigID: "cfg"},
		"changed SLI semantics": {SLIContractID: "c2", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg"},
		"different window":      {SLIContractID: "c", SubjectID: "s", WindowID: "w2", SourceConfigID: "cfg"},
		"changed source/config": {SLIContractID: "c", SubjectID: "s", WindowID: "w", SourceConfigID: "cfg2"},
		"missing comparability": nil,
	}
	for name, curCmp := range mismatches {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			policy := writePolicyFile(t, dir, regressionPolicy())
			cur := writeMeasurementFile(t, dir, "meas.json", makeMeasurementWithComparability(map[string]float64{"m": 100}, curCmp))
			basePath := writeMeasurementFile(t, dir, "baseline.json", makeMeasurementWithComparability(map[string]float64{"m": 100}, base))
			result := gate.Evaluate(gate.Request{MeasurementPath: cur, PolicyPath: policy, BaselinePath: basePath})
			assert.Equal(t, gate.GateNoGrade, result.GateResult, "incomparable baseline must be NO_GRADE")
			assert.Contains(t, result.Reasons, "BASELINE_INCOMPARABLE")
		})
	}
	t.Run("fully matching identity grades", func(t *testing.T) {
		dir := t.TempDir()
		policy := writePolicyFile(t, dir, regressionPolicy())
		cur := writeMeasurementFile(t, dir, "meas.json", makeMeasurementWithComparability(map[string]float64{"m": 100}, base))
		basePath := writeMeasurementFile(t, dir, "baseline.json", makeMeasurementWithComparability(map[string]float64{"m": 100}, base))
		result := gate.Evaluate(gate.Request{MeasurementPath: cur, PolicyPath: policy, BaselinePath: basePath})
		assert.Equal(t, gate.GatePass, result.GateResult)
		assert.NotContains(t, result.Reasons, "BASELINE_INCOMPARABLE")
	})
}

// KSL-T5 version boundaries: a legacy policy (v1) is never read with trust-correct
// (v2) semantics, and a legacy measurement contract (slo.v3) can never satisfy
// protected comparability by silently ignoring the trust-required fields.
func TestEvaluate_VersionBoundaries(t *testing.T) {
	vals := map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}

	t.Run("v1 policy is not trust-correct (regression not protected)", func(t *testing.T) {
		dir := t.TempDir()
		policy := writePolicyFile(t, dir, legacyPolicy()) // slint.policy.v1
		// Trust-correct v4 measurements that DO match — only the v1 policy blocks it.
		cur := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(vals, "Complete"))
		basePath := writeMeasurementFile(t, dir, "baseline.json", makeMeasurement(vals, "Complete"))
		result := gate.Evaluate(gate.Request{MeasurementPath: cur, PolicyPath: policy, BaselinePath: basePath})
		assert.Contains(t, result.Reasons, "BASELINE_INCOMPARABLE")
	})

	t.Run("v3 measurement cannot satisfy protected comparability", func(t *testing.T) {
		dir := t.TempDir()
		policy := writePolicyFile(t, dir, defaultPolicy()) // slint.policy.v2
		cur := writeMeasurementFile(t, dir, "meas.json", makeLegacyMeasurement(vals, "Complete"))
		basePath := writeMeasurementFile(t, dir, "baseline.json", makeLegacyMeasurement(vals, "Complete"))
		result := gate.Evaluate(gate.Request{MeasurementPath: cur, PolicyPath: policy, BaselinePath: basePath})
		assert.Contains(t, result.Reasons, "BASELINE_INCOMPARABLE")
	})
}

func TestEvaluate_CoverageRequiredWarnsOnUncoveredMeasuredSLI(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds: []map[string]any{
			{"name": "covered", "metric": "covered_delta", "operator": ">=", "value": 1},
		},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		Coverage:      map[string]any{"required": true},
		PromoteToFail: []string{"threshold_miss"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"covered_delta":   2,
		"uncovered_delta": 1,
	}, "Complete"))

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GateWarn, result.GateResult)
	assert.Contains(t, result.Reasons, "COVERAGE_GAP")
	var coverage gate.Check
	for _, c := range result.Checks {
		if c.Category == "coverage" {
			coverage = c
		}
	}
	assert.Equal(t, "warn", coverage.Status)
	assert.Equal(t, "uncovered_delta", coverage.Metric)
}

func TestEvaluate_CoverageGapCanPromoteToFail(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:    []map[string]any{},
		Regression:    map[string]any{"enabled": false},
		Reliability:   map[string]any{"required": false},
		Coverage:      map[string]any{"required": true},
		PromoteToFail: []string{"coverage_gap"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"uncovered_delta": 1,
	}, "Complete"))

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GateFail, result.GateResult)
	assert.Contains(t, result.Reasons, "COVERAGE_GAP")
	var coverage gate.Check
	for _, c := range result.Checks {
		if c.Category == "coverage" {
			coverage = c
		}
	}
	assert.Equal(t, "fail", coverage.Status)
	assert.Equal(t, "uncovered_delta", coverage.Metric)
}

func TestEvaluate_CoverageInformationalSuppressesGap(t *testing.T) {
	dir := t.TempDir()
	p := policyFixture{
		Thresholds:  []map[string]any{},
		Regression:  map[string]any{"enabled": false},
		Reliability: map[string]any{"required": false},
		Coverage: map[string]any{
			"required":      true,
			"informational": []string{"activity_delta"},
		},
		PromoteToFail: []string{"coverage_gap"},
	}
	policy := writePolicyFile(t, dir, p)
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(map[string]float64{
		"activity_delta": 1,
	}, "Complete"))

	result := gate.Evaluate(gate.Request{MeasurementPath: meas, PolicyPath: policy})

	assert.Equal(t, gate.GatePass, result.GateResult)
	for _, c := range result.Checks {
		assert.NotEqual(t, "coverage", c.Category)
	}
}
