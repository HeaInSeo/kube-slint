# Consumer Onboarding Probe (Phase 4-a)

## Purpose
This directory contains a minimal `kubebuilder` layout to validate how easily an external library consumer can import `kube-slint` and evaluate Default SLIs.

## Outcome
- **Go Import**: Successful (`go mod edit -replace` to local parent, then `go get`).
- **Integration Test**: Successful. The `harness.Session` config successfully booted, scraped the dummy operator's `:8080/metrics` endpoint, and finalized without panicking.
- **Evidence for Deletion**: This proves that `presets/` is NOT required for a consumer to evaluate default SLI metrics (e.g. `up` metric via `spec.UnsafePromKey("up")`).

## How to run
```bash
# Requires bin/k8s binaries for setup-envtest
KUBEBUILDER_ASSETS="$PWD/bin/k8s/1.33.0-linux-amd64" go test ./... -v
```
