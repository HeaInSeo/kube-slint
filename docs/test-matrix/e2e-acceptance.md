# E2E Acceptance Criteria

Date: 2026-07-04
Status: Planning criteria for quality roadmap Sprint 3

## Purpose

This document defines how kind E2E scenarios should be accepted once the
implementation agent turns the matrix into executable tests.

## Acceptance Principles

- E2E scenarios must not collapse measurement failure into app correctness
  failure.
- Dangerous configuration must reject before creating cluster resources.
- Gate result must be machine-readable and human-explainable.
- Artifacts must be unique enough for parallel runs.
- Cleanup must be ownership-scoped.
- Secret material must not appear in logs or artifacts.

## Required Per-Scenario Checks

Every kind E2E scenario should assert:

- command exit code;
- `gate_result`;
- `evaluation_status`;
- `measurement_status`;
- relevant reason codes;
- expected artifact presence or absence;
- cleanup result;
- absence of raw secrets in logs/artifacts.

## CI Placement

Recommended phases:

| Phase | Contents | Trigger |
|---|---|---|
| PR smoke | one happy path, one invalid policy, one external URL reject | PR touching gate/harness/security |
| Nightly | full E2E matrix | scheduled |
| Manual | expensive parallel and regression scenarios | workflow_dispatch |

## Promotion Gate Recommendation

For promotion CI:

```yaml
fail-on: FAIL_OR_NOGRADE
```

Rationale:

- `FAIL` blocks known policy violations.
- `NO_GRADE` blocks missing or untrustworthy measurement.
- `WARN` can remain non-blocking for first-run adoption unless stricter policy
  is selected.

## Non-Acceptance Cases

Do not accept an E2E scenario if:

- invalid input produces `PASS`;
- external ServiceURLFormat creates a curl pod;
- cleanup deletes non-owned resources;
- logs contain raw token or Authorization material;
- missing metrics service is reported as app test assertion failure without a
  kube-slint measurement status;
- ClusterRoleBinding is required for default measurement.

## Handoff

Executable E2E work should be tracked as implementation tickets. Each ticket
must name:

- scenario ID;
- setup;
- expected gate result;
- expected exit code;
- artifacts to inspect;
- docs to update;
- security impact.
