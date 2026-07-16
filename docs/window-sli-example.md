# Window SLI Example

This example shows the complete shape for a Prometheus range/window SLI. It
uses `pkg/slo/fetch/promrange` to query Prometheus `query_range`, computes
scalar window values in kube-slint, then gates those scalar results with the
normal policy path.

## Go Session

```go
import (
    "time"

    "github.com/HeaInSeo/kube-slint/pkg/slint"
    "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promrange"
    "github.com/HeaInSeo/kube-slint/pkg/slo/spec"
)

windowFetcher := promrange.New(
    "http://prometheus.monitoring.svc:9090",
    `http_request_duration_seconds`,
    30*time.Second,
)

windowSpecs := []spec.SLISpec{
    {
        ID:    "http_request_duration_p95",
        Title: "HTTP request duration p95",
        Unit:  "seconds",
        Kind:  "window_latency",
        Inputs: []spec.MetricRef{
            spec.UnsafePromKey("http_request_duration_seconds"),
        },
        Compute: spec.ComputeSpec{Mode: spec.ComputeWindowP95},
    },
    {
        ID:    "http_error_ratio",
        Title: "HTTP error ratio",
        Unit:  "ratio",
        Kind:  "window_ratio",
        Inputs: []spec.MetricRef{
            spec.UnsafePromKey(`rate(http_requests_total{code=~"5.."}[5m])`),
            spec.UnsafePromKey(`rate(http_requests_total[5m])`),
        },
        Compute: spec.ComputeSpec{Mode: spec.ComputeWindowRatio},
    },
}

sess := slint.NewSession(slint.SessionConfig{
    Namespace:     "my-operator-system",
    TestCase:      "window-sli",
    Specs:         windowSpecs,
    WindowFetcher: windowFetcher,
    ArtifactsDir:  "artifacts",
})

sess.Start()
// run workload
_, _ = sess.End(ctx)
```

## Policy

```yaml
schema_version: "slint.policy.v1"

thresholds:
  - name: "http_request_duration_p95_max"
    metric: "http_request_duration_p95"
    operator: "<="
    value: 0.5
  - name: "http_error_ratio_max"
    metric: "http_error_ratio"
    operator: "<="
    value: 0.01

regression:
  enabled: false
  tolerance_percent: 10

reliability:
  required: false
  min_level: "partial"

coverage:
  required: true
  informational: []

promote_to_fail:
  - "threshold_miss"
  # Optional:
  # - "coverage_gap"
```

## Boundaries

- `window_p95` computes a nearest-rank percentile over scalar sample values.
  It does not compute Prometheus histogram bucket quantiles.
- `window_ratio` computes `sum(input[0]) / sum(input[1])` over the returned
  samples.
- Specialized burn-rate semantics remain separate future work.
