package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errFetcher는 항상 에러를 반환하는 페처.
type errFetcher struct{ count int }

func (f *errFetcher) Fetch(_ context.Context, _ time.Time) (fetch.Sample, error) {
	f.count++
	return fetch.Sample{}, errors.New("fetch error")
}

// endErrFetcher는 두 번째 Fetch에서만 에러를 반환하는 페처.
type endErrFetcher struct{ calls int }

func (f *endErrFetcher) Fetch(_ context.Context, _ time.Time) (fetch.Sample, error) {
	f.calls++
	if f.calls == 2 {
		return fetch.Sample{}, errors.New("end fetch error")
	}
	return fetch.Sample{Values: map[string]float64{"m": 1}}, nil
}

var baseTime = time.Unix(1000, 0)
var endTime = time.Unix(1060, 0)

func makeReq(specs []spec.SLISpec) ExecuteRequest {
	return ExecuteRequest{
		Config: RunConfig{
			StartedAt:  baseTime,
			FinishedAt: endTime,
		},
		Specs:       specs,
		Reliability: &summary.Reliability{},
	}
}

// --- Execute 에러 경로 ---

func TestExecute_ZeroTime_Error(t *testing.T) {
	eng := New(&mockStaticFetcher{}, &mockWriter{}, nil)
	_, err := eng.Execute(context.Background(), ExecuteRequest{
		Config: RunConfig{}, // StartedAt/FinishedAt zero
	})
	assert.ErrorContains(t, err, "StartedAt/FinishedAt must be set")
}

