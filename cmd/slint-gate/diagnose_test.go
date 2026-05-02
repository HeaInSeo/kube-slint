package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/internal/gate"
	"github.com/stretchr/testify/assert"
)

func capturePrintDiagnostics(result *gate.Summary) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printDiagnostics(result)

	_ = w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintDiagnostics_PassSilent(t *testing.T) {
	result := &gate.Summary{GateResult: gate.GatePass, Reasons: []string{}}
	out := capturePrintDiagnostics(result)
	assert.Empty(t, out)
}

func TestPrintDiagnostics_MeasurementMissing(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{"MEASUREMENT_INPUT_MISSING"},
	}
	out := capturePrintDiagnostics(result)
	assert.Contains(t, out, "MEASUREMENT_INPUT_MISSING")
	assert.Contains(t, out, "sess.End")
	assert.Contains(t, out, "kubectl auth can-i")
}

func TestPrintDiagnostics_PolicyMissing(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{"POLICY_MISSING"},
	}
	out := capturePrintDiagnostics(result)
	assert.Contains(t, out, "POLICY_MISSING")
	assert.Contains(t, out, "slint-gate init")
}

func TestPrintDiagnostics_ThresholdMiss(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"THRESHOLD_MISS"},
	}
	out := capturePrintDiagnostics(result)
	assert.Contains(t, out, "THRESHOLD_MISS")
}

func TestPrintDiagnostics_BaselineAbsentFirstRun(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateWarn,
		Reasons:    []string{"BASELINE_ABSENT_FIRST_RUN"},
	}
	out := capturePrintDiagnostics(result)
	assert.Contains(t, out, "BASELINE_ABSENT_FIRST_RUN")
	assert.Contains(t, out, "baseline-update-prepare")
}

func TestPrintDiagnostics_UnknownReason(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"UNKNOWN_CODE"},
	}
	out := capturePrintDiagnostics(result)
	assert.Contains(t, out, "UNKNOWN_CODE")
}

func TestPrintDiagnostics_MultipleReasons(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{"THRESHOLD_MISS", "REGRESSION_DETECTED"},
	}
	out := capturePrintDiagnostics(result)
	assert.True(t, strings.Contains(out, "THRESHOLD_MISS"))
	assert.True(t, strings.Contains(out, "REGRESSION_DETECTED"))
}

func TestPrintDiagnostics_NoReasons(t *testing.T) {
	result := &gate.Summary{
		GateResult: gate.GateFail,
		Reasons:    []string{},
	}
	out := capturePrintDiagnostics(result)
	// 헤더는 출력되지만 reason 항목 없음
	assert.Contains(t, out, "Gate Result: FAIL")
}
