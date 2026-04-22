package spec

// JUMIAHSmokeGuardrailSpecs returns the minimum guardrail rules that should
// pass for the VM-lab JUMI -> AH smoke path.
func JUMIAHSmokeGuardrailSpecs() []SLISpec {
	return []SLISpec{
		{
			ID:          "jumi_jobs_created_smoke",
			Title:       "JUMI Jobs Created Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should create at least two Jobs for producer and consumer nodes.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_jobs_created_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 2, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_artifacts_registered_smoke",
			Title:       "JUMI Artifacts Registered Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should register at least one producer output through AH.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_artifacts_registered_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_input_resolve_requests_smoke",
			Title:       "JUMI Input Resolve Requests Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should resolve at least one artifact binding before starting the consumer node.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_resolve_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_input_remote_fetch_smoke",
			Title:       "JUMI Input Remote Fetch Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should observe at least one remote-fetch resolution decision.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_remote_fetch_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_input_materializations_smoke",
			Title:       "JUMI Input Materializations Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should mark at least one resolved input as requiring materialization.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_input_materializations_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_sample_runs_finalized_smoke",
			Title:       "JUMI Sample Runs Finalized Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should finalize at least one sample run through AH.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_sample_runs_finalized_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "jumi_gc_evaluate_requests_smoke",
			Title:       "JUMI GC Evaluate Requests Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should trigger at least one GC evaluation request.",
			Inputs: []MetricRef{
				UnsafePromKey(`jumi_gc_evaluate_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "ah_artifacts_registered_smoke",
			Title:       "AH Artifacts Registered Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should register at least one artifact inside AH inventory.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_artifacts_registered_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "ah_resolve_requests_smoke",
			Title:       "AH Resolve Requests Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should receive at least one resolve request from JUMI.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_resolve_requests_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "ah_fallback_smoke",
			Title:       "AH Fallback Smoke",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Smoke run should exercise at least one remote-fetch fallback path.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_fallback_total`),
			},
			Compute: ComputeSpec{Mode: ComputeDelta},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpLT, Target: 1, Level: LevelFail},
			}},
		},
		{
			ID:          "ah_gc_backlog_bytes_smoke",
			Title:       "AH GC Backlog Bytes Smoke",
			Unit:        "bytes",
			Kind:        "gauge",
			Description: "Smoke run should not leave non-zero AH GC backlog at the end of the window.",
			Inputs: []MetricRef{
				UnsafePromKey(`ah_gc_backlog_bytes`),
			},
			Compute: ComputeSpec{Mode: ComputeEnd},
			Judge: &JudgeSpec{Rules: []Rule{
				{Metric: "value", Op: OpGT, Target: 0, Level: LevelFail},
			}},
		},
	}
}
