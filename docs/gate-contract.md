# Gate Contract

Date: 2026-07-04
Status: Canonical summary/policy/gate contract

## Purpose

This document consolidates the summary schema, policy schema, and gate result
semantics. Invalid summary or policy input must not silently produce `PASS`.

## Summary Schema

Current summary schema version:

```text
slo.v3
```

Required JSON field:

```json
{ "schemaVersion": "slo.v3" }
```

Required validation:

| Field or condition | Required behavior |
|---|---|
| missing `schemaVersion` | reject |
| unsupported `schemaVersion` | reject |
| missing `generatedAt` | reject |
| invalid `generatedAt` | reject |
| empty result ID | reject |
| duplicate result ID | reject |
| unknown result status | reject |
| malformed JSON | reject |
| metric value NaN/Inf | open decision: reject recommended |

Invalid schema version cannot produce PASS.

Allowed result statuses:

- `pass`;
- `warn`;
- `fail`;
- `block`;
- `skip`.

Unknown statuses must be rejected.

## Policy Schema

Current policy schema version:

```yaml
schema_version: "slint.policy.v1"
```

Missing or unsupported `schema_version` is rejected.

Top-level fields:

- `schema_version`;
- `thresholds`;
- `regression`;
- `reliability`;
- `coverage`;
- `promote_to_fail` (preferred) / `fail_on` (deprecated alias — both are
  honored as a union; see `docs/sli-gate-onboarding-ux.md`'s naming section).

Thresholds require:

- `name`;
- `metric`;
- `operator`;
- `value`.

Allowed operators:

```text
< <= > >= ==
```

`!=` is not supported — it is correctly treated as an unknown operator (see
below), which already satisfies "invalid policy never produces PASS."

Policy invalid cases:

- duplicate threshold name (empty names are allowed and auto-assigned
  `unnamed-threshold`; see the Priority 0 implementation notes);
- missing metric;
- unknown operator;
- non-finite value;
- negative regression tolerance;
- unknown `promote_to_fail`/`fail_on` value.

Coverage governance is strict by default in generated policies:

```yaml
coverage:
  required: true
  informational:
    - reconcile_success_delta
```

When enabled, measured summary results with a scalar value must either be
covered by a threshold rule or listed under `coverage.informational`.
Uncovered measured SLIs produce `coverage` checks. Generated policies list
`coverage_gap` in `promote_to_fail`, and omitted/empty `promote_to_fail` uses
the same strict default. Set `coverage.required: false` to disable coverage-gap
checks for a policy.

## Inspect vs Gate

Inspect readiness is not a gate verdict.

`slint-gate inspect --summary ... --policy ...` is an advisory diagnostic
command. It helps a human see which SLIs were measured, which expected profile
SLIs are missing, and which measured SLIs are not covered by policy. It may
print coverage-gap next actions, but it must not produce `gate_result`, must
not decide `PASS`/`WARN`/`FAIL`/`NO_GRADE`, and must not be used as the CI
enforcement step.

CI enforcement must call the gate evaluator path: the default `slint-gate`
command or the `.github/actions/slint-gate` composite action. Coverage gaps
become actual gate checks only in that evaluation path. For example,
`coverage.required: true` plus `coverage_gap` in `promote_to_fail` can produce
`gate_result=FAIL`; `inspect --policy` with the same files should still exit
successfully and explain what to fix.

## Gate Result Semantics

Result values:

| Result | Meaning |
|---|---|
| `PASS` | Measurement was sufficient and policy checks passed. |
| `WARN` | A non-blocking concern exists, or first-run baseline is absent when regression is optional. |
| `FAIL` | Policy violation detected and configured as hard failure. |
| `NO_GRADE` | kube-slint cannot make a trustworthy policy decision. |

Recommended priority:

```text
FAIL > NO_GRADE > WARN > FIRST_RUN_WARNING > PASS
```

Invalid policy or summary input must not produce `PASS`.

## Scenario Semantics

| Scenario | Expected gate result |
|---|---|
| valid summary, valid policy, thresholds pass | `PASS` |
| threshold miss and `threshold_miss` in `promote_to_fail`/`fail_on` | `FAIL` |
| threshold miss and `threshold_miss` not in `promote_to_fail`/`fail_on` | `WARN`, never `PASS` |
| regression detected and `regression_detected` in `promote_to_fail`/`fail_on` | `FAIL` |
| regression detected and `regression_detected` not in `promote_to_fail`/`fail_on` | `WARN`, never `PASS` |
| regression enabled, first run, no baseline | `WARN` with `BASELINE_ABSENT_FIRST_RUN` |
| corrupt or unreadable baseline | `NO_GRADE` |
| measurement collection failed | `NO_GRADE` |
| invalid policy | `NO_GRADE` or execution reject |
| invalid summary | `NO_GRADE` or execution reject |

## exit-on Modes

CLI flag `--exit-on` (preferred) / `--fail-on` (deprecated alias); GitHub
Action input `exit-on` (preferred) / `fail-on` (deprecated alias). `--exit-on`
wins if both are passed; using only `--fail-on`/`fail-on` still works but
emits a deprecation notice.

| Mode | Fails CI on |
|---|---|
| `NEVER` | never; caller inspects output |
| `FAIL` | `FAIL` |
| `FAIL_OR_WARN` | `FAIL`, `WARN` |
| `FAIL_OR_NOGRADE` | `FAIL`, `NO_GRADE` |
| `FAIL_WARN_OR_NOGRADE` | `FAIL`, `WARN`, `NO_GRADE` |

Unknown `--exit-on`/`exit-on` values are invalid.

## KSL-T Trust-Correct Contracts

KSL-T brings protected qualification behavior into conformance with the canonical
trust rules `KSL-R1~R4`. It introduces explicit trust-correct contracts and fences
the legacy ones; **it does not activate protected qualification** — real
policy-neutral measurement wiring (KSL-E) and a separate central activation
decision remain outstanding.

Trust-correct contract versions:

| Contract | Legacy | Trust-correct |
|---|---|---|
| policy | `slint.policy.v1` | `slint.policy.v2` |
| measurement | `slo.v3` | `slo.v4` (adds per-SLI `comparability`) |

Both versions are accepted, but a legacy contract is never silently treated as
trust-correct.

- **T1 — whole-policy strict validity.** A policy is validated in full before any
  protected grade: unknown keys at **every** level are rejected (not warned),
  exactly one YAML document is allowed, duplicate mapping keys are rejected,
  operators/enums/numeric domains must be supported, required coordinates
  (`metric`, `operator`) must be present, and identities (`name`) must be unique.
  An invalid-but-readable policy yields `NO_GRADE`; an ordinary check `FAIL` never
  masks policy invalidity.
- **T2 — producer verdict is non-authoritative.** A measurement producer's
  `pass/warn/fail/block/skip` status is recorded as a diagnostic
  (`measurement_diagnostic`) only; it never creates a protected
  `PASS/WARN/FAIL/NO_GRADE`. Gate Policy is the sole qualification authority.
- **T3 — positive per-check evidence sufficiency.** A check grades only when the
  referenced SLI's evidence is positively sufficient, judged from typed facts
  (value present, not in the reliability skipped-set, no missing inputs) — never
  from the producer verdict. Missing/unreliable evidence for a required check is
  `NO_GRADE` for that check; an unrelated insufficient SLI does not poison an
  independently sufficient one.
- **T4 — provable baseline comparability.** Regression runs only when current and
  baseline carry a complete, matching `comparability` identity across all
  coordinates (`sliContractId`, `subjectId`, `windowId`, `sourceConfigId`).
  Window semantics come from the explicit `windowId`, never inferred from run
  start/finish. Absent/mismatched comparability is `NO_GRADE`
  (`BASELINE_INCOMPARABLE`), never a silent comparison.
- **T5 — explicit measurement-contract fence.** Comparability lives behind the
  trust-correct measurement contract (`slo.v4`). A legacy `slo.v3` measurement, or
  a legacy `slint.policy.v1` policy, cannot satisfy protected comparability by
  silently ignoring the trust-required fields — such regressions are `NO_GRADE`.
  Historical artifacts are never silently reinterpreted under the new semantics.

## Open Decisions

- ~~Whether NaN/Inf metric values are invalid input or measurement failure.~~
  Resolved (Priority 0 implementation): summary-side NaN/Inf is invalid JSON
  and is rejected as `measCorrupt` before any policy logic runs; policy-side
  NaN threshold values are explicitly rejected by `validatePolicy`.
- ~~Which unknown summary/policy fields may be ignored for compatibility.~~
  Resolved (KSL-T1): unknown policy keys at any level are rejected as invalid;
  no unknown policy field is silently ignored.
- Whether required baseline absence should produce `NO_GRADE` or `FAIL`.
