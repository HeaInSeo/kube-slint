# kube-slint Release Notes (Draft)

## Version: v1.0.0-rc.1 (Proposed)

### 1. Executive Summary
This release marks the final transition of `kube-slint` from a standalone Operator to an embeddable **Observability Library & E2E Test Harness**. All remaining standalone deployment artifacts have been permanently removed, and consumer usability has been rigorously validated through two phases of onboarding probes. 

### 2. Major Changes
- **Permanent Removal of `presets/` and `scripts/check-slo-metrics.sh`:** Following an evidence-based policy review, hardcoded Go SLI presets and manual bash scripts were completely removed. Consumers should now configure their default SLIs purely through JSON strings (e.g., `spec.UnsafePromKey("up")`) and rely on the robust `harness.Session` execution logs for debugging.
- **Legacy E2E Isolation:** The old standalone integration tests have been quarantined behind `//go:build legacy_e2e` build tags. They no longer block standard `go test ./...` workflows.
- **Improved Metrics Output:** The E2E JSON report (`sli-summary.json`) definition was strictified, ensuring seamless continuous integration artifact archiving.

### 3. Consumer Validation Probes (New Insights)
- **[Phase 4-a] Go Import Consumer UX (Success):** A minimal Kubebuilder dummy operator successfully attached the `harness.Session` without requiring any package from `presets/`. 
- **[Phase 4-b] Kustomize Remote UX (Finding):** Tested remote consumption via `github.com/HeaInSeo/kube-slint//config/...`. The path is technically functional, but identified UX debt where manifests (like `ServiceMonitor`) retain hardcoded `kube-slint` labels. External consumers must currently use local Kustomize overrides to patch these labels.

### 4. Known Limitations & Backlog
- **Kustomize Parameterization (UX Debt):** The `config/` directory still lacks a parameterized structure (e.g., Helm/nameReferences) for drop-in remote Kustomize usage.
- **Mock Integration Test:** The rewrite of `test/e2e` to act as a proper library interaction tutorial (Phase 3 actualization) is deferred to the next patch cycle.

### 5. Upgrade Guide / Impact
- **Breaking Change:** Any project relying on `github.com/.../kube-slint/presets` will fail to compile. Replace `presets.DefaultUp()` with native JSON string parsing.
- **Breaking Change:** Manual evaluation workflows using `check-slo-metrics.sh` are no longer supported. The `harness.Attach(...)` E2E execution is the single source of truth for metric calculation.
