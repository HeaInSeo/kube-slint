# kind E2E Scenario Matrix

Date: 2026-07-04
Status: Planning matrix for quality roadmap Sprint 3

## Purpose

kube-slint runs against Kubernetes workloads under test. A kind E2E matrix is
needed to verify realistic paths without redefining `test/e2e` as real-cluster
operator deployment E2E.

This document is a planning artifact. It does not change the current
mock-based `test/e2e` contract.

## Scenario Matrix

| ID | Scenario | Expected result |
|---|---|---|
| E2E-1 | Run with namespace-scoped RBAC only | succeeds |
| E2E-2 | Run without ClusterRoleBinding | succeeds |
| E2E-3 | Metrics Service missing | clear `NO_GRADE` or config error |
| E2E-4 | Metrics endpoint returns HTTP 500 | measurement failure, not app test failure |
| E2E-5 | Metrics endpoint returns malformed Prometheus text | parser failure and `NO_GRADE` |
| E2E-6 | External ServiceURLFormat configured | reject before scraping |
| E2E-7 | Cleanup runs after measurement | only kube-slint-owned resources are deleted |
| E2E-8 | Ten parallel sessions | no artifact or RunID collision |
| E2E-9 | First run with absent baseline | `WARN` or `NO_GRADE` per policy |
| E2E-10 | Regression detected | gate result `FAIL`; CI exit code 1 under strict fail-on |
| E2E-11 | Invalid policy | reject before trusted evaluation |
| E2E-12 | Invalid summary | reject before trusted evaluation |

## Required Artifacts

Each scenario should record:

- test namespace;
- effective kube-slint config;
- generated curl pod name;
- RBAC manifest used;
- raw metrics fixture or endpoint behavior;
- `sli-summary.json`;
- `slint-gate-summary.json`;
- exit code;
- cleanup result.

Artifacts must not contain tokens, Authorization headers, kubeconfig secrets,
or raw credential material.

## Scenario Details

### E2E-1 and E2E-2: namespace-scoped RBAC

Setup:

- Apply ServiceAccount, Role, and RoleBinding only.
- Do not create ClusterRoleBinding.

Acceptance:

- measurement succeeds;
- generated or example RBAC remains namespace-scoped;
- docs and CI do not imply cluster-wide RBAC is the default.

### E2E-3 to E2E-5: measurement failure

Setup:

- remove metrics Service;
- return HTTP 500;
- return malformed Prometheus text.

Acceptance:

- correctness test failure and measurement failure remain separate;
- gate output explains why the run cannot be graded;
- invalid/missing measurement does not become `PASS`.

### E2E-6: external ServiceURLFormat

Setup:

- configure a URL such as `https://%s.%s.evil.example/metrics`.

Acceptance:

- execution rejects before creating curl pod;
- no Authorization material is sent to the external host;
- error message explains cluster-local URL requirement.

### E2E-7: cleanup safety

Setup:

- create kube-slint-owned and non-kube-slint pods in the namespace.

Acceptance:

- cleanup removes only resources with kube-slint ownership metadata;
- non-owned resources remain.

### E2E-8: parallel sessions

Setup:

- run ten sessions in the same namespace.

Acceptance:

- unique RunIDs;
- unique artifact names;
- latest alias behavior remains documented;
- no session deletes another session's resources.

### E2E-9 and E2E-10: baseline and regression

Setup:

- first run without baseline;
- second run with baseline and induced regression.

Acceptance:

- first run produces documented non-PASS or warning behavior;
- regression produces `FAIL` when `regression_detected` is in `fail_on`.

### E2E-11 and E2E-12: invalid inputs

Setup:

- use bad fixtures from `docs/test-matrix/bad-fixtures.md`.

Acceptance:

- invalid input is rejected or produces `NO_GRADE`;
- invalid input never produces `PASS`.

## Open Decisions

- Whether the default first-run baseline absence remains `WARN` or can be
  promoted to `NO_GRADE` by a policy mode.
- Which scenario belongs in CI by default versus nightly/manual workflow.
- Whether external URL rejection is tested with a fake DNS host or pure config
  validation.
