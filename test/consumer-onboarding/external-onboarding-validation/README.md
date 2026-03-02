# External Consumer Onboarding Validation (Phase 5-a)

## Purpose
This directory acts as a **true external consumer probe**. It mimics a brand new Kubebuilder Operator trying to integrate `kube-slint` via the Go API and deploy its observability stack via Kustomize Remote Imports.

It intentionally has its own `go.mod` (using a local `replace` directive) to simulate external module resolution.

## Key Goals
1. **API Onboarding UX**: Prove that dropping `kube-slint`'s Go library (`harness`, `spec`) into an external `main.go` compiles and runs cleanly.
2. **Kustomize Remote UX**: Validate the Explicit Local Override patch strategy (P4-1/P4-2) by importing the RC tag and patching the target labels.

## How to Run

### 1. Kustomize Render Verification
```bash
# Renders the Prometheus objects for this mock operator
kubectl kustomize kustomize/
```
**Success Criteria:** You should see `app.kubernetes.io/name: my-operator` injected into the `matchLabels` and `metadata.labels` of the `ServiceMonitor`.

### 2. Go API Compilation
```bash
# Resolves dependencies and builds the mock
go mod tidy
go build -o /dev/null main.go
```
**Success Criteria:** Zero compilation errors.

## Friction Points Observed (Consumer POV)
- **API (Session Initialization):** Consumers must pass a completely populated `SessionConfig`. It is not instantly obvious how to correctly wire `MetricsServiceName` without reading the deeper docs.
- **API (Specs Formulation):** Using `spec.UnsafePromKey()` vs a native string might feel slightly non-idiomatic to new users, but it forces safety.
- **Go Mod:** As expected, consuming it as a library works flawlessly.
