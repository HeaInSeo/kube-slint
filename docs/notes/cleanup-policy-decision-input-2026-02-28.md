# Cleanup Policy Decision Input

**Date:** 2026-02-28
**Purpose:** Provide options and recommendations for evaluating questionable/legacy directories and scripts before making irreversible changes. 

## 1. `presets/` Directory (Repo-wide Representative Trace)

### Current Status
- **Location:** `presets/controller_runtime/` and `presets/my_operator/`.
- **Role:** Originally intended to provide hardcoded Go SLI presets for users to import.
- **State:** Inactive (API drift).
- **Contents:** `v1.go` files containing 100% block-commented out functions (`// func ...`).
- **References:** The namespace or logical structure may be loosely referenced in legacy documentation, but structurally there is ZERO live invocation in `pkg/slo` or `test/e2e/harness` via Go code.

### Rationale for Lingering
Early in the project (v1-v3), `kube-slint` may have planned a "Registry" model where operators could dynamically program SLIs with Go code (e.g. `reg.MustRegister(CRCreatedDelta())`). However, the project evolved towards the "Configurable SLI Specs (JSON/YAML)" approach via E2E Hooks where standard Prometheus expressions suffice without hard-coupled Go models.

### Maintenance / Confusion Cost
- Confuses newcomers reading `presets/` who might think they are supposed to import it into their operator logic.
- Pollutes text searches and refactoring tools. 
- Contradicts the current "non-invasive" and "JSON-configured" design principles.

### Options Matrix

| Option | Pros | Cons |
| --- | --- | --- |
| **Keep as-is** | Safest; avoids touching code completely. | High confusion cost. Retains obsolete junk. |
| **Keep but clarify** | Better than as-is; adds a `README.md` explaining it's a graveyard. | Defeats the "library" purpose. Still clutter. |
| **Relocate (e.g. to docs/)** | Preserves historical intent while avoiding compilation footprint. | Go Code format is unnatural for modern JSON-based docs. |
| **Soft-deprecate** | Signals intent to remove without breaking anything instantly. | Just delays the inevitable clean up. |
| **Delete** | Repo remains strictly library-oriented and lightweight. | We lose the "inspiration" snippets if we don't document them elsewhere. |

### Recommendation (Current Policy)
**Delete (Conditional)**. Since `kube-slint` is configured entirely by `SessionConfig.Specs` (JSON array) and PromQL metrics, offering "hardcoded Go SLIs" directly contradicts the new design philosophy. 
**Deletion Condition**: Confirm that equivalent JSON payload examples exist in `docs/current/...` (Custom SLI Tutorial covers this). **AND** successfully complete **Phase 4-a (Consumer Onboarding Probe)** to prove that a library consumer can perform default SLI measurement without relying on any code from `presets/`. Once this Phase 4-a evidence is secured, it is safe to delete.

---

## 2. `scripts/check-slo-metrics.sh`

### Current Status
- **Location:** `scripts/check-slo-metrics.sh`
- **Role:** Maintainer helper for manual metric verification.
- **State:** Inactive / Not wired to `Makefile` standard runs or CI.
- **References:** Not used by standard CI workflows or `test/e2e`.

### Rationale for Lingering
Before the V4 Harness (Session Engine) correctly polled and evaluated PromQL JSON, developers likely ran `check-slo-metrics.sh` with raw commands like `curl localhost:8080/metrics | grep workqueue_adds_total` to check if metrics were cleanly scraped by the cluster before writing parsing logic.

### Maintenance / Confusion Cost
- Exposes a secondary, obsolete mechanism that is no longer truthful. The current `curlPodFetcher` logic does exactly this within `session.go` and handles multi-pod resolution properly.
- Might mislead contributors into thinking they need to run this script manually during development.

### Options Matrix

| Option | Pros | Cons |
| --- | --- | --- |
| **Keep as-is** | Might still be marginally useful for local ad-hoc curl. | Diverges deeply with proper `harness` mechanism. |
| **Keep but clarify** | Can explicitly state "For manual debugging ONLY". | Another file to maintain. |
| **Relocate** | N/A (Scripts folder is technically fine). | N/A |
| **Migrate to Make** | Standardizes the helper command. | Still bypasses the E2E harness logic. |
| **Delete** | Aligns with the "Harness is the Source of Truth" philosophy. | Recreating curl commands locally takes a minute if someone needs it. |

### Recommendation (Current Policy)
**Delete (Conditional)**. Since it undermines the E2E verification cycle (which now fully wraps curl behavior inside its adapter engine), maintaining it provides zero value and promotes manual intervention over automated `sli-summary.json` generation.
**Deletion Condition**: Verify that the harness logging `curlPodFetcher` is verbose enough to debug metric fetching failures. **AND** successfully complete **Phase 4-a / 4-b (Consumer Onboarding / UX Probe)** to prove that the automated E2E testing pathway provides sufficient telemetry without needing a manual bash script. If the Phase 4 probes log adequately on failure, the script is fully obsolete.

---

## 3. Approval Checkbox (For User + ChatGPT)

- [ ] `presets/`: Approve conditional delete policy (JSON examples + Phase 4-a success evidence)?
- [ ] `check-slo-metrics.sh`: Approve conditional delete policy (harness debug sufficiency + Phase 4-a/4-b evidence)?
