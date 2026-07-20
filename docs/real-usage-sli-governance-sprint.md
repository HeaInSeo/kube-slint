# Real-Usage SLI Governance Hardening Sprint

Date: 2026-07-16
Status: Complete
Decision source: `docs/DECISIONS.md` D-029

## Confirmed Facts

- `docs/DECISIONS.md` D-001 defines kube-slint as a shift-left operational
  SLI guardrail, not a Prometheus-specific tool.
- `docs/DECISIONS.md` D-002 and D-008 separate measurement, policy
  evaluation, and CI failure.
- `docs/verification-sources.md` defines the current engine as a two-point
  model and records window/range sources as requiring a future engine
  extension.
- `pkg/slo/fetch.MetricsFetcher` and `SnapshotFetcher` are source-neutral
  interfaces, but current README/examples strongly bias the first-time user
  toward Prometheus text keys and `/metrics` scraping.

## Problem Statement

Real consumer usage found four product gaps:

1. An SLI can be defined and computed, then never asserted or covered by a
   gate policy. kube-slint currently does not report this as a governance
   problem.
2. Latency/window SLIs such as startup latency, gRPC request latency, p95/p99,
   and burn-rate checks do not fit the current two-point engine.
3. `UnsafePromKey` and Prometheus-key examples make the API feel more
   Prometheus-specific than the accepted product identity allows.
4. Non-Prometheus source adapters are possible, but common HTTP JSON/expvar
   boilerplate is still left to each consumer.

## Sprint A: Guardrail Coverage & Source-Neutral UX

Schedule: 2026-07-16 to 2026-07-19

Goal: make coverage gaps visible and reduce Prometheus-specific first-use
wording without changing library semantics.

Planned work:

- [x] Add diagnostics for measured SLIs that are not covered by threshold or
  regression policy.
- [x] Show a three-way relationship in onboarding/inspection output:
  profile-expected SLIs, measured SLIs, and policy-covered SLIs.
- [x] Keep the initial coverage signal advisory. Do not turn uncovered measured
  SLIs into automatic FAIL behavior in this sprint.
- [x] Update README/docs/examples so `MetricsFetcher`/`SnapshotFetcher` are
  described as source-neutral, and raw Prometheus helpers are presented as
  Prometheus-specific conveniences.
- [x] Prefer `PromMetric(name, labels)` in Prometheus examples where it is
  practical; keep `UnsafePromKey` documented as an escape hatch for raw
  exposition keys.
- [x] Add `InputKey(key)` as the source-neutral default helper for JSON,
  expvar, mock, or custom fetchers that already agree on canonical input keys.
- [x] Add guardrails so public onboarding examples do not drift back to
  `UnsafePromKey` for simple input keys or ordinary labeled Prometheus metrics.

Acceptance criteria:

- [x] `slint-gate inspect` identifies
  measured-but-not-policy-covered SLIs.
- [x] Docs do not imply Prometheus is required for kube-slint's product model.
- [x] No window/range SLI behavior is claimed as implemented.
- [x] `UnsafePromKey` remains supported, but examples reserve it for raw
  Prometheus text keys or PromQL expressions that safer helpers cannot express.

## Sprint B: Non-Prometheus Adapters & Window Design

Schedule: 2026-07-20 to 2026-07-26

Goal: prove source-neutral usage beyond Prometheus scrape while keeping
window/range semantics design-first.

Planned work:

- [x] Add a small HTTP JSON/expvar source adapter path
  (`pkg/slo/fetch/jsonendpoint`).
- [x] Factor common source-adapter boilerplate only where it reduces real duplication:
  HTTP GET, JSON flattening, and optional start-snapshot caching.
- [x] Decide whether the adapter starts as an example or a public package:
  `pkg/slo/fetch/jsonendpoint` is accepted as a small public adapter because it
  only depends on the existing stable `fetch.SnapshotFetcher` contract.
- [x] Draft the window/range SLI design for `WindowFetcher`, compute modes, and
  summary/gate compatibility.

Acceptance criteria:

- [x] A non-Prometheus source path is documented or demonstrated without requiring
  custom scratch code for every consumer.
- [x] Window/range SLI support has an accepted design boundary before any runtime
  implementation begins.

## Non-Goals

- Do not implement histogram quantiles, burn-rate/window_ratio semantics, or
  concrete PromQL range fetchers in the initial window engine foundation.
- Do not make coverage diagnostics fail CI from `inspect`; D-034 later makes
  gate coverage gaps strict by default in generated/default promotion behavior.
- Do not rename or remove `UnsafePromKey` in a breaking way.
- Do not turn kube-slint into a generic correctness test framework.

## Sprint C: Window Engine Foundation

Schedule: 2026-07-16 to 2026-07-19

Goal: implement the smallest useful runtime window path after the D-030 design
boundary, without changing the existing two-point behavior.

Planned work:

- [x] Promote `fetch.WindowFetcher` from commented future interface to real
  optional interface.
- [x] Add `ExecuteRequest.WindowFetcher` without changing `MetricsFetcher` or
  `SessionConfig`.
- [x] Add scalar window compute modes: `window_min`, `window_max`,
  `window_avg`, `window_p95`, and `window_p99`.
- [x] Preserve existing two-point behavior for `delta`/`start`/`end` specs.
- [x] Treat missing/failing window collection as summary-level skip/partial or
  failed collection, not a correctness-test failure.

Acceptance criteria:

- [x] Existing engine tests pass unchanged.
- [x] New tests cover window average, p95, missing window fetcher, and window
  fetch failure reliability state.
- [x] Gate and summary schemas remain unchanged for scalar window results.

## Open Risks

- `inspect` coverage diagnostics remain advisory; the gate path is now strict
  by default for coverage gaps after D-034.
- Prometheus-specific names remain in the public API for compatibility.
- A public JSON/expvar adapter package could become an API commitment before
  enough consumer usage exists.
- SessionConfig-level wiring for window fetchers is now implemented.
- Histogram quantiles and burn-rate semantics still require dedicated design.

## Sprint D: SessionConfig Window Wiring

Status: Complete

- [x] Added `SessionConfig.WindowFetcher`.
- [x] `Session.End()` passes the window fetcher into the engine.
- [x] Window-only specs avoid constructing the default curlpod point fetcher.

## Sprint E: Concrete WindowFetcher Source

Status: Complete

- [x] Added `pkg/slo/fetch/promrange` for Prometheus `/api/v1/query_range`.
- [x] Matrix results are converted into `[]fetch.Sample`.
- [x] Series keys are derived from Prometheus metric labels.

## Sprint F: Advanced Window Semantics

Status: Partial complete

- [x] Added `window_ratio` as `sum(input[0]) / sum(input[1])`.
- [x] Missing input or zero denominator produces a skipped SLI, not a
  misleading scalar.
- [ ] Histogram bucket quantiles remain unimplemented.
- [ ] Specialized burn-rate semantics beyond generic `window_ratio` remain
  unimplemented.

## Sprint G: Coverage Governance

Status: Complete

- [x] Added `policy.coverage.required`.
- [x] Added `policy.coverage.informational` to distinguish intentional
  informational SLIs from accidental coverage gaps.
- [x] Added `coverage_gap` as a `promote_to_fail` category.
- [x] D-034 later changed the default posture: generated/default promotion
  behavior is strict, while `coverage.required: false` remains the explicit
  opt-out.
