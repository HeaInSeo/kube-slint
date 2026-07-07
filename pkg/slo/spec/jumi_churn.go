//go:build ignore

package spec

import "time"

// JUMIChurnSpecs returns SLI specs for JUMI K8s object churn measurement.
//
// These specs are consumed by a K8sObjectFetcher (pkg/slo/fetch/k8sobject)
// configured with:
//
//	Resource:        "pods" or "jobs"
//	MetricPrefix:    "k8s_jobs" / "k8s_pods"
//	ExcludeSelector: "app.kubernetes.io/managed-by=kube-slint"
//
// Counter reset policy for delta metrics is no_grade so that a JUMI operator
// restart during measurement produces NO_GRADE rather than a false PASS.
func JUMIChurnSpecs() []SLISpec {
	return []SLISpec{
		// --- Job lifecycle counters ---
		{
			ID:          "jumi_k8s_jobs_created_delta",
			Title:       "JUMI K8s Jobs Created (net delta)",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Net change in Job count during the measurement window. Excludes kube-slint-managed resources.",
			Inputs:      []MetricRef{UnsafePromKey("k8s_jobs_count")},
			Compute:     ComputeSpec{Mode: ComputeDelta, OnCounterReset: CounterResetNoGrade},
		},
		// --- Pod lifecycle counters ---
		{
			ID:          "jumi_k8s_pods_created_delta",
			Title:       "JUMI K8s Pods Created (net delta)",
			Unit:        "count",
			Kind:        "delta_counter",
			Description: "Net change in Pod count during the measurement window. Excludes kube-slint-managed resources.",
			Inputs:      []MetricRef{UnsafePromKey("k8s_pods_count")},
			Compute:     ComputeSpec{Mode: ComputeDelta, OnCounterReset: CounterResetNoGrade},
		},
		// --- End-of-window gauges ---
		{
			ID:          "jumi_k8s_orphan_objects_end",
			Title:       "JUMI K8s Orphan Objects (end)",
			Unit:        "count",
			Kind:        "gauge",
			Description: "Objects with no ownerReference at end of window — potential resource leak.",
			Inputs:      []MetricRef{UnsafePromKey("k8s_pods_orphan_end")},
			Compute:     ComputeSpec{Mode: ComputeEnd},
			Judge: &JudgeSpec{
				Rules: []Rule{
					{Op: OpGT, Target: 0, Level: LevelWarn},
				},
			},
		},
		{
			ID:    "jumi_k8s_ownerref_missing_end",
			Title: "JUMI K8s OwnerRef Missing (end)",
			Unit:  "count",
			Kind:  "gauge",
			// Same-kind-only check (see docs/DECISIONS.md D-018): this counts
			// objects whose ownerReference UID is absent from *this same
			// Resource kind's* listing. A Pod's usual owner (a ReplicaSet or
			// Job) is a different kind and will never appear here, so a
			// perfectly healthy Pod is indistinguishable from one whose owner
			// was actually deleted. Judged as Warn, not Fail, for that reason
			// — do not raise this to Fail without cross-kind owner
			// resolution.
			Description: "Objects whose ownerReference UID does not exist in the same-kind object set (same-kind check only; see D-018).",
			Inputs:      []MetricRef{UnsafePromKey("k8s_pods_ownerref_missing_end")},
			Compute:     ComputeSpec{Mode: ComputeEnd},
			Judge: &JudgeSpec{
				Rules: []Rule{
					{Op: OpGT, Target: 0, Level: LevelWarn},
				},
			},
		},
		{
			ID:          "jumi_k8s_stuck_terminating_end",
			Title:       "JUMI K8s Stuck Terminating (end)",
			Unit:        "count",
			Kind:        "gauge",
			Description: "Objects in Terminating state beyond the configured threshold at end of window.",
			Inputs:      []MetricRef{UnsafePromKey("k8s_pods_stuck_terminating_end")},
			Compute:     ComputeSpec{Mode: ComputeEnd},
			Judge: &JudgeSpec{
				Rules: []Rule{
					{Op: OpGT, Target: 0, Level: LevelFail},
				},
			},
		},
	}
}

// DefaultStuckTerminatingThreshold is the recommended threshold for JUMI churn gates.
// Pods terminating longer than this are flagged as stuck.
const DefaultStuckTerminatingThreshold = 5 * time.Minute
