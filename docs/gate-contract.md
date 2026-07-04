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
- `fail_on`.

Thresholds require:

- `name`;
- `metric`;
- `operator`;
- `value`.

Allowed operators:

```text
< <= > >= == !=
```

Policy invalid cases:

- empty threshold name;
- duplicate threshold name;
- missing metric;
- unknown operator;
- non-finite value;
- negative regression tolerance;
- unknown `fail_on` value.

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
| threshold miss and `threshold_miss` in `fail_on` | `FAIL` |
| threshold miss and `threshold_miss` not in `fail_on` | `WARN`, never `PASS` |
| regression detected and `regression_detected` in `fail_on` | `FAIL` |
| regression detected and `regression_detected` not in `fail_on` | `WARN`, never `PASS` |
| regression enabled, first run, no baseline | `WARN` with `BASELINE_ABSENT_FIRST_RUN` |
| corrupt or unreadable baseline | `NO_GRADE` |
| measurement collection failed | `NO_GRADE` |
| invalid policy | `NO_GRADE` or execution reject |
| invalid summary | `NO_GRADE` or execution reject |

## fail-on Modes

| Mode | Fails CI on |
|---|---|
| `NEVER` | never; caller inspects output |
| `FAIL` | `FAIL` |
| `FAIL_OR_WARN` | `FAIL`, `WARN` |
| `FAIL_OR_NOGRADE` | `FAIL`, `NO_GRADE` |
| `FAIL_WARN_OR_NOGRADE` | `FAIL`, `WARN`, `NO_GRADE` |

Unknown `fail-on` values are invalid.

## Open Decisions

- Whether NaN/Inf metric values are invalid input or measurement failure.
- Which unknown summary/policy fields may be ignored for compatibility.
- Whether required baseline absence should produce `NO_GRADE` or `FAIL`.
