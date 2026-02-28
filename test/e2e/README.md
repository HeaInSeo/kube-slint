# kube-slint Integration Tests

This directory contains integration and end-to-end (E2E) testing assets for `kube-slint`. 

As `kube-slint` evolved from a standalone controller to an embeddable Observability Library, the testing strategy has also been modernized. The core harness engine is now verified using **Mock-based, in-memory Integration Tests** rather than heavyweight Kubernetes deployments.

## Testing Strategy: Mock Integration

The primary testing path is `harness_integration_test.go`. It bypasses the need for a live Kubernetes cluster or `curl` binaries by injecting an `httptest.Server` directly into the `harness.SessionConfig.Fetcher`.

### Advantages
1. **Flakiness Zero**: All network delays, Pod restart timeouts, and cluster provisioning errors are eliminated. Tests execute in ~0.01 seconds.
2. **Deterministic Edge Cases**: We forcefully inject network errors (HTTP 500) and missing metric payloads to verify the Engine's Safe-Fail / Block behaviors deterministically.
3. **No K8s Dependencies**: The core business logic (Compute, Judge, Summary generation) is rigorously tested in pure Go memory.

*Note: This approach aligns with our philosophy that "Monitoring failure != Test failure". The E2E tests assert that the SLI Engine accurately captures and reports the failure states rather than asserting the system under test is healthy.*

### Running the Integration Tests

To execute the modern table-driven test suite:

```bash
go test ./test/e2e/ -run TestHarnessIntegration_TableDriven -v
```

### Representative Scenarios Tested
The integration suite guarantees coverage across 5 core dimensions:
* **Happy Path**: Verifies that standard single-metric inputs trigger a `Pass` evaluation.
* **Missing Metric**: Asserts that omitted metrics skip the rules gracefully without panicking the engine.
* **Network Fetch Error**: Simulates `http.StatusInternalServerError` to ensure the scrape failure blocks the SLI evaluation and degrades the reliability score properly.
* **Delta Path**: Injects varying start and end snapshot values sequentially to prove `ComputeDelta` derives the differences correctly.
* **Multi-metric Mixed Result**: Mixes Pass and Fail targets in a single Session to confirm robust struct marshalling and individual result separation.
