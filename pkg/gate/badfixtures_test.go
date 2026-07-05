package gate_test

// Executable bad-fixture tests per docs/test-strategy.md's Bad Fixture
// Matrix. Each fixture must never produce gate.GatePass — either the policy
// or measurement input is rejected outright (PolicyStatus/MeasurementStatus
// != ok) or the gate result is NO_GRADE/FAIL.

import (
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
)

func badSummaryFixture(name string) string {
	return "testdata/summary/" + name
}

func badPolicyFixture(name string) string {
	return "testdata/policy/" + name
}

func assertNeverPass(t *testing.T, result *gate.Summary) {
	t.Helper()
	if result.GateResult == gate.GatePass {
		t.Fatalf("expected gate result to never be PASS for an invalid fixture; got PASS (policy_status=%s measurement_status=%s reasons=%v)",
			result.PolicyStatus, result.MeasurementStatus, result.Reasons)
	}
}

func TestBadFixtures_Summary(t *testing.T) {
	dir := t.TempDir()
	policy := writePolicyFile(t, dir, defaultPolicy())

	cases := []string{
		"missing-schema-version.json",
		"wrong-schema-version.json",
		"empty-result-id.json",
		"duplicate-result-id.json",
		"unknown-result-status.json",
		"invalid-generated-at.json",
		"malformed-json.json",
		"nan-metric-value.json",
		"inf-metric-value.json",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			result := gate.Evaluate(gate.Request{
				MeasurementPath: badSummaryFixture(name),
				PolicyPath:      policy,
			})
			assertNeverPass(t, result)
		})
	}
}

func TestBadFixtures_Policy(t *testing.T) {
	dir := t.TempDir()
	meas := writeMeasurementFile(t, dir, "meas.json", makeMeasurement(
		map[string]float64{"reconcile_total_delta": 3, "workqueue_depth_end": 0}, "Complete",
	))

	cases := []string{
		"missing-policy-version.yaml",
		"wrong-policy-version.yaml",
		"unknown-operator.yaml",
		"duplicate-threshold-name.yaml",
		"missing-metric.yaml",
		"negative-tolerance.yaml",
		"nan-threshold-value.yaml",
		"unknown-fail-on.yaml",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			result := gate.Evaluate(gate.Request{
				MeasurementPath: meas,
				PolicyPath:      badPolicyFixture(name),
			})
			assertNeverPass(t, result)
		})
	}
}
