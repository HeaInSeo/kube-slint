# Window-Based SLI Design

Date: 2026-07-16
Status: Initial scalar window aggregation implemented
Decision source: `docs/DECISIONS.md` D-029, D-030, D-031

## Confirmed Facts

- The current engine is a two-point model: it fetches one start sample and one
  end sample, then evaluates each `SLISpec` against those scalar maps.
- `fetch.WindowFetcher` is implemented as an optional source interface in
  `pkg/slo/fetch/fetcher.go`.
- The engine supports scalar window aggregation modes:
  `window_min`, `window_max`, `window_avg`, `window_p95`, and `window_p99`.
- `docs/verification-sources.md` still treats range/window sources as a
  separate source class. They must use `WindowFetcher`, not a two-point
  `MetricsFetcher` workaround.

## Target Use Cases

- Startup latency measured as elapsed time between two semantic events.
- Request latency percentiles over a test window, such as p95/p99.
- Burn-rate or error-ratio checks over a look-back window.
- Soak-style aggregations over more than two samples.

## Proposed Boundary

Window-based SLIs should enter the engine through a separate source contract,
not through overloaded two-point samples.

```go
type WindowFetcher interface {
    FetchRange(ctx context.Context, start, end time.Time) ([]fetch.Sample, error)
}
```

The engine request carries a window fetcher separately from the current
`MetricsFetcher`. Two-point SLIs continue using the current path. Window SLIs
opt into a new compute family and cannot silently fall back to start/end delta
semantics.

## Proposed Spec Shape

Implemented compute modes are explicit and narrow:

- `window_min`
- `window_max`
- `window_avg`
- `window_p95`
- `window_p99`

Each mode consumes numeric values for `SLISpec.Inputs` across the returned
window samples. Missing or empty windows map to `StatusSkip`; gate evaluation
then treats the missing scalar conservatively through the existing summary
contract.

## Summary And Gate Compatibility

The summary result shape can likely stay `id/value/status/reason` for the first
implementation, because a window SLI still emits one scalar value for policy
evaluation. If later diagnostics need raw sample counts or histogram bucket
metadata, add optional fields instead of changing the existing result contract.

The gate layer should not know whether a scalar came from two-point or window
computation. Threshold and regression checks should continue to operate over
`Summary.ResultValues()`.

## Non-Goals

- Do not compute p95/p99 from only two samples.
- Do not encode window series into synthetic JSON strings inside
  `fetch.Sample.Values`.
- Do not make PromQL range queries the only window source.
- Do not change the existing `MetricsFetcher` contract.

## Open Questions

- Whether a future implementation should add histogram bucket quantiles in
  addition to the implemented generic percentile over scalar samples.
- Whether `WindowFetcher` should return ordered samples by contract or whether
  the engine should sort by `Sample.At`.
- Whether empty windows should be `StatusSkip` with `NO_GRADE` semantics by
  default at the gate policy layer, beyond the current skipped result.
- Whether startup latency belongs in `WindowFetcher` or should be modeled as a
  small event-pair fetcher that emits a scalar directly.
- Whether to add `window_ratio`/burn-rate compute modes once a concrete source
  shape exists.
