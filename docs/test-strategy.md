# Test Strategy

Date: 2026-07-04
Status: Canonical test matrix planning source

## Purpose

This document consolidates bad fixture, kind E2E, and E2E acceptance planning.
It defines what must fail before implementation converts the cases into
executable tests.

## Bad Fixture Matrix

Status: implemented as executable tests in `pkg/gate/testdata/{summary,policy}/`
and `pkg/gate/badfixtures_test.go` (`TestBadFixtures_Summary`,
`TestBadFixtures_Policy`), plus `pkg/slo/fetch/curlpod/urlvalidate_test.go`
and `client_test.go` for the security fixtures — see the notes below for two
rows whose "expected result" changed from originally planned during
implementation.

Summary fixtures:

| File | Expected result |
|---|---|
| `summary/missing-schema-version.json` | reject — implemented |
| `summary/wrong-schema-version.json` | reject — implemented |
| `summary/empty-result-id.json` | reject — implemented |
| `summary/duplicate-result-id.json` | reject — implemented |
| `summary/unknown-result-status.json` | reject — implemented |
| `summary/invalid-generated-at.json` | reject — implemented |
| `summary/nan-metric-value.json` | reject — implemented (bare NaN is invalid JSON, rejected before any policy logic) |
| `summary/inf-metric-value.json` | reject — implemented (same as above) |
| `summary/malformed-json.json` | reject — implemented |

Policy fixtures:

| File | Expected result |
|---|---|
| `policy/missing-policy-version.yaml` | reject — implemented |
| `policy/wrong-policy-version.yaml` | reject — implemented |
| `policy/unknown-operator.yaml` | reject — implemented (uses `!=`; `!=` is not a supported operator, see gate-contract.md) |
| `policy/duplicate-threshold-name.yaml` | reject — implemented |
| `policy/missing-metric.yaml` | reject — implemented (per-check `NO_GRADE`, not a whole-policy reject) |
| `policy/negative-tolerance.yaml` | reject — implemented |
| `policy/nan-threshold-value.yaml` | reject — implemented |
| `policy/empty-threshold-name.yaml` | **allowed, not rejected** — resolved conflict: `evalThreshold` intentionally auto-names an empty threshold `"unnamed-threshold"` and proceeds; this is existing, tested (`TestEvaluate_UnnamedThreshold`), accepted behavior. Not implemented as a bad fixture. |
| `policy/unknown-fail-on.yaml` | reject — implemented |
| `policy/baseline-required-but-missing.yaml` | `NO_GRADE` or `FAIL`; still open policy, not implemented this pass |

Security fixtures:

| File | Expected result |
|---|---|
| `security/external-service-url.yaml` | reject — implemented (`ValidateMetricsURL`) |
| `security/external-service-url-template-injection.yaml` | reject — implemented (DNS-label validation on service/namespace) |
| `security/ftp-service-url.yaml` | reject — implemented |
| `security/insecure-tls-default.yaml` | reject by default — implemented (`curlpod.New()`'s `TLSInsecureSkipVerify` default changed to `false`; `DangerouslySkipTLSVerify` is the explicit opt-in) |
| `security/clusterrolebinding-default.yaml` | reject — already implemented and tested (`TestRunInit_EmitRBAC`) |
| `security/privileged-curlpod.yaml` | reject — already implemented and tested |
| `security/hostpath-curlpod.yaml` | reject — already implemented and tested |
| `security/kube-system-target.yaml` | reject — implemented (`isDangerousNamespace`; `DangerouslyAllowKubeSystemNamespace` is the explicit opt-in) |
| `security/cleanup-without-owner-label.yaml` | reject — already safe by construction (delete targets are derived exclusively from the label-filtered list step in the same call; see the code comment on `applySweepDeletes`) |

Fixture rules:

- no real tokens or kubeconfigs;
- filenames describe rejected behavior;
- expected result lives in the test table;
- planned fixtures stay out of executable testdata until validators exist.

## kind E2E Scenario Matrix

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
| E2E-10 | Regression detected | gate result `FAIL`; strict CI exit code 1 |
| E2E-11 | Invalid policy | reject before trusted evaluation |
| E2E-12 | Invalid summary | reject before trusted evaluation |

## E2E Acceptance Criteria

Every kind E2E scenario should assert:

- command exit code;
- `gate_result`;
- `evaluation_status`;
- `measurement_status`;
- relevant reason codes;
- expected artifact presence or absence;
- cleanup result;
- absence of raw secrets in logs/artifacts.

Do not accept an E2E scenario if:

- invalid input produces `PASS`;
- external ServiceURLFormat creates a curl pod;
- cleanup deletes non-owned resources;
- logs contain raw token or Authorization material;
- missing metrics service is reported as app test assertion failure without a
  kube-slint measurement status;
- ClusterRoleBinding is required for default measurement.

## CI Placement

Recommended phases:

| Phase | Contents | Trigger |
|---|---|---|
| PR smoke | happy path, invalid policy, external URL reject | PR touching gate/harness/security |
| Nightly | full E2E matrix | scheduled |
| Manual | expensive parallel and regression scenarios | workflow_dispatch |

Promotion gates should use:

```yaml
fail-on: FAIL_OR_NOGRADE
```
