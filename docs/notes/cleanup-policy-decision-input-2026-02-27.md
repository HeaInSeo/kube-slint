# Cleanup Policy Decision Input

**Date:** 2026-02-27
**Purpose:** Provide options and recommendations for evaluating questionable/legacy directories and scripts before making irreversible changes. 

## 1. `presets/` Directory (Repo-wide Representative Trace)

### Current Status
- **Location:** `presets/controller_runtime/` and `presets/my_operator/`.
- **State:** Inactive (API drift).
- **Contents:** `v1.go` files containing 100% block-commented out functions (`// func ...`).
- **References:** The namespace or logical structure may be loosely referenced in legacy documentation, but structurally there is ZERO live invocation in `pkg/slo` or `test/e2e/harness` via Go code.

### Rationale for Lingering
Early in the project, `kube-slint` may have planned a "Registry" model where operators could dynamically program SLIs with Go code (e.g. `reg.MustRegister(CRCreatedDelta())`). However, the project evolved towards the "Configurable SLI Specs (JSON/YAML)" approach via E2E Hooks where standard Prometheus expressions suffice without hard-coupled Go models.

### Maintenance / Confusion Cost
- Confuses newcomers reading `presets/` who might think they are supposed to import it into their operator logic.
- Pollutes text searches and refactoring tools. 

### Options Matrix

| Option | Pros | Cons |
| --- | --- | --- |
| **Keep as-is** | Safest; avoids touching code completely. | High confusion cost. Retains obsolete junk. |
| **Keep but clarify** | Better than as-is; adds a `README.md` explaining it's a graveyard. | Defeats the "library" purpose. Still clutter. |
| **Relocate (e.g. to docs/)** | Preserves historical intent while avoiding compilation footprint. | Go Code format is unnatural for modern JSON-based docs. |
| **Delete** | Repo remains strictly library-oriented and lightweight. | We lose the "inspiration" snippets if we don't document them elsewhere. |

### Recommendation (Not Final)
**Delete**. Since `kube-slint` is configured entirely by `SessionConfig.Specs` (JSON array) and PromQL metrics, offering "hardcoded Go SLIs" directly contradicts the new design philosophy. If users need examples, they should look at the "JSON Payload Examples" inside `docs/current/...`

---

## 2. `scripts/check-slo-metrics.sh`

### Current Status
- **Location:** `scripts/check-slo-metrics.sh`
- **State:** Inactive / Not wired to `Makefile` standard runs.
- **References:** Not used by standard CI workflows or `test/e2e`.

### Rationale for Lingering
Before the V4 Harness (Session Engine) correctly polled and evaluated PromQL JSON, developers likely ran `check-slo-metrics.sh` with raw commands like `curl localhost:8080/metrics | grep workqueue_adds_total` to check if metrics were cleanly scraped by the cluster.

### Maintenance / Confusion Cost
- Exposes a secondary, obsolete mechanism that is no longer truthful. The current `curlPodFetcher` logic does exactly this within `session.go` and handles multi-pod resolution properly.

### Options Matrix

| Option | Pros | Cons |
| --- | --- | --- |
| **Keep as-is** | Might still be marginally useful for local ad-hoc curl. | Diverges deeply with proper `harness` mechanism. |
| **Keep but clarify** | Can explicitly state "For manual debugging ONLY". | Another file to maintain. |
| **Relocate** | N/A (Scripts folder is technically fine). | N/A |
| **Delete** | Aligns with the "Harness is the Source of Truth" philosophy. | Recreating curl commands locally takes a minute if someone needs it. |

### Recommendation (Not Final)
**Delete**. Since it undermines the E2E verification cycle (which now fully wraps curl behavior inside its adapter engine), maintaining it provides zero value and promotes manual intervention over automated `sli-summary.json` generation.

---

## 3. Approval Checkbox (For User + ChatGPT)

- [ ] `presets/`: Delete / Relocate / Keep?
- [ ] `check-slo-metrics.sh`: Delete / Clarify / Keep?