func TestExecute_StartFetchError_ReturnsSummary(t *testing.T) {
	writer := &mockWriter{}
	eng := New(&errFetcher{}, writer, nil)
	sum, err := eng.Execute(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.Equal(t, "Failed", sum.Reliability.CollectionStatus)
	assert.NotNil(t, writer.lastWritten)
}

func TestExecute_EndFetchError_ReturnsSummary(t *testing.T) {
	writer := &mockWriter{}
	eng := New(&endErrFetcher{}, writer, nil)
	sum, err := eng.Execute(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.Equal(t, "Failed", sum.Reliability.CollectionStatus)
}

// --- evalSLI compute mode 경로 ---

func TestEvalSLI_ComputeStart(t *testing.T) {
	s := spec.SLISpec{
		ID:      "start_val",
		Inputs:  []spec.MetricRef{{Key: "m"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeStart},
	}
	result := evalSLI(s, map[string]float64{"m": 7}, map[string]float64{"m": 3})
	require.NotNil(t, result.Value)
	assert.Equal(t, 7.0, *result.Value)
}

func TestEvalSLI_ComputeEnd(t *testing.T) {
	s := spec.SLISpec{
		ID:      "end_val",
		Inputs:  []spec.MetricRef{{Key: "m"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeEnd},
	}
	result := evalSLI(s, map[string]float64{"m": 3}, map[string]float64{"m": 9})
	require.NotNil(t, result.Value)
	assert.Equal(t, 9.0, *result.Value)
}

func TestEvalSLI_ComputeSingle(t *testing.T) {
	s := spec.SLISpec{
		ID:      "single_val",
		Inputs:  []spec.MetricRef{{Key: "m"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeSingle},
	}
	result := evalSLI(s, map[string]float64{"m": 5}, map[string]float64{"m": 10})
	require.NotNil(t, result.Value)
	assert.Equal(t, 5.0, *result.Value)
}

func TestEvalSLI_DeltaNegative_Warn(t *testing.T) {
	s := spec.SLISpec{
		ID:      "neg_delta",
		Inputs:  []spec.MetricRef{{Key: "m"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
	}
	// end < start → 카운터 리셋 의심
	result := evalSLI(s, map[string]float64{"m": 10}, map[string]float64{"m": 3})
	assert.Equal(t, summary.StatusWarn, result.Status)
	assert.Contains(t, result.Reason, "counter reset")
}

func TestEvalSLI_UnknownMode_Skip(t *testing.T) {
	s := spec.SLISpec{
		ID:      "unknown",
		Inputs:  []spec.MetricRef{{Key: "m"}},
		Compute: spec.ComputeSpec{Mode: "bogus"},
	}
	result := evalSLI(s, map[string]float64{"m": 1}, map[string]float64{"m": 2})
	assert.Equal(t, summary.StatusSkip, result.Status)
	assert.Contains(t, result.Reason, "unknown compute mode")
}

func TestEvalSLI_MissingInput_Skip(t *testing.T) {
	s := spec.SLISpec{
		ID:      "missing",
		Inputs:  []spec.MetricRef{{Key: "not_present"}},
		Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
	}
	result := evalSLI(s, map[string]float64{}, map[string]float64{})
	assert.Equal(t, summary.StatusSkip, result.Status)
	assert.Contains(t, result.InputsMissing, "not_present")
}

// --- judge ---

func TestJudge_WarnRule(t *testing.T) {
	rules := []spec.Rule{
		{Op: spec.OpGT, Target: 0, Level: spec.LevelWarn},
	}
	status, reason := judge(5, rules)
	assert.Equal(t, summary.StatusWarn, status)
	assert.Contains(t, reason, "warn")
}

func TestJudge_FailOverridesWarn(t *testing.T) {
	rules := []spec.Rule{
		{Op: spec.OpGT, Target: 0, Level: spec.LevelWarn},
		{Op: spec.OpGT, Target: 3, Level: spec.LevelFail},
	}
	status, _ := judge(5, rules)
	assert.Equal(t, summary.StatusFail, status)
}

func TestJudge_NoRuleMatch_Pass(t *testing.T) {
	rules := []spec.Rule{
		{Op: spec.OpGT, Target: 100, Level: spec.LevelFail},
	}
	status, reason := judge(5, rules)
	assert.Equal(t, summary.StatusPass, status)
	assert.Empty(t, reason)
}

// --- compare ---

func TestCompare_AllOps(t *testing.T) {
	cases := []struct {
		op     spec.Op
		v      float64
		target float64
		want   bool
	}{
		{spec.OpLE, 5, 5, true},
		{spec.OpLE, 6, 5, false},
		{spec.OpGE, 5, 5, true},
		{spec.OpGE, 4, 5, false},
		{spec.OpLT, 4, 5, true},
		{spec.OpLT, 5, 5, false},
		{spec.OpGT, 6, 5, true},
		{spec.OpGT, 5, 5, false},
		{spec.OpEQ, 5, 5, true},
		{spec.OpEQ, 4, 5, false},
		{"invalid", 1, 1, false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, compare(tc.v, tc.op, tc.target))
	}
}

// --- emptySummary ---

func TestEmptySummary(t *testing.T) {
	eng := &Engine{}
	rel := &summary.Reliability{}
	cfg := RunConfig{
		RunID:      "test-run",
		StartedAt:  baseTime,
		FinishedAt: endTime,
	}
	sum := eng.emptySummary(cfg, rel, []string{"warning1"})
	assert.Equal(t, "slo.v3", sum.SchemaVersion)
	assert.Equal(t, []string{"warning1"}, sum.Warnings)
	assert.Empty(t, sum.Results)
}

// --- MapMethodToRunMode ---

func TestMapMethodToRunMode(t *testing.T) {
	cases := []struct {
		method   MeasurementMethod
		location RunLocation
		trigger  RunTrigger
	}{
		{InsideSnapshot, RunLocationInside, RunTriggerNone},
		{InsideAnnotation, RunLocationInside, RunTriggerAnnotation},
		{OutsideSnapshot, RunLocationOutside, RunTriggerNone},
		{"unknown", RunLocationInside, RunTriggerNone}, // default
	}
	for _, tc := range cases {
		got := MapMethodToRunMode(tc.method)
		assert.Equal(t, tc.location, got.Location)
		assert.Equal(t, tc.trigger, got.Trigger)
	}
}

// --- ExecuteStandard ---

func TestExecuteStandard(t *testing.T) {
	values := map[string]float64{"m": 5}
	writer := &mockWriter{}
	eng := New(&mockStaticFetcher{values: values}, writer, nil)

	sum, err := ExecuteStandard(context.Background(), eng, ExecuteRequestStandard{
		Method: InsideSnapshot,
		Config: RunConfig{
			StartedAt:  baseTime,
			FinishedAt: endTime,
		},
		Specs: []spec.SLISpec{
			{
				ID:      "m_delta",
				Inputs:  []spec.MetricRef{{Key: "m"}},
				Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			},
		},
		Reliability: &summary.Reliability{},
	})
	require.NoError(t, err)
	assert.Equal(t, "slo.v3", sum.SchemaVersion)
	assert.Equal(t, "v4", sum.Config.Format)
	assert.Equal(t, "inside", sum.Config.Mode.Location)
}

// --- Execute 전체 파이프라인 (스킵 SLI 포함) ---

func TestExecute_SkippedSLI_PartialEvaluation(t *testing.T) {
	values := map[string]float64{"present": 10}
	writer := &mockWriter{}
	eng := New(&mockStaticFetcher{values: values}, writer, nil)

	specs := []spec.SLISpec{
		{
			ID:      "present",
			Inputs:  []spec.MetricRef{{Key: "present"}},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:      "missing",
			Inputs:  []spec.MetricRef{{Key: "absent"}},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
	}

	sum, err := eng.Execute(context.Background(), makeReq(specs))
	require.NoError(t, err)
	assert.Equal(t, "Partial", sum.Reliability.EvaluationStatus)
	assert.Contains(t, sum.Reliability.SkippedSLIs, "missing")
	assert.Contains(t, sum.Reliability.MissingInputs, "absent")
}
