# Planned Bad Fixtures

Date: 2026-07-04

This directory is a planning area for bad fixtures that are not yet committed
as executable testdata. It exists so the quality/docs workstream can define
negative inputs without claiming runtime behavior has already changed.

Executable fixtures should move to the appropriate package `testdata`
directory only after the implementation agent adds or confirms the validator.

## Planned Layout

```text
testdata-plan/bad-fixtures/
  summary/
    missing-schema-version.json
    wrong-schema-version.json
    empty-result-id.json
    duplicate-result-id.json
    unknown-result-status.json
    invalid-generated-at.json
    nan-metric-value.json
    inf-metric-value.json
    malformed-json.json
  policy/
    missing-policy-version.yaml
    wrong-policy-version.yaml
    unknown-operator.yaml
    duplicate-threshold-name.yaml
    missing-metric.yaml
    negative-tolerance.yaml
    nan-threshold-value.yaml
    empty-threshold-name.yaml
    unknown-fail-on.yaml
    baseline-required-but-missing.yaml
  security/
    external-service-url.yaml
    external-service-url-template-injection.yaml
    ftp-service-url.yaml
    insecure-tls-default.yaml
    clusterrolebinding-default.yaml
    privileged-curlpod.yaml
    hostpath-curlpod.yaml
    kube-system-target.yaml
    cleanup-without-owner-label.yaml
```

## Expected Result Source

The expected result for each fixture is defined in:

- `docs/test-matrix/bad-fixtures.md`

## Rules

- Do not add real tokens or kubeconfigs.
- Do not use production cluster names.
- Do not move a fixture into executable tests until the validator and expected
  behavior are implemented or accepted.
- If the expected behavior is still an open decision, mark it in the fixture
  test table instead of guessing.
