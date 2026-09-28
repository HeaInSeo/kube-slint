package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch"
	"github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promtext"
	"github.com/HeaInSeo/kube-slint/pkg/slo/spec"
	"github.com/HeaInSeo/kube-slint/pkg/slo/summary"
)

// KSL-E2P fixtures: controller-runtime series shapes as exposed by the Bori
// operator (three controllers, result-labelled reconcile counters, full
// workqueue label set, client-go REST counters by code/method/host).
const e2pStartText = `
controller_runtime_reconcile_total{controller="boridataplane",result="error"} 1
controller_runtime_reconcile_total{controller="boridataplane",result="requeue"} 0
controller_runtime_reconcile_total{controller="boridataplane",result="requeue_after"} 2
controller_runtime_reconcile_total{controller="boridataplane",result="success"} 10
controller_runtime_reconcile_total{controller="boriother",result="success"} 500
controller_runtime_reconcile_total{controller="borithird",result="success"} 70
controller_runtime_reconcile_errors_total{controller="boridataplane"} 1
controller_runtime_reconcile_errors_total{controller="boriother"} 9
workqueue_depth{controller="boridataplane",name="boridataplane",priority=""} 3
workqueue_depth{controller="boriother",name="boriother",priority=""} 40
rest_client_requests_total{code="200",host="10.96.0.1:443",method="GET"} 1000
rest_client_requests_total{code="500",host="10.96.0.1:443",method="GET"} 1
rest_client_requests_total{code="503",host="10.96.0.1:443",method="PATCH"} 2
`

const e2pEndText = `
controller_runtime_reconcile_total{controller="boridataplane",result="error"} 2
controller_runtime_reconcile_total{controller="boridataplane",result="requeue"} 1
controller_runtime_reconcile_total{controller="boridataplane",result="requeue_after"} 4
controller_runtime_reconcile_total{controller="boridataplane",result="success"} 20
controller_runtime_reconcile_total{controller="boriother",result="success"} 900
controller_runtime_reconcile_total{controller="borithird",result="success"} 75
controller_runtime_reconcile_errors_total{controller="boridataplane"} 2
controller_runtime_reconcile_errors_total{controller="boriother"} 30
workqueue_depth{controller="boridataplane",name="boridataplane",priority=""} 5
workqueue_depth{controller="boriother",name="boriother",priority=""} 41
rest_client_requests_total{code="200",host="10.96.0.1:443",method="GET"} 1500
rest_client_requests_total{code="500",host="10.96.0.1:443",method="GET"} 3
rest_client_requests_total{code="503",host="10.96.0.1:443",method="PATCH"} 2
rest_client_requests_total{code="504",host="10.96.0.1:443",method="GET"} 4
rest_client_requests_total{code="5000",host="10.96.0.1:443",method="GET"} 99
`

// pointFetcher returns start values for StartedAt and end values otherwise.
type pointFetcher struct {
	startAt    time.Time
	start, end map[string]float64
}

func (p *pointFetcher) Fetch(_ context.Context, at time.Time) (fetch.Sample, error) {
	if at.Equal(p.startAt) {
		return fetch.Sample{At: at, Values: p.start}, nil
	}
	return fetch.Sample{At: at, Values: p.end}, nil
}

func parseE2P(t *testing.T, text string) map[string]float64 {
	t.Helper()
	// Fetchers apply Aggregate, so bare-name sums are present too; selectors
	// must never pick them up.
	m, err := promtext.ParseTextToMapWithAggregates(strings.NewReader(text))
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return m
}

func e2pSpecs() []spec.SLISpec {
	delta := spec.ComputeSpec{Mode: spec.ComputeDelta}
	return []spec.SLISpec{
		{
			ID: "reconcile-delta", Unit: "count", Kind: "delta_counter", Compute: delta,
			Inputs: []spec.MetricRef{spec.SelectMetric("controller_runtime_reconcile_total", spec.AggregateSum,
				spec.LabelEq("controller", "boridataplane"))},
		},
		{
			ID: "reconcile-errors-delta", Unit: "count", Kind: "delta_counter", Compute: delta,
			Inputs: []spec.MetricRef{spec.SelectMetric("controller_runtime_reconcile_errors_total", spec.AggregateNone,
				spec.LabelEq("controller", "boridataplane"))},
		},
		{
			ID: "workqueue-depth-end", Unit: "count", Kind: "gauge", Compute: spec.ComputeSpec{Mode: spec.ComputeEnd},
			Inputs: []spec.MetricRef{spec.SelectMetric("workqueue_depth", spec.AggregateNone,
				spec.LabelEq("name", "boridataplane"), spec.LabelEq("controller", "boridataplane"), spec.LabelEq("priority", ""))},
		},
		{
			ID: "rest-errors-delta", Unit: "count", Kind: "delta_counter", Compute: delta,
			Inputs: []spec.MetricRef{spec.SelectMetric("rest_client_requests_total", spec.AggregateSum,
				spec.LabelRegexp("code", "^5[0-9][0-9]$"))},
		},
	}
}

