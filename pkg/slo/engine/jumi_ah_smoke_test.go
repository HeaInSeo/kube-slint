package engine

import (
	"context"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

func TestEngineExecuteWithJUMIAHSmokeGuardrailSpecs(t *testing.T) {
	startAt := time.Unix(100, 0)
	endAt := time.Unix(200, 0)
	fetcher := &twoSnapshotFetcher{
		start: fetch.Sample{
			At: startAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:            0,
				`jumi_artifacts_registered_total`:    0,
				`jumi_input_resolve_requests_total`:  0,
				`jumi_input_remote_fetch_total`:      0,
				`jumi_input_materializations_total`:  0,
				`jumi_sample_runs_finalized_total`:   0,
				`jumi_gc_evaluate_requests_total`:    0,
				`ah_artifacts_registered_total`:      0,
				`ah_resolve_requests_total`:          0,
				`ah_fallback_total`:                  0,
				`ah_gc_backlog_bytes`:                0,
			},
		},
		end: fetch.Sample{
			At: endAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:            2,
				`jumi_artifacts_registered_total`:    1,
				`jumi_input_resolve_requests_total`:  1,
				`jumi_input_remote_fetch_total`:      1,
				`jumi_input_materializations_total`:  1,
				`jumi_sample_runs_finalized_total`:   1,
				`jumi_gc_evaluate_requests_total`:    1,
				`ah_artifacts_registered_total`:      1,
				`ah_resolve_requests_total`:          1,
				`ah_fallback_total`:                  1,
				`ah_gc_backlog_bytes`:                0,
			},
		},
	}
	writer := &mockWriter{}
	eng := New(fetcher, writer, nil)

	sum, err := eng.Execute(context.Background(), ExecuteRequest{
		Config: RunConfig{
			RunID:      "jumi-ah-smoke",
			StartedAt:  startAt,
			FinishedAt: endAt,
		},
		Specs:       spec.JUMIAHSmokeGuardrailSpecs(),
		Reliability: &summary.Reliability{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(sum.Results) != len(spec.JUMIAHSmokeGuardrailSpecs()) {
		t.Fatalf("len(results) = %d, want %d", len(sum.Results), len(spec.JUMIAHSmokeGuardrailSpecs()))
	}
	for _, result := range sum.Results {
		if result.Status != summary.StatusPass {
			t.Fatalf("result %s status = %s, want pass", result.ID, result.Status)
		}
	}
}

func TestEngineExecuteWithJUMIAHSmokeGuardrailSpecsFailure(t *testing.T) {
	startAt := time.Unix(100, 0)
	endAt := time.Unix(200, 0)
	fetcher := &twoSnapshotFetcher{
		start: fetch.Sample{
			At: startAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:            0,
				`jumi_artifacts_registered_total`:    0,
				`jumi_input_resolve_requests_total`:  0,
				`jumi_input_remote_fetch_total`:      0,
				`jumi_input_materializations_total`:  0,
				`jumi_sample_runs_finalized_total`:   0,
				`jumi_gc_evaluate_requests_total`:    0,
				`ah_artifacts_registered_total`:      0,
				`ah_resolve_requests_total`:          0,
				`ah_fallback_total`:                  0,
				`ah_gc_backlog_bytes`:                0,
			},
		},
		end: fetch.Sample{
			At: endAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:            1,
				`jumi_artifacts_registered_total`:    0,
				`jumi_input_resolve_requests_total`:  0,
				`jumi_input_remote_fetch_total`:      0,
				`jumi_input_materializations_total`:  0,
				`jumi_sample_runs_finalized_total`:   0,
				`jumi_gc_evaluate_requests_total`:    0,
				`ah_artifacts_registered_total`:      0,
				`ah_resolve_requests_total`:          0,
				`ah_fallback_total`:                  0,
				`ah_gc_backlog_bytes`:                64,
			},
		},
	}
	writer := &mockWriter{}
	eng := New(fetcher, writer, nil)

	sum, err := eng.Execute(context.Background(), ExecuteRequest{
		Config: RunConfig{
			RunID:      "jumi-ah-smoke-fail",
			StartedAt:  startAt,
			FinishedAt: endAt,
		},
		Specs:       spec.JUMIAHSmokeGuardrailSpecs(),
		Reliability: &summary.Reliability{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	failures := 0
	for _, result := range sum.Results {
		if result.Status == summary.StatusFail {
			failures++
		}
	}
	if failures == 0 {
		t.Fatal("expected at least one failing smoke guardrail result")
	}
}
