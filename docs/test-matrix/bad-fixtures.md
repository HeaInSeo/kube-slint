# Bad Fixture Matrix

Date: 2026-07-04
Status: Planning matrix for quality roadmap Sprint 2

## Purpose

Bad fixtures define inputs that must fail validation or produce non-PASS gate
results. They prevent malformed summary, malformed policy, and dangerous
security configuration from being silently accepted.

## Summary Fixtures

| File | Input shape | Expected result |
|---|---|---|
| `summary/missing-schema-version.json` | no `schemaVersion` | reject |
| `summary/wrong-schema-version.json` | `schemaVersion: "slo.v999"` | reject |
| `summary/empty-result-id.json` | result ID is empty string | reject |
| `summary/duplicate-result-id.json` | same result ID appears twice | reject |
| `summary/unknown-result-status.json` | result status is not allowed | reject |
| `summary/invalid-generated-at.json` | invalid or missing timestamp | reject |
| `summary/nan-metric-value.json` | metric value is NaN | reject recommended; open policy |
| `summary/inf-metric-value.json` | metric value is Infinity | reject recommended; open policy |
| `summary/malformed-json.json` | invalid JSON | reject |

## Policy Fixtures

| File | Input shape | Expected result |
|---|---|---|
| `policy/missing-policy-version.yaml` | no `schema_version` | reject |
| `policy/wrong-policy-version.yaml` | unsupported `schema_version` | reject |
| `policy/unknown-operator.yaml` | threshold operator not in allowed set | reject |
| `policy/duplicate-threshold-name.yaml` | duplicate threshold name | reject |
| `policy/missing-metric.yaml` | threshold has no metric | reject |
| `policy/negative-tolerance.yaml` | `tolerance_percent < 0` | reject |
| `policy/nan-threshold-value.yaml` | threshold value is NaN | reject recommended |
| `policy/empty-threshold-name.yaml` | threshold name empty | reject |
| `policy/unknown-fail-on.yaml` | unsupported `fail_on` value | reject |
| `policy/baseline-required-but-missing.yaml` | future required baseline mode with no baseline | `NO_GRADE` or `FAIL`; open policy |

## Security Fixtures

| File | Input shape | Expected result |
|---|---|---|
| `security/external-service-url.yaml` | ServiceURLFormat points to public host | reject |
| `security/external-service-url-template-injection.yaml` | `%s.%s.evil.com` host | reject |
| `security/ftp-service-url.yaml` | unsupported URL scheme | reject |
| `security/insecure-tls-default.yaml` | TLS skip verify without dangerous opt-in | reject or compatibility warning until migrated |
| `security/clusterrolebinding-default.yaml` | default generated RBAC contains ClusterRoleBinding | reject |
| `security/privileged-curlpod.yaml` | curl pod privileged securityContext | reject |
| `security/hostpath-curlpod.yaml` | curl pod mounts hostPath | reject |
| `security/kube-system-target.yaml` | default run targets `kube-system` | reject |
| `security/cleanup-without-owner-label.yaml` | cleanup selector lacks kube-slint owner labels | reject |

## Gate Fixture Scenarios

| Scenario | Expected result |
|---|---|
| threshold miss with `threshold_miss` in `fail_on` | `FAIL` |
| threshold miss without `threshold_miss` in `fail_on` | `WARN`, never `PASS` |
| regression detected with `regression_detected` in `fail_on` | `FAIL` |
| regression detected without `regression_detected` in `fail_on` | `WARN`, never `PASS` |
| collection status failed | `NO_GRADE` |
| corrupt baseline | `NO_GRADE` |
| first-run baseline absent and regression enabled | `WARN` |
| invalid policy | reject or `NO_GRADE`, never `PASS` |
| invalid summary | reject or `NO_GRADE`, never `PASS` |

## Fixture Implementation Rules

- Each fixture must include expected result metadata in the test table.
- Fixture filenames should describe the rejected behavior.
- Fixtures must not contain real tokens, kubeconfigs, or CI secrets.
- Security fixtures should include both default rejection and explicit opt-in
  cases where a dangerous option exists.
- If behavior is not implemented yet, keep the fixture under
  `testdata-plan/bad-fixtures` until implementation lands.

## Developer Handoff

Use the ticket backlog in `docs/quality-roadmap-ticket-backlog.md`.

Minimum first implementation batch:

1. summary schema fixtures;
2. policy schema fixtures;
3. gate result priority fixtures;
4. ServiceURLFormat security fixtures.
