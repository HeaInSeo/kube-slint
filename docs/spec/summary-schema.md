# Summary Schema Contract

Date: 2026-07-04
Status: Contract draft aligned with current `slo.v3` implementation

## Purpose

The measurement summary is the machine-readable artifact that connects
kube-slint measurement to `slint-gate` policy evaluation. Invalid summaries
must not silently produce `PASS`.

## Current Schema Version

```text
slo.v3
```

The JSON field is:

```json
{
  "schemaVersion": "slo.v3"
}
```

## Minimum Valid Summary

```json
{
  "schemaVersion": "slo.v3",
  "generatedAt": "2026-07-04T00:00:00Z",
  "config": {
    "startedAt": "2026-07-04T00:00:00Z",
    "finishedAt": "2026-07-04T00:01:00Z",
    "mode": { "location": "outside", "trigger": "none" }
  },
  "results": [
    {
      "id": "reconcile_total_delta",
      "status": "pass",
      "value": 1
    }
  ]
}
```

## Required Validation

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
| unknown top-level field | open decision: reject or metadata-only ignore |

## Result Status Values

Allowed values:

| Status | Meaning | Gate effect |
|---|---|---|
| `pass` | measured and within local SLI rule | no negative effect |
| `warn` | measured with warning or soft violation | may contribute WARN |
| `fail` | local SLI rule failed | may contribute FAIL |
| `block` | upstream/pipeline failure | may contribute FAIL |
| `skip` | measurement skipped or insufficient | contributes NO_GRADE when no value exists |

Unknown statuses must be rejected. They must not be interpreted as `skip`,
`warn`, or `pass`.

## Reliability Fields

Reliability metadata should describe whether measurement collection was
trustworthy enough to grade.

Important current fields:

- `collectionStatus`
- `evaluationStatus`
- `confidenceScore`
- scrape timing fields

`CollectionStatus=Failed` must not silently produce `PASS`. Per post-RC
hardening, failed collection is an unconditional `NO_GRADE` at gate time.

## Unknown Field Policy

Open decision:

- summary readers may ignore unknown metadata fields for append-compatible
  evolution;
- summary readers must not ignore unknown semantic fields that change result
  status, metric identity, or gate meaning.

Recommended rule:

- unknown fields under `config.tags` and future `metadata` are append-only and
  may be ignored;
- unknown result status, unknown schema version, and duplicate IDs are always
  invalid.

## Bad Fixtures

Required bad fixtures are listed in `docs/test-matrix/bad-fixtures.md`.

Minimum summary fixture set:

- `missing-schema-version.json`
- `wrong-schema-version.json`
- `empty-result-id.json`
- `duplicate-result-id.json`
- `unknown-result-status.json`
- `invalid-generated-at.json`
- `nan-metric-value.json`
- `malformed-json.json`

## Acceptance Criteria

- [ ] Invalid schema version cannot produce PASS.
- [ ] Duplicate result IDs cannot produce last-write-wins behavior.
- [ ] Unknown statuses cannot be ignored.
- [ ] NaN/Inf behavior is documented and tested.
- [ ] Summary validation failures surface as invalid input or `NO_GRADE`, not
  as successful gate evaluation.

## Related Documents

- `docs/integration/summary-schema.md`
- `docs/spec/policy-schema.md`
- `docs/spec/gate-result-semantics.md`
- `docs/test-matrix/bad-fixtures.md`
