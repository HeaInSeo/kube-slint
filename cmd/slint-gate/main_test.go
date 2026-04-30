package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HeaInSeo/kube-slint/internal/gate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	in := &gate.Summary{
		SchemaVersion: "slint.gate.v1",
		GateResult:    gate.GatePass,
		Reasons:       []string{},
		Checks:        []gate.Check{},
	}

	require.NoError(t, writeJSON(path, in))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var out gate.Summary
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, gate.GatePass, out.GateResult)
}

func TestWriteJSON_NoHTMLEscape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	in := &gate.Summary{
		Checks: []gate.Check{
			{Name: "t", Expected: ">= 1"},
		},
		Reasons: []string{},
	}

	require.NoError(t, writeJSON(path, in))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	// SetEscapeHTML(false) 설정으로 ">=" 가 HTML 엔터티(>=)로 이스케이프되지 않아야 함
	assert.Contains(t, string(data), `>=`)
	assert.NotContains(t, string(data), "\\u003e")
}

func TestWriteJSON_InvalidPath(t *testing.T) {
	err := writeJSON("/nonexistent/path/out.json", &gate.Summary{})
	assert.Error(t, err)
}

func TestRenderGitHubStepSummary_NoEnvVar(t *testing.T) {
	// GITHUB_STEP_SUMMARY 환경 변수가 없으면 무시하고 에러 없이 반환해야 함
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	result := &gate.Summary{
		GateResult:        gate.GatePass,
		EvaluationStatus:  "evaluated",
		MeasurementStatus: "ok",
		BaselineStatus:    "absent_first_run",
		PolicyStatus:      "ok",
		OverallMessage:    "Policy checks passed.",
		Reasons:           []string{},
		Checks:            []gate.Check{},
	}
	assert.NoError(t, renderGitHubStepSummary(result))
}

func TestRenderGitHubStepSummary_WritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	sumPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", sumPath)

	result := &gate.Summary{
		GateResult:        gate.GateFail,
		EvaluationStatus:  "evaluated",
		MeasurementStatus: "ok",
		BaselineStatus:    "present",
		PolicyStatus:      "ok",
		OverallMessage:    "Policy violation detected.",
		Reasons:           []string{"THRESHOLD_MISS"},
		Checks: []gate.Check{
			{
				Name:     "reconcile_min",
				Category: "threshold",
				Status:   "fail",
				Metric:   "reconcile_total_delta",
				Observed: float64(0),
				Expected: ">= 1",
				Message:  "threshold miss",
			},
		},
	}

	require.NoError(t, renderGitHubStepSummary(result))

	data, err := os.ReadFile(sumPath)
	require.NoError(t, err)
	body := string(data)

	assert.Contains(t, body, "# slint-gate Result")
	assert.Contains(t, body, gate.GateFail)
	assert.Contains(t, body, "THRESHOLD_MISS")
	assert.Contains(t, body, "reconcile_min")
	assert.Contains(t, body, "threshold miss")
}

func TestRenderGitHubStepSummary_NoChecks(t *testing.T) {
	dir := t.TempDir()
	sumPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", sumPath)

	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{},
		Checks:     []gate.Check{},
	}

	require.NoError(t, renderGitHubStepSummary(result))

	data, err := os.ReadFile(sumPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "(no checks)")
}

func TestRenderGitHubStepSummary_NoReasons(t *testing.T) {
	dir := t.TempDir()
	sumPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", sumPath)

	result := &gate.Summary{
		GateResult: gate.GatePass,
		Reasons:    []string{},
		Checks:     []gate.Check{},
	}

	require.NoError(t, renderGitHubStepSummary(result))

	data, err := os.ReadFile(sumPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "(none)")
}

func TestRenderGitHubStepSummary_CheckWithNilObserved(t *testing.T) {
	dir := t.TempDir()
	sumPath := filepath.Join(dir, "step-summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", sumPath)

	result := &gate.Summary{
		GateResult: gate.GateNoGrade,
		Reasons:    []string{},
		Checks: []gate.Check{
			{Name: "c", Category: "threshold", Status: "no_grade", Observed: nil, Expected: ">= 1"},
		},
	}

	require.NoError(t, renderGitHubStepSummary(result))

	data, err := os.ReadFile(sumPath)
	require.NoError(t, err)
	// nil observed → 빈 셀로 렌더링
	assert.True(t, strings.Contains(string(data), "| c |"))
}
