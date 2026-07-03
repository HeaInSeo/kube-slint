package promtext

import (
	"strings"
	"testing"
)

func TestAggregate_SumsLabeledSeriesUnderBaseName(t *testing.T) {
	m := map[string]float64{
		`reconcile_total{controller="a"}`: 2,
		`reconcile_total{controller="b"}`: 3,
	}
	out := Aggregate(m)
	if out["reconcile_total"] != 5 {
		t.Fatalf("expected aggregated reconcile_total=5, got %v", out["reconcile_total"])
	}
	// original labeled keys must still be present
	if out[`reconcile_total{controller="a"}`] != 2 {
		t.Fatalf("expected original labeled key to be preserved")
	}
}

func TestAggregate_DoesNotDoubleCountRealBareSeries(t *testing.T) {
	// Regression test for R3/F3: if the raw text already has its own unlabeled
	// series for a name, the aggregate must not add labeled values on top of it.
	m := map[string]float64{
		"reconcile_total":                 100,
		`reconcile_total{controller="a"}`: 2,
	}
	out := Aggregate(m)
	if out["reconcile_total"] != 100 {
		t.Fatalf("expected real unlabeled series to be left untouched, got %v", out["reconcile_total"])
	}
}

func TestAggregate_ExcludesHistogramBuckets(t *testing.T) {
	// Regression test for R3/F3: histogram buckets are cumulative, not
	// additive — summing them under the bare metric name is meaningless.
	m := map[string]float64{
		`http_request_duration_seconds_bucket{le="0.1"}`:  3,
		`http_request_duration_seconds_bucket{le="0.5"}`:  7,
		`http_request_duration_seconds_bucket{le="+Inf"}`: 9,
	}
	out := Aggregate(m)
	if _, ok := out["http_request_duration_seconds_bucket"]; ok {
		t.Fatalf("histogram buckets must not be aggregated under the base name, got %v", out)
	}
}

func TestAggregate_ExcludesSummaryQuantiles(t *testing.T) {
	// Regression test for R3/F3: summary quantiles are positional, not
	// additive.
	m := map[string]float64{
		`request_latency{quantile="0.5"}`: 12,
		`request_latency{quantile="0.9"}`: 40,
	}
	out := Aggregate(m)
	if _, ok := out["request_latency"]; ok {
		t.Fatalf("summary quantiles must not be aggregated under the base name, got %v", out)
	}
}

func TestParseTextToMapWithAggregates(t *testing.T) {
	raw := `rest_client_requests_total{code="200",method="GET"} 5
rest_client_requests_total{code="500",method="GET"} 1
`
	out, err := ParseTextToMapWithAggregates(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("ParseTextToMapWithAggregates returned error: %v", err)
	}
	if out["rest_client_requests_total"] != 6 {
		t.Fatalf("expected aggregated rest_client_requests_total=6, got %v", out["rest_client_requests_total"])
	}
}
