# Gate Result Semantics

Date: 2026-07-04
Status: Contract draft aligned with post-RC hardening

## Purpose

`slint-gate` turns measurement summaries, policies, and optional baselines into
CI-facing results. This document defines the meaning and priority of those
results.

## Result Values

| Result | Meaning | Typical CI handling |
|---|---|---|
| `PASS` | Measurement was sufficient and policy checks passed. | Continue. |
| `WARN` | A non-blocking concern exists, or first-run baseline is absent when regression is optional. | Continue unless caller uses strict fail mode. |
| `FAIL` | Policy violation detected and configured as hard failure. | Fail CI. |
| `NO_GRADE` | kube-slint cannot make a trustworthy policy decision. | Usually fail promotion CI with `FAIL_OR_NOGRADE`. |

## Priority

Recommended aggregate priority:

```text
FAIL > NO_GRADE > WARN > FIRST_RUN_WARNING > PASS
```

Current implementation uses `BASELINE_ABSENT_FIRST_RUN` as a warning reason.
The separate `FIRST_RUN_WARNING` label is a planning shorthand, not a current
wire value.

## Core Principle

Measurement failure is not correctness test failure.

That means:

- the user's E2E test may continue;
- kube-slint may still emit a summary;
- `slint-gate` may return `NO_GRADE`;
- CI can fail the gate step when configured with `FAIL_OR_NOGRADE`.

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
| unsupported schema | `NO_GRADE` or execution reject |

## fail-on Modes

Current action/CLI modes:

| Mode | Fails CI on |
|---|---|
| `NEVER` | never; caller inspects output |
| `FAIL` | `FAIL` |
| `FAIL_OR_WARN` | `FAIL`, `WARN` |
| `FAIL_OR_NOGRADE` | `FAIL`, `NO_GRADE` |
| `FAIL_WARN_OR_NOGRADE` | `FAIL`, `WARN`, `NO_GRADE` |

Unknown `fail-on` values are invalid.

## Invalid Input Rule

Invalid policy or summary input must not produce `PASS`.

Allowed handling:

- reject before evaluation with non-zero exit;
- write a gate summary with invalid status and `NO_GRADE`;
- both, when the CLI contract explicitly documents it.

Disallowed handling:

- ignore invalid fields and pass;
- treat unknown enum values as default safe values;
- allow duplicate IDs or duplicate threshold names to overwrite each other.

## First-Run Baseline Rule

Current first-run behavior:

- regression enabled but no baseline: `WARN`;
- regression disabled: no baseline warning;
- corrupt baseline: `NO_GRADE`.

Open decision:

- add a policy mode where baseline absence is required and produces
  `NO_GRADE` or `FAIL`.

## Acceptance Criteria

- [ ] `FAIL` outranks all other results.
- [ ] `NO_GRADE` outranks `WARN`.
- [ ] First-run warning is documented separately from corrupt/missing required
  baseline.
- [ ] Unknown `fail-on` values are rejected.
- [ ] Invalid policy/summary cannot result in `PASS`.
- [ ] CI examples use `FAIL_OR_NOGRADE` for promotion gates.
