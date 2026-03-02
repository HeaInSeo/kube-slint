# External Onboarding Validation Report (Phase 5-a)
*Date: 2026-03-02*

## 1. Context & Goal
The objective of Phase 5-a is to execute a minimal consumer onboarding proof-of-concept using a clean, mock Go module. We aimed to validate the explicit Kustomize Local Override strategy and the `kube-slint` Go API consumption experience for a fresh Kubebuilder operator.

## 2. Validation Constraints (Policy-First)
- **Zero modification to `pkg/...`**: Prove the API works *as is*.
- **No Kustomize Structural Rewrites**: Utilize the MVP documentation model (P4) to circumvent silent drop-in failures.
- **Evidence-based reporting**: Accurately log the friction points observed during implementation.

## 3. Results Overview
- **Go Compilation:** `PASS`. The `go mod` fetched the RC tag smoothly, and `harness.NewSession` compiled without package leaks or internal K8s dependency conflicts.
- **Kustomize Rendering:** `PASS`. `kubectl kustomize` successfully replaced the upstream `kube-slint` label with `my-operator`.
- **E2E Testing:** `PASS`. All CI pipelines remain unblocked.

## 4. Friction Analysis (Evidence)
While the onboarding is technically unblocked, the following frictions were documented from a "first-time consumer" perspective:

### 4.1 Document UX
- **Severity: Low**
- Friction: The `README.md` tutorial for patching gives a clear guide, but Kustomize beginners might still stumble over exact `patches` array syntax if they aren't using Kustomize natively. However, this is largely Kustomize's learning curve, not `kube-slint`'s fault.

### 4.2 Go API UX
- **Severity: Medium**
- Friction: `Session.Start()` does not return an error, but it feels like it should to standard Go developers. It runs asynchronously in a fire-and-forget style. (Observation: I incorrectly tried checking `err := session.Start()` initially, leading to a build failure).
- Friction: The `spec.UnsafePromKey()` structure forces users to wrap raw Prometheus metric strings. While secure, the naming (`Unsafe`) feels intimidating.

### 4.3 Kustomize Structural UX
- **Severity: Medium**
- Friction: The consumer is forced to manually override `app.kubernetes.io/name` because `samples/prometheus` ships with a hardcoded test label. The *Explicit Local Override* works perfectly, but fundamentally, Helm or Kustomize `replacements` would offer a 1-line configuration without needing a dedicated patch file. This confirms our decision to defer structural changes to the backlog while providing an active workaround.

## 5. Next Step Recommendations (Deferred)
No immediate refactoring is recommended to keep Diff small.
* **Deferred 1**: Future version should reconsider returning an `error` from `Session.Start()` or explicitly documenting its sync/async nature in a GoDoc.
* **Deferred 2**: Reconsider renaming `UnsafePromKey` to `RawPromKey` to reduce intimidation.
