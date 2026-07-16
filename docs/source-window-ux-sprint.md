# Source And Window UX Sprint

Date: 2026-07-16
Status: Sprint U4 complete; U5 pending
Decision source: `docs/DECISIONS.md` D-033

## Confirmed Facts

- `slint-gate inspect` can surface measured-but-not-policy-covered SLIs.
- `policy.coverage.required` and `coverage.informational` provide opt-in
  governance for measured-but-not-gated SLIs.
- `SessionConfig.WindowFetcher`, `pkg/slo/fetch/promrange`, and scalar window
  compute modes are implemented.
- The remaining UX gap is not capability availability; it is guiding users
  toward the right source, spec, and policy action.

## Sprint U1: Coverage Gap Next Actions

Schedule: 2026-07-17

Goal: make `inspect` output actionable when it finds measured-but-not-covered
SLIs.

Planned work:

- [x] For each uncovered measured SLI, recommend one of:
  - add a threshold rule;
  - add it to `coverage.informational`;
  - remove/ignore the signal if it is accidental.
- [x] Keep the recommendation advisory. Do not change gate behavior.

Acceptance criteria:

- [x] `slint-gate inspect --policy` output includes concrete next-action wording
  for coverage gaps.
- [x] Existing gate semantics remain unchanged.

## Sprint U2: Source Selection Guide

Schedule: 2026-07-17

Goal: help users choose the correct source adapter before writing code.

Planned work:

- [x] Add a docs/README table mapping common scenarios to source types:
  default curlpod, portforward, jsonendpoint, and promrange.
- [x] Use user-facing terms alongside Go names:
  "point source", "snapshot source", "range/window source".

Acceptance criteria:

- [x] A new user can identify the likely fetcher for `/metrics`, local
  port-forward, expvar/status JSON, and Prometheus range query use cases.

## Sprint U3: Window SLI End-To-End Example

Schedule: 2026-07-18

Goal: show the complete path for range/window SLIs.

Planned work:

- [x] Add a small example or docs section using `promrange.New(...)`.
- [x] Include one `window_p95` example and one `window_ratio` example.
- [x] Include matching policy threshold examples.

Acceptance criteria:

- [x] The example shows `SessionConfig.WindowFetcher`, `SLISpec`, and policy YAML
  together.
- [x] The example does not imply histogram quantiles or specialized burn-rate
  semantics are implemented.

## Sprint U4: Coverage Policy Recommendation Flow

Schedule: 2026-07-18 to 2026-07-19

Goal: reduce manual policy editing for coverage governance.

Planned work:

- [x] Teach policy/profile examples how to mark known informational SLIs.
- [x] Consider whether `recommend-policy` should emit a commented
  `coverage.informational` block for informational profile candidates.
- [x] Keep generated hard-fail behavior conservative.

Acceptance criteria:

- [x] Users have a clear path from coverage warning to policy edit.
- [x] `coverage_gap` is not promoted to FAIL by default.

## Sprint U5: Terminology Pass

Schedule: 2026-07-19

Goal: make source-neutral terminology consistent.

Planned work:

- Use "point source", "snapshot source", and "range/window source" in user
  docs where helpful.
- Preserve public Go API names: `MetricsFetcher`, `SnapshotFetcher`,
  `WindowFetcher`.
- Avoid implying kube-slint is Prometheus-only.

Acceptance criteria:

- README and onboarding docs use consistent source terminology.
- Existing product identity guardrails still pass.

## Non-Goals

- No new engine compute modes.
- No histogram quantile implementation.
- No new CI failure defaults.
- No breaking API rename.

## Open Risks

- `recommend-policy` may become too prescriptive if it guesses whether a
  project-specific SLI is informational.
- More examples can drift unless they are kept small and tied to source-of-
  truth docs.
