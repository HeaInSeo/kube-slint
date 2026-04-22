package spec

// JUMIAHMinimumSpecs returns the minimum batch-oriented spec set needed to
// observe JUMI/AH integration work during the fast kind+tilt development loop.
func JUMIAHMinimumSpecs() []SLISpec {
	return []SLISpec{
		{
			ID:          "jumi_jobs_created_delta",
			Title:       "JUMI Jobs Created Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many Kubernetes Jobs JUMI created during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_jobs_created_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_fast_fail_trigger_delta",
			Title:       "JUMI Fast Fail Trigger Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks fast-fail trigger activity while changing JUMI executor behavior.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_fast_fail_trigger_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_artifacts_registered_delta",
			Title:       "JUMI Artifacts Registered Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many node outputs JUMI published to AH during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_artifacts_registered_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_input_resolve_requests_delta",
			Title:       "JUMI Input Resolve Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many artifact input resolution requests JUMI issued during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_resolve_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_input_remote_fetch_delta",
			Title:       "JUMI Input Remote Fetch Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how often JUMI accepted remote-fetch resolution decisions during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_remote_fetch_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_input_materializations_delta",
			Title:       "JUMI Input Materializations Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many resolved inputs required materialization during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_materializations_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_sample_runs_finalized_delta",
			Title:       "JUMI Sample Runs Finalized Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many sample runs JUMI finalized through AH during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_sample_runs_finalized_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_gc_evaluate_requests_delta",
			Title:       "JUMI GC Evaluate Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many GC evaluation requests JUMI issued to AH during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_gc_evaluate_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "jumi_cleanup_backlog_objects_end",
			Title:       "JUMI Cleanup Backlog Objects",
			Unit:        "count",
			Kind:        "gauge",
			Description: "Tracks cleanup debt visible at the end of the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_cleanup_backlog_objects`),
			},
			Compute: ComputeSpec{Mode: ComputeEnd},
		},
		{
			ID:          "ah_artifacts_registered_delta",
			Title:       "AH Artifacts Registered Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many artifacts AH accepted into inventory during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_artifacts_registered_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "ah_resolve_requests_delta",
			Title:       "AH Resolve Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how often JUMI asks AH to resolve a handoff during the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_resolve_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "ah_fallback_delta",
			Title:       "AH Fallback Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks fallback decisions while validating same-node reuse versus remote fetch behavior.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_fallback_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
		},
		{
			ID:          "ah_gc_backlog_bytes_end",
			Title:       "AH GC Backlog Bytes",
			Unit:        "bytes",
			Kind:        "gauge",
			Description: "Tracks retained backlog visible at the end of the measurement window.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_gc_backlog_bytes`),
			},
			Compute: ComputeSpec{Mode: ComputeEnd},
		},
	}
}
