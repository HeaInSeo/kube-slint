//go:build ignore

// Package jumiahspec demonstrates how to write consumer-specific SLI specs
// for a JUMI → AH dataplane integration. Copy and adapt for your own operator.
//
// Usage in your E2E test:
//
//	import "github.com/HeaInSeo/kube-slint/pkg/slint"
//
//	sess := slint.NewSession(slint.SessionConfig{
//	    Namespace:          "jumi-system",
//	    MetricsServiceName: "jumi-controller-metrics",
//	    ArtifactsDir:       "artifacts",
//	    Specs:              JUMIAHMinimumSpecs(),
//	})
package jumiahspec

import "github.com/HeaInSeo/kube-slint/pkg/slo/spec"

// JUMIAHMinimumSpecs returns the minimum batch-oriented spec set for the
// JUMI/AH integration development loop.
func JUMIAHMinimumSpecs() []spec.SLISpec {
	return []spec.SLISpec{
		{
			ID:          "jumi_jobs_created_delta",
			Title:       "JUMI Jobs Created Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many Kubernetes Jobs JUMI created during the measurement window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_jobs_created_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_fast_fail_trigger_delta",
			Title:       "JUMI Fast Fail Trigger Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks fast-fail trigger activity.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_fast_fail_trigger_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_artifacts_registered_delta",
			Title:       "JUMI Artifacts Registered Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many node outputs JUMI published to AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_artifacts_registered_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_input_resolve_requests_delta",
			Title:       "JUMI Input Resolve Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many artifact input resolution requests JUMI issued.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_resolve_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_input_remote_fetch_delta",
			Title:       "JUMI Input Remote Fetch Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how often JUMI accepted remote-fetch resolution decisions.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_remote_fetch_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_input_materializations_delta",
			Title:       "JUMI Input Materializations Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many resolved inputs required materialization.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_materializations_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_sample_runs_finalized_delta",
			Title:       "JUMI Sample Runs Finalized Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many sample runs JUMI finalized through AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_sample_runs_finalized_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_gc_evaluate_requests_delta",
			Title:       "JUMI GC Evaluate Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many GC evaluation requests JUMI issued to AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_gc_evaluate_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "jumi_cleanup_backlog_objects_end",
			Title:       "JUMI Cleanup Backlog Objects",
			Unit:        "count",
			Kind:        "gauge",
			Description: "Cleanup debt visible at the end of the measurement window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_cleanup_backlog_objects`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeEnd},
		},
		{
			ID:          "ah_artifacts_registered_delta",
			Title:       "AH Artifacts Registered Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how many artifacts AH accepted into inventory.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_artifacts_registered_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "ah_resolve_requests_delta",
			Title:       "AH Resolve Requests Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks how often JUMI asks AH to resolve a handoff.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_resolve_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "ah_fallback_delta",
			Title:       "AH Fallback Delta",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Tracks fallback decisions during same-node reuse vs remote fetch validation.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_fallback_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:          "ah_gc_backlog_bytes_end",
			Title:       "AH GC Backlog Bytes",
			Unit:        "bytes",
			Kind:        "gauge",
			Description: "Retained GC backlog visible at the end of the measurement window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_gc_backlog_bytes`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeEnd},
		},
		// Phase 1: handoff gRPC client counters (JUMI → AH call layer)
		{
			ID: "jumi_handoff_resolve_delta", Title: "JUMI Handoff Resolve Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks how many handoff resolve calls JUMI issued to AH over gRPC.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_handoff_resolve_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID: "jumi_handoff_resolve_errors_delta", Title: "JUMI Handoff Resolve Errors Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks gRPC errors on the handoff resolve call path.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_handoff_resolve_errors_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID: "jumi_handoff_register_artifact_delta", Title: "JUMI Handoff Register Artifact Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks how many artifact registrations JUMI sent to AH over gRPC.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_handoff_register_artifact_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		// Phase 1: K8s spawner lifecycle counters
		{
			ID: "jumi_k8s_node_prepare_delta", Title: "JUMI K8s Node Prepare Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks how many K8s Job prepare calls JUMI issued during the window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_k8s_node_prepare_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID: "jumi_k8s_node_start_delta", Title: "JUMI K8s Node Start Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks how many K8s Jobs JUMI started during the window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_k8s_node_start_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID: "jumi_k8s_node_succeeded_delta", Title: "JUMI K8s Node Succeeded Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks K8s Jobs that reached Succeeded state during the window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_k8s_node_succeeded_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID: "jumi_k8s_node_failed_delta", Title: "JUMI K8s Node Failed Delta",
			Unit: "count", Kind: "delta_counter",
			Description: "Tracks K8s Jobs that reached Failed state during the window.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_k8s_node_failed_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
	}
}

// JUMIAHSmokeGuardrailSpecs returns smoke guardrail rules for the JUMI → AH path.
// Each rule fails if the expected activity did not occur during the window.
func JUMIAHSmokeGuardrailSpecs() []spec.SLISpec {
	return []spec.SLISpec{
		{
			ID: "jumi_jobs_created_smoke", Title: "JUMI Jobs Created Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should create at least two Jobs.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_jobs_created_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 2, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_artifacts_registered_smoke", Title: "JUMI Artifacts Registered Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should register at least one producer output through AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_artifacts_registered_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_input_resolve_requests_smoke", Title: "JUMI Input Resolve Requests Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should resolve at least one artifact binding.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_resolve_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_input_remote_fetch_smoke", Title: "JUMI Input Remote Fetch Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should observe at least one remote-fetch resolution decision.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_remote_fetch_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_input_materializations_smoke", Title: "JUMI Input Materializations Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should mark at least one resolved input as requiring materialization.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_input_materializations_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_sample_runs_finalized_smoke", Title: "JUMI Sample Runs Finalized Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should finalize at least one sample run through AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_sample_runs_finalized_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "jumi_gc_evaluate_requests_smoke", Title: "JUMI GC Evaluate Requests Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should trigger at least one GC evaluation request.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`jumi_gc_evaluate_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "ah_artifacts_registered_smoke", Title: "AH Artifacts Registered Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should register at least one artifact inside AH.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_artifacts_registered_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "ah_resolve_requests_smoke", Title: "AH Resolve Requests Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should receive at least one resolve request from JUMI.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_resolve_requests_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "ah_fallback_smoke", Title: "AH Fallback Smoke",
			Unit: "count", Kind: "delta_counter",
			Description: "Smoke run should exercise at least one remote-fetch fallback path.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_fallback_total`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeDelta},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}}},
		},
		{
			ID: "ah_gc_backlog_bytes_smoke", Title: "AH GC Backlog Bytes Smoke",
			Unit: "bytes", Kind: "gauge",
			Description: "Smoke run should not leave non-zero AH GC backlog at window end.",
			Inputs:      []spec.MetricRef{spec.UnsafePromKey(`ah_gc_backlog_bytes`)},
			Compute:     spec.ComputeSpec{Mode: spec.ComputeEnd},
			Judge:       &spec.JudgeSpec{Rules: []spec.Rule{{Op: spec.OpGT, Target: 0, Level: spec.LevelFail}}},
		},
	}
}
