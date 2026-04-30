package engine

import (
	"context"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

type twoSnapshotFetcher struct {
	start fetch.Sample
	end   fetch.Sample
}

func (f *twoSnapshotFetcher) Fetch(_ context.Context, at time.Time) (fetch.Sample, error) {
	if at.Equal(f.start.At) {
		return f.start, nil
	}
	return f.end, nil
}

func TestEngineExecuteWithJUMIAHMinimumSpecs(t *testing.T) {
	startAt := time.Unix(100, 0)
	endAt := time.Unix(200, 0)
	fetcher := &twoSnapshotFetcher{
		start: fetch.Sample{
			At: startAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:           10,
				`jumi_artifacts_registered_total`:   4,
				`jumi_input_resolve_requests_total`: 5,
				`jumi_input_remote_fetch_total`:     3,
				`jumi_input_materializations_total`: 2,
				`jumi_sample_runs_finalized_total`:  1,
				`jumi_gc_evaluate_requests_total`:   1,
				`jumi_fast_fail_trigger_total`:      1,
				`jumi_cleanup_backlog_objects`:      0,
				`ah_artifacts_registered_total`:     8,
				`ah_resolve_requests_total`:         20,
				`ah_fallback_total`:                 2,
				`ah_gc_backlog_bytes`:               128,
			},
		},
		end: fetch.Sample{
			At: endAt,
			Values: map[string]float64{
				`jumi_jobs_created_total`:           13,
				`jumi_artifacts_registered_total`:   7,
				`jumi_input_resolve_requests_total`: 9,
				`jumi_input_remote_fetch_total`:     5,
				`jumi_input_materializations_total`: 4,
				`jumi_sample_runs_finalized_total`:  3,
				`jumi_gc_evaluate_requests_total`:   3,
				`jumi_fast_fail_trigger_total`:      1,
				`jumi_cleanup_backlog_objects`:      2,
				`ah_artifacts_registered_total`:     11,
				`ah_resolve_requests_total`:         29,
				`ah_fallback_total`:                 4,
				`ah_gc_backlog_bytes`:               512,
			},
		},
	}
	writer := &mockWriter{}
	eng := New(fetcher, writer, nil)

	sum, err := eng.Execute(context.Background(), ExecuteRequest{
		Config: RunConfig{
			RunID:      "jumi-ah-minimum",
			StartedAt:  startAt,
			FinishedAt: endAt,
		},
		Specs:       spec.JUMIAHMinimumSpecs(),
		Reliability: &summary.Reliability{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(sum.Results) != len(spec.JUMIAHMinimumSpecs()) {
		t.Fatalf("len(results) = %d, want %d", len(sum.Results), len(spec.JUMIAHMinimumSpecs()))
	}
	values := map[string]float64{}
	for _, result := range sum.Results {
		if result.Value != nil {
			values[result.ID] = *result.Value
		}
	}
	if values["jumi_jobs_created_delta"] != 3 {
		t.Fatalf("jumi_jobs_created_delta = %v, want 3", values["jumi_jobs_created_delta"])
	}
	if values["jumi_artifacts_registered_delta"] != 3 {
		t.Fatalf("jumi_artifacts_registered_delta = %v, want 3", values["jumi_artifacts_registered_delta"])
	}
	if values["jumi_input_resolve_requests_delta"] != 4 {
		t.Fatalf("jumi_input_resolve_requests_delta = %v, want 4", values["jumi_input_resolve_requests_delta"])
	}
	if values["jumi_input_remote_fetch_delta"] != 2 {
		t.Fatalf("jumi_input_remote_fetch_delta = %v, want 2", values["jumi_input_remote_fetch_delta"])
	}
	if values["jumi_input_materializations_delta"] != 2 {
		t.Fatalf("jumi_input_materializations_delta = %v, want 2", values["jumi_input_materializations_delta"])
	}
	if values["jumi_sample_runs_finalized_delta"] != 2 {
		t.Fatalf("jumi_sample_runs_finalized_delta = %v, want 2", values["jumi_sample_runs_finalized_delta"])
	}
	if values["jumi_gc_evaluate_requests_delta"] != 2 {
		t.Fatalf("jumi_gc_evaluate_requests_delta = %v, want 2", values["jumi_gc_evaluate_requests_delta"])
	}
	if values["ah_artifacts_registered_delta"] != 3 {
		t.Fatalf("ah_artifacts_registered_delta = %v, want 3", values["ah_artifacts_registered_delta"])
	}
	if values["ah_resolve_requests_delta"] != 9 {
		t.Fatalf("ah_resolve_requests_delta = %v, want 9", values["ah_resolve_requests_delta"])
	}
	if values["jumi_cleanup_backlog_objects_end"] != 2 {
		t.Fatalf("jumi_cleanup_backlog_objects_end = %v, want 2", values["jumi_cleanup_backlog_objects_end"])
	}
}
