# Window-Based SLI Design

Date: 2026-07-16
Status: Proposed design boundary; not implemented
Decision source: `docs/DECISIONS.md` D-029

## Confirmed Facts

- The current engine is a two-point model: it fetches one start sample and one
  end sample, then evaluates each `SLISpec` against those scalar maps.
- `fetch.WindowFetcher` is intentionally documented as future work in
  `pkg/slo/fetch/fetcher.go`.
- `docs/verification-sources.md` says range/window sources require an engine
  extension and must not be added by implementing `MetricsFetcher` alone.

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

The future engine request should carry a window fetcher separately from the
current `MetricsFetcher`. Two-point SLIs continue using the current path.
Window SLIs opt into a new compute family and cannot silently fall back to
start/end delta semantics.

## Proposed Spec Shape

New compute modes should be explicit and narrow, for example:

- `window_min`
- `window_max`
- `window_avg`
- `window_p95`
- `window_p99`
- `window_ratio`

Each mode must define what input keys it consumes and how missing/empty windows
map to `StatusSkip`/`NO_GRADE`.

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

- Whether the first implementation should support generic percentile over a
  sequence of scalar samples, histogram bucket quantiles, or both.
- Whether `WindowFetcher` should return ordered samples by contract or whether
  the engine should sort by `Sample.At`.
- Whether empty windows should be `StatusSkip` with `NO_GRADE` semantics by
  default.
- Whether startup latency belongs in `WindowFetcher` or should be modeled as a
  small event-pair fetcher that emits a scalar directly.