func runE2P(t *testing.T, tc *TrustContract, start, end map[string]float64, specs ...spec.SLISpec) (*summary.Summary, error) {
	t.Helper()
	startAt := time.Unix(1000, 0)
	eng := New(&pointFetcher{startAt: startAt, start: start, end: end}, &mockWriter{}, nil)
	return eng.Execute(context.Background(), ExecuteRequest{
		Config:      RunConfig{StartedAt: startAt, FinishedAt: time.Unix(1060, 0), TrustContract: tc},
		Specs:       specs,
		Reliability: &summary.Reliability{},
	})
}

func resultByID(t *testing.T, sum *summary.Summary, id string) summary.SLIResult {
	t.Helper()
	for _, r := range sum.Results {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("result %q not found", id)
	return summary.SLIResult{}
}

func TestSelector_E2PFourIntentsProtected(t *testing.T) {
	sum, err := runE2P(t, e1TrustContract(), parseE2P(t, e2pStartText), parseE2P(t, e2pEndText), e2pSpecs()...)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sum.SchemaVersion != summary.SchemaVersionTrust {
		t.Fatalf("schema = %q, want %q", sum.SchemaVersion, summary.SchemaVersionTrust)
	}
	if err := summary.Validate(*sum); err != nil {
		t.Fatalf("slo.v4 summary must validate: %v", err)
	}
	want := map[string]float64{
		"reconcile-delta":        (2 + 1 + 4 + 20) - (1 + 0 + 2 + 10), // sum across result, other controllers excluded
		"reconcile-errors-delta": 2 - 1,
		"workqueue-depth-end":    5,
		"rest-errors-delta":      (3 + 2 + 4) - (1 + 2), // 5xx across method/host; "5000" excluded by anchoring
	}
	for id, v := range want {
		r := resultByID(t, sum, id)
		if r.Status != summary.StatusPass || r.Value == nil || *r.Value != v {
			t.Errorf("%s: status=%s value=%v reason=%q, want pass %v", id, r.Status, r.Value, r.Reason, v)
		}
		if r.Comparability == nil || !r.Comparability.Complete() {
			t.Errorf("%s: comparability identity must be complete", id)
		}
	}
	if len(sum.Reliability.MissingInputs) != 0 || len(sum.Reliability.SkippedSLIs) != 0 {
		t.Fatalf("no input may be missing or skipped: %+v", sum.Reliability)
	}
}

func TestSelector_E2PDeterministicAcrossRuns(t *testing.T) {
	start, end := parseE2P(t, e2pStartText), parseE2P(t, e2pEndText)
	first, err := runE2P(t, e1TrustContract(), start, end, e2pSpecs()...)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for i := 0; i < 20; i++ {
		// Fresh maps each iteration so map iteration order varies.
		again, err := runE2P(t, e1TrustContract(), parseE2P(t, e2pStartText), parseE2P(t, e2pEndText), e2pSpecs()...)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		for j := range first.Results {
			a, b := first.Results[j], again.Results[j]
			if *a.Value != *b.Value || a.Comparability.SLIContractID != b.Comparability.SLIContractID {
				t.Fatalf("%s not deterministic: %v/%s vs %v/%s", a.ID, *a.Value, a.Comparability.SLIContractID, *b.Value, b.Comparability.SLIContractID)
			}
		}
	}
}

func TestSelector_ZeroMatchFailsClosed(t *testing.T) {
	s := spec.SLISpec{
		ID: "rest-errors-delta", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		Inputs: []spec.MetricRef{spec.SelectMetric("rest_client_requests_total", spec.AggregateSum,
			spec.LabelRegexp("code", "^5[0-9][0-9]$"))},
	}
	only2xx := map[string]float64{`rest_client_requests_total{code="200",host="h",method="GET"}`: 1}
	sum, err := runE2P(t, nil, only2xx, only2xx, s)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	r := sum.Results[0]
	if r.Status != summary.StatusSkip || r.Value != nil {
		t.Fatalf("zero match must skip without value, got %s %v", r.Status, r.Value)
	}
	if !strings.Contains(r.Reason, "matched no series") || len(r.InputsMissing) != 1 {
		t.Fatalf("reason=%q missing=%v", r.Reason, r.InputsMissing)
	}
}

func TestSelector_UndeclaredMultiMatchFailsClosed(t *testing.T) {
	s := spec.SLISpec{
		ID: "reconcile-delta", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		Inputs: []spec.MetricRef{spec.SelectMetric("controller_runtime_reconcile_total", spec.AggregateNone,
			spec.LabelEq("controller", "boridataplane"))},
	}
	sum, err := runE2P(t, nil, parseE2P(t, e2pStartText), parseE2P(t, e2pEndText), s)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	r := sum.Results[0]
	if r.Status != summary.StatusSkip || r.Value != nil || !strings.Contains(r.Reason, "without declared aggregation") {
		t.Fatalf("undeclared multi-match must skip: %s %v %q", r.Status, r.Value, r.Reason)
	}
}

func TestSelector_MalformedRejectedBeforeFetch(t *testing.T) {
	bad := []spec.SLISpec{
		{ID: "bad-regexp", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			Inputs: []spec.MetricRef{spec.SelectMetric("m", spec.AggregateSum, spec.LabelRegexp("code", "("))}},
		{ID: "bad-mode", Compute: spec.ComputeSpec{Mode: spec.ComputeWindowAvg},
			Inputs: []spec.MetricRef{spec.SelectMetric("m", spec.AggregateSum, spec.LabelEq("a", "x"))}},
		{ID: "bad-agg", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			Inputs: []spec.MetricRef{spec.SelectMetric("m", spec.Aggregation("max"), spec.LabelEq("a", "x"))}},
	}
	for _, s := range bad {
		t.Run(s.ID, func(t *testing.T) {
			w := &mockWriter{}
			eng := New(&mockStaticFetcher{values: map[string]float64{}}, w, nil)
			_, err := eng.Execute(context.Background(), ExecuteRequest{
				Config: RunConfig{StartedAt: time.Unix(1, 0), FinishedAt: time.Unix(2, 0)},
				Specs:  []spec.SLISpec{s},
			})
			if err == nil {
				t.Fatal("malformed selector must fail Execute")
			}
			if w.lastWritten != nil {
				t.Fatal("no summary may be written for a malformed selector")
			}
		})
	}
}

func TestSelector_ContractIDDistinctFromExactKey(t *testing.T) {
	sel := spec.SLISpec{ID: "x", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		Inputs: []spec.MetricRef{spec.SelectMetric("m", spec.AggregateSum, spec.LabelEq("a", "b"))}}
	exact := spec.SLISpec{ID: "x", Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
		Inputs: []spec.MetricRef{spec.PromMetric("m", spec.Labels{"a": "b"})}}
	if sliContractID(sel) == sliContractID(exact) {
		t.Fatal("a selector input must not share an identity with an exact-key input")
	}
	one := sel
	one.Inputs = []spec.MetricRef{spec.SelectMetric("m", spec.AggregateNone, spec.LabelEq("a", "b"))}
	if sliContractID(sel) == sliContractID(one) {
		t.Fatal("declared aggregation must be part of the identity")
	}
}

// Legacy exact-key identities are pinned to the values computed at
// main@a2407ff, before selectors existed, so slo.v4 identities of existing
// callers cannot drift.
func TestSelector_LegacyExactKeyContractIDPinned(t *testing.T) {
	legacy := []struct {
		spec spec.SLISpec
		want string
	}{
		{spec.SLISpec{ID: "reconcile-errors-delta", Unit: "count", Kind: "delta_counter",
			Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
			Inputs:  []spec.MetricRef{spec.PromMetric("controller_runtime_reconcile_errors_total", spec.Labels{"controller": "boridataplane"})}},
			"slic-v1-6be3e3c3b1b844381feaabdbfca90b20"},
		{spec.SLISpec{ID: "latency_p95", Unit: "ms", Kind: "latency",
			Compute: spec.ComputeSpec{Mode: spec.ComputeWindowP95},
			Inputs:  []spec.MetricRef{{Key: "request_ms"}, {Key: "request_ms"}}},
			"slic-v1-de3c74387942360611328b9f3a037774"},
	}
	for _, l := range legacy {
		if got := sliContractID(l.spec); got != l.want {
			t.Errorf("%s: SLIContractID = %s, want pinned %s", l.spec.ID, got, l.want)
		}
	}
}
