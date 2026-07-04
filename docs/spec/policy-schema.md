# Policy Schema Contract

Date: 2026-07-04
Status: Contract draft aligned with current `slint.policy.v1` implementation

## Purpose

The policy file tells `slint-gate` which measurement results should block CI.
Invalid policy input must be rejected or downgraded to a non-PASS gate result.

## Current Schema Version

```yaml
schema_version: "slint.policy.v1"
```

Missing or unsupported `schema_version` is invalid.

## Minimal Policy

```yaml
schema_version: "slint.policy.v1"

thresholds:
  - name: "reconcile_total_delta_min"
    metric: "reconcile_total_delta"
    operator: ">="
    value: 1

regression:
  enabled: false
  tolerance_percent: 10

reliability:
  required: false
  min_level: "partial"

fail_on:
  - "threshold_miss"
  - "regression_detected"
```

## Top-Level Fields

| Field | Required | Meaning |
|---|---:|---|
| `schema_version` | yes | Policy compatibility version. |
| `thresholds` | no, but recommended | Absolute gate checks. |
| `regression` | no | Baseline comparison settings. |
| `reliability` | no | Measurement reliability requirements. |
| `fail_on` | no | Which violation categories become `FAIL`. |

Current implementation warns about unknown top-level fields. Future policy
compatibility should decide which unknown fields are allowed.

## Threshold Contract

Each threshold must have:

- `name`
- `metric`
- `operator`
- `value`

Required behavior:

| Condition | Required behavior |
|---|---|
| empty threshold name | reject |
| duplicate threshold name | reject |
| missing metric | reject |
| unknown operator | reject |
| non-finite value | reject recommended |
| value type not numeric | reject |

Allowed operators:

```text
< <= > >= == !=
```

## Regression Contract

Regression checks compare current summary values against a baseline summary.

Fields:

```yaml
regression:
  enabled: true
  tolerance_percent: 10
```

Required behavior:

- `tolerance_percent` must be finite and non-negative.
- If regression is disabled, missing baseline does not produce WARN or
  NO_GRADE.
- If regression is enabled and baseline is absent on first run, current
  behavior is `WARN` with reason `BASELINE_ABSENT_FIRST_RUN`.
- If regression is required by future policy, missing baseline should become
  `NO_GRADE` or `FAIL` according to the documented policy.
- Corrupt or unreadable baseline should be `NO_GRADE`.

## Reliability Contract

Reliability settings determine how measurement completeness affects gate
evaluation.

Current principle:

- measurement failure is not correctness test failure;
- gate can still report `NO_GRADE`;
- CI can fail on `NO_GRADE` with `FAIL_OR_NOGRADE`.

`CollectionStatus=Failed` must not resolve to `PASS`.

## fail_on Contract

Allowed values:

```text
threshold_miss
regression_detected
```

Unknown values must be rejected. They must not silently downgrade or disappear.

If `fail_on` is omitted or empty, kube-slint currently uses default hard-fail
categories for threshold miss and regression detected.

## Unknown Field Policy

Current implementation warns for unknown top-level policy fields. Long-term
compatibility policy should distinguish:

- harmless metadata extensions;
- semantic policy fields that must be rejected if unsupported.

Recommended rule:

Unknown semantic policy fields should become invalid input once a formal
extension mechanism exists.

## Bad Fixtures

Required bad fixtures are listed in `docs/test-matrix/bad-fixtures.md`.

Minimum policy fixture set:

- `missing-policy-version.yaml`
- `unknown-operator.yaml`
- `duplicate-threshold-name.yaml`
- `missing-metric.yaml`
- `negative-tolerance.yaml`
- `nan-threshold-value.yaml`
- `empty-threshold-name.yaml`
- `baseline-required-but-missing.yaml`

## Acceptance Criteria

- [ ] Missing or unsupported `schema_version` is rejected.
- [ ] Unknown operators are rejected.
- [ ] Duplicate threshold names are rejected.
- [ ] Non-finite numeric values are not silently accepted.
- [ ] Unknown `fail_on` values are rejected.
- [ ] First-run baseline behavior is visible and documented.
- [ ] Invalid policy cannot produce PASS.

## Related Documents

- `.slint/policy.example.yaml`
- `docs/spec/summary-schema.md`
- `docs/spec/gate-result-semantics.md`
- `docs/test-matrix/bad-fixtures.md`
