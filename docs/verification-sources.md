# Verification Sources — Design Boundary

kube-slint has three user-facing source terms:

- point source: a `MetricsFetcher` that returns one keyed numeric sample per
  fetch.
- snapshot source: a `SnapshotFetcher` that can pre-capture the start sample at
  `Session.Start()` before the workload runs.
- range/window source: a `WindowFetcher` that returns many samples for a time
  window.

The default engine path is still a **2-point model**: it calls
`MetricsFetcher.Fetch()` twice (at `StartedAt` and `FinishedAt`) and evaluates
the delta or snapshot between those two scalar maps.

This document draws the boundary between what the current engine can support and what requires
a future engine extension.

## Tier 1 — Works with the current 2-point engine

These source types return one `fetch.Sample` per call and fit directly into the
existing point/snapshot source interfaces.

| Source type | Interface | Notes |
|---|---|---|
| `point_scrape` | `MetricsFetcher` | HTTP GET /metrics at a point in time |
| `http_json` | `SnapshotFetcher` | HTTP JSON endpoint; numeric leaves flatten to dot-separated input keys |
| `expvar_json` | `SnapshotFetcher` | Go expvar `/debug/vars`; same JSON flattening path as `http_json` |
| `portforward` | `SnapshotFetcher` | kubectl port-forward + HTTP scrape; PreFetch caches start snapshot |
| `curlpod` | `SnapshotFetcher` | in-cluster curl Pod; PreFetch caches start snapshot |
| `baseline_compare` | _(file-side)_ | Load a prior `sli-summary.json` as baseline for regression |

Adding a new Tier 1 source requires only implementing `MetricsFetcher` (or optionally `SnapshotFetcher`);
the engine, spec, and gate layers need no changes.

## Tier 2 — Range/window sources

These source types need N samples over a time window, not two discrete scalars.
They **cannot** be added by implementing `MetricsFetcher` alone. The initial
window engine path supports scalar aggregations (`window_min`, `window_max`,
`window_avg`, `window_p95`, `window_p99`, `window_ratio`) through
`fetch.WindowFetcher`.

| Source type | Blocker | Description |
|---|---|---|
| `promql_query` | Range result ([]Sample) | Implemented by `pkg/slo/fetch/promrange` for Prometheus `query_range` matrix results |
| `soak_analysis_run` | Multi-point aggregation | Requires p50/p95/p99 over the full window |
| `burn_rate` | Sliding window ratio | Error budget burn over a look-back period |
| `p95_over_window` | Histogram aggregation | Needs raw bucket series, not two snapshots |

### What extension is needed

`WindowFetcher` returns `[]Sample` for a `(start, end time.Time)` range. The
engine request carries an optional `WindowFetcher` alongside the existing
`MetricsFetcher`. `pkg/slint.SessionConfig` also accepts an optional
`WindowFetcher` for consumer-facing sessions.

## Design rule

> A new measurement source is Tier 1 if and only if its result can be expressed as
> `map[string]float64` at a single point in time.
> Everything else is Tier 2.

When evaluating a new source, apply this rule first. If Tier 2, open a design discussion
before writing code.
