# kube-slint Summary Schema

**Single source of truth**: `github.com/HeaInSeo/kube-slint/pkg/slo/summary`

External tools must not define their own summary struct. Import the package or consume `slint-gate` as a CLI — both enforce the same contract.

## Supported version

```
schemaVersion: "slo.v3"
```

`slint-gate` rejects any summary whose `schemaVersion` is empty or differs from `slo.v3` with `MeasurementStatus=unsupported_schema` and `GateResult=NO_GRADE`.

## Go API

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/summary"

// Load and validate a summary file
s, err := summary.LoadFile("artifacts/sli-summary.json")

// Write a summary atomically (temp-file + rename)
err = summary.WriteFile("artifacts/sli-summary.json", s)

// Comprehensive validation (schemaVersion + generatedAt + result IDs)
err = summary.Validate(s)

// Schema version constant
fmt.Println(summary.SchemaVersion) // "slo.v3"
```

## Minimal summary

The smallest valid summary a measurement tool must produce:

```json
{
  "schemaVersion": "slo.v3",
  "generatedAt": "2026-06-01T10:00:00Z",
  "config": {
    "startedAt": "2026-06-01T09:55:00Z",
    "finishedAt": "2026-06-01T09:59:00Z",
    "mode": { "location": "outside", "trigger": "none" }
  },
  "results": [
    {
      "id": "reconcile_total_delta",
      "status": "pass",
      "value": 42
    }
  ]
}
```

## Full summary

All optional fields included:

```json
{
  "schemaVersion": "slo.v3",
  "generatedAt": "2026-06-01T10:00:00Z",
  "config": {
    "runId": "run-abc123",
    "startedAt": "2026-06-01T09:55:00Z",
    "finishedAt": "2026-06-01T09:59:00Z",
    "mode": { "location": "outside", "trigger": "none" },
    "tags": { "env": "kind", "operator": "hello-operator" },
    "format": "v4",
    "evidencePaths": {
      "raw_metrics_start": "artifacts/metrics-start.txt",
      "raw_metrics_end": "artifacts/metrics-end.txt"
    }
  },
  "reliability": {
    "collectionStatus": "Complete",
    "evaluationStatus": "Complete",
    "confidenceScore": 0.9,
    "startSkewMs": 12,
    "endSkewMs": 8,
    "scrapeLatencyMs": 450
  },
  "results": [
    {
      "id": "reconcile_total_delta",
      "title": "Reconcile Success Delta",
      "unit": "count",
      "kind": "delta_counter",
      "value": 42,
      "status": "pass",
      "inputsUsed": ["controller_runtime_reconcile_total{result=\"success\"}"]
    },
    {
      "id": "workqueue_depth_end",
      "title": "Workqueue Depth (end)",
      "unit": "count",
      "kind": "gauge",
      "value": 0,
      "status": "pass",
      "inputsUsed": ["workqueue_depth{name=\"hello\"}"]
    },
    {
      "id": "churn_delta",
      "title": "Churn Delta",
      "unit": "count",
      "kind": "delta_counter",
      "status": "skip",
      "reason": "counter reset: measurement unreliable (no_grade policy)",
      "inputsUsed": ["jumi_jobs_created_total"]
    }
  ],
  "warnings": []
}
```

## SLIResult.status values

| status  | meaning                                        | gate effect        |
|---------|------------------------------------------------|--------------------|
| `pass`  | within bounds                                  | no change          |
| `warn`  | soft violation (e.g. counter reset suspected)  | WARN               |
| `fail`  | hard violation (judge rule fired)              | FAIL               |
| `block` | pipeline/upstream failure                      | FAIL               |
| `skip`  | measurement excluded or input missing          | NO_GRADE (if value is null) |

## CLI contract

`slint-gate` accepts exactly one measurement file (`--summary`) in the format above.
Any file whose `schemaVersion != "slo.v3"` is rejected before evaluation begins.
