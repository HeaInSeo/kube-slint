# Verification Sources — Design Boundary

kube-slint's engine is currently a **2-point model**: it calls `MetricsFetcher.Fetch()` twice
(at `StartedAt` and `FinishedAt`) and evaluates the delta or snapshot between those two scalars.

This document draws the boundary between what the current engine can support and what requires
a future engine extension.

## Tier 1 — Works with the current 2-point engine

These source types return one `fetch.Sample` per call and fit directly into the existing
`MetricsFetcher` / `SnapshotFetcher` interfaces.

| Source type | Interface | Notes |
|---|---|---|
| `point_scrape` | `MetricsFetcher` | HTTP GET /metrics at a point in time |
| `portforward` | `MetricsFetcher` | kubectl port-forward + HTTP scrape |
| `curlpod` | `SnapshotFetcher` | in-cluster curl Pod; PreFetch caches start snapshot |
| `baseline_compare` | _(file-side)_ | Load a prior `sli-summary.json` as baseline for regression |

Adding a new Tier 1 source requires only implementing `MetricsFetcher` (or optionally `SnapshotFetcher`);
the engine, spec, and gate layers need no changes.

## Tier 2 — Requires engine extension

These source types need N samples over a time window, not two discrete scalars.
They **cannot** be added by implementing `MetricsFetcher` alone.

| Source type | Blocker | Description |
|---|---|---|
| `promql_query` | Range result ([]Sample) | PromQL `range_query` returns a matrix, not a scalar |
| `soak_analysis_run` | Multi-point aggregation | Requires p50/p95/p99 over the full window |
| `burn_rate` | Sliding window ratio | Error budget burn over a look-back period |
| `p95_over_window` | Histogram aggregation | Needs raw bucket series, not two snapshots |

### What extension is needed

A `WindowFetcher` interface (stub defined in `pkg/slo/fetch/fetcher.go`) would return `[]Sample`
for a `(start, end time.Time)` range. The engine would need:

1. A new `ComputeMode` (e.g. `ComputeP95`, `ComputeSoak`) that accepts `[]Sample` instead of two maps.
2. `evalSLI` to branch on whether the spec's compute mode is 2-point or window-based.
3. Engine request to carry an optional `WindowFetcher` alongside the existing `MetricsFetcher`.

This is a **breaking change to the engine API** and must be designed before any implementation
to avoid fragmenting the existing clean 2-point path.

## Design rule

> A new measurement source is Tier 1 if and only if its result can be expressed as
> `map[string]float64` at a single point in time.
> Everything else is Tier 2.

When evaluating a new source, apply this rule first. If Tier 2, open a design discussion
before writing code.
