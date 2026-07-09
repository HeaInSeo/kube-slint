package main

import (
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/pkg/gate"
	"github.com/stretchr/testify/assert"
)

// capturePrintDiagnostics delegates to the already-fixed captureStdout
// (inspect_test.go) rather than duplicating its own pipe-capture logic —
// a prior version here drained the pipe only after printDiagnostics
// returned, the same deadlock-on-large-output risk captureStdout was
// already fixed for (see its doc comment).
func capturePrintDiagnostics(t *testing.T, result *gate.Summary) string {
	t.Helper()
	return captureStdout(t, func() { printDiagnostics(result) })
}

func TestPrintDiagnostics_PassSilent(t *testing.T) {
	result := &gate.Summary{GateResult: gate.GatePass, Reasons: []string{}}
	out := capturePrintDiagnostics(t, result)
	assert.Empty(t, out)
}

func TestPrintDiagnostics_MeasurementMissing(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{"MEASUREMENT_INPUT_MISSING"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "MEASUREMENT_INPUT_MISSING")
	assert.Contains(t, out, "sess.End")
	assert.Contains(t, out, "kubectl auth can-i")
}

func TestPrintDiagnostics_PolicyMissing(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{"POLICY_MISSING"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "POLICY_MISSING")
	assert.Contains(t, out, "slint-gate init")
}

func TestPrintDiagnostics_PolicyInvalid_MentionsSchemaVersion(t *testing.T) {
	// Regression test for N4: schema_version is now a strictly validated
	// field (see pkg/gate/gate.go validatePolicy), so a missing/wrong
	// schema_version is a common cause of POLICY_INVALID. The hint must call
	// this out explicitly instead of only mentioning YAML syntax/operators.
	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{"POLICY_INVALID"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "POLICY_INVALID")
	assert.Contains(t, out, "schema_version")
	assert.Contains(t, out, "slint.policy.v1")
}

func TestPrintDiagnostics_ThresholdMiss(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"THRESHOLD_MISS"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "THRESHOLD_MISS")
}

func TestPrintDiagnostics_BaselineAbsentFirstRun(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateWarn,
		Reasons:    []string{"BASELINE_ABSENT_FIRST_RUN"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "BASELINE_ABSENT_FIRST_RUN")
	assert.Contains(t, out, "baseline-update-prepare")
}

func TestPrintDiagnostics_UnknownReason(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"UNKNOWN_CODE"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.Contains(t, out, "UNKNOWN_CODE")
}

func TestPrintDiagnostics_MultipleReasons(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"THRESHOLD_MISS", "REGRESSION_DETECTED"},
	}
	out := capturePrintDiagnostics(t, result)
	assert.True(t, strings.Contains(out, "THRESHOLD_MISS"))
	assert.True(t, strings.Contains(out, "REGRESSION_DETECTED"))
}

func TestPrintDiagnostics_NoReasons(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{},
	}
	out := capturePrintDiagnostics(t, result)
	// 헤더는 출력되지만 reason 항목 없음
	assert.Contains(t, out, "Gate Result: FAIL")
}
