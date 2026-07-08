//go:build kind

// Package e2e demonstrates end-to-end kube-slint integration with hello-operator.
//
// Prerequisites (see README for full setup):
//
//	kind create cluster --name slint-demo
//	kubectl apply -f manifests/
//	kind load docker-image hello-operator:dev --name slint-demo
//
// Run:
//
//	SLINT_SA_TOKEN=$(kubectl -n hello-system create token kube-slint) \
//	  go test ./e2e/... -v -timeout 120s
package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slint"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
)

// helloOperatorSpecs defines what kube-slint should measure on hello-operator.
func helloOperatorSpecs() []spec.SLISpec {
	return []spec.SLISpec{
		{
			ID:      "hello_reconcile_delta",
			Title:   "Hello Reconcile Delta",
			Unit:    "count",
			Kind:    "delta_counter",
			Inputs:  []spec.MetricRef{spec.UnsafePromKey("hello_reconcile_total")},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			// At least one reconcile must have fired.
			Judge: &spec.JudgeSpec{
				Rules: []spec.Rule{{Op: spec.OpLT, Target: 1, Level: spec.LevelFail}},
			},
		},
		{
			ID:      "hello_reconcile_errors_delta",
			Title:   "Hello Reconcile Errors Delta",
			Unit:    "count",
			Kind:    "delta_counter",
			Inputs:  []spec.MetricRef{spec.UnsafePromKey("hello_reconcile_errors_total")},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
		{
			ID:      "hello_workqueue_depth_end",
			Title:   "Hello Workqueue Depth",
			Unit:    "count",
			Kind:    "gauge",
			Inputs:  []spec.MetricRef{spec.UnsafePromKey("hello_workqueue_depth")},
			Compute: spec.ComputeSpec{Mode: spec.ComputeEnd},
		},
		{
			ID:      "hello_workqueue_adds_delta",
			Title:   "Hello Workqueue Adds Delta",
			Unit:    "count",
			Kind:    "delta_counter",
			Inputs:  []spec.MetricRef{spec.UnsafePromKey("hello_workqueue_adds_total")},
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		},
	}
}

func TestHelloOperatorSLI(t *testing.T) {
	// 1. Get the bearer token for the kube-slint ServiceAccount.
	//    In CI: export SLINT_SA_TOKEN=$(kubectl -n hello-system create token kube-slint)
	//    On a real cluster pod: token is at /var/run/secrets/kubernetes.io/serviceaccount/token
	token, err := slint.ReadServiceAccountTokenFromEnv("SLINT_SA_TOKEN", "")
	if err != nil {
		t.Skipf("SLINT_SA_TOKEN not set — skipping live cluster test: %v", err)
	}

	// 2. Build the session.
	sess := slint.NewSession(slint.SessionConfig{
		Namespace:          "hello-system",
		MetricsServiceName: "hello-operator-metrics",
		ServiceAccountName: "kube-slint",
		Token:              token,
		ArtifactsDir:       "../artifacts", // relative to e2e/ → kind-hello-operator/artifacts/
		Specs:              helloOperatorSpecs(),
		// hello-operator uses plain HTTP on port 8080, not HTTPS.
		ServiceURLFormat:      slint.ServiceURLHTTP,
		TLSInsecureSkipVerify: false,
	})

	// 3. Start: captures the pre-workload metrics snapshot.
	sess.Start()

	// 4. Simulate workload — hello-operator runs its reconcile loop in the background.
	t.Log("Waiting 10s for hello-operator to emit metrics...")
	time.Sleep(10 * time.Second)

	// 5. End: captures the post-workload snapshot, computes deltas, writes artifacts.
	sum, err := sess.End(context.Background())
	if err != nil {
		t.Logf("kube-slint End() warning: %v", err)
	}
	if sum == nil {
		t.Fatal("expected non-nil summary")
	}

	// 6. Report results.
	t.Logf("Reliability: %s", sum.Reliability.CollectionStatus)
	for _, r := range sum.Results {
		val := "<nil>"
		if r.Value != nil {
			val = fmt.Sprintf("%v", *r.Value)
		}
		t.Logf("  %-40s status=%-8s value=%s", r.ID, r.Status, val)
	}

	// 7. Optionally evaluate policy gate inline.
	summaryPath := "../artifacts/sli-summary.json"
	if _, statErr := os.Stat(summaryPath); statErr == nil {
		t.Logf("Artifact written: %s", summaryPath)
		t.Log("Run slint-gate to evaluate against policy (from kind-hello-operator/):")
		t.Log("  go run ../../cmd/slint-gate --summary artifacts/sli-summary.json --policy .slint/policy.yaml --exit-on FAIL_OR_NOGRADE")
	}
}
