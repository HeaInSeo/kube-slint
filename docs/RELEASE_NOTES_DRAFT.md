# kube-slint v1.0.0-rc.1

## Executive Summary

This release marks the completion of kube-slint’s transition into an embeddable **observability library and E2E harness**, with policy-driven cleanup, consumer validation, and a modernized integration test strategy.

Key outcomes in this RC:

* legacy standalone artifacts and obsolete helper assets were removed via an evidence-based cleanup process,
* consumer onboarding was validated from both **Go import** and **Kustomize remote resource** perspectives,
* the old flaky legacy E2E path was replaced with a fast, deterministic, **mock-based in-memory integration test** flow.

---

## Highlights

### 1) Policy-backed cleanup completed

Following staged evidence collection and approval-based execution, the following obsolete assets were removed:

* `presets/` (hardcoded Go SLI preset remnants)
* `scripts/check-slo-metrics.sh` (manual metric inspection script)

These removals reflect kube-slint’s current design direction:

* **JSON/YAML-driven SLI specs**
* **harness-based automated evaluation**
* **non-invasive instrumentation philosophy**

---

### 2) Consumer validation completed (two perspectives)

#### Phase 4-a — Go import consumer onboarding (Success)

A minimal consumer operator path successfully imported kube-slint and executed a default SLI measurement flow using library APIs and JSON-equivalent metric declarations.

This confirmed:

* `presets/` is not required for consumer onboarding,
* the harness/session path is viable for external consumers,
* debugging information emitted by the harness is sufficient for normal diagnosis workflows.

#### Phase 4-b — Kustomize remote consumer UX probe (Findings)

Remote Kustomize consumption using pinned refs (e.g. `github.com/...//config/...?...`) was verified to be **technically functional**, but UX debt was identified in the payload structure.

Main finding:

* remote fetch works,
* but current Kustomize assets still contain hardcoded assumptions (for example, labels tied to `kube-slint`), which makes direct remote consumption inconvenient without local overrides.

This is tracked as **follow-up UX debt**, not a release blocker for this RC.

---

## 3) Legacy E2E replacement completed (modernized test strategy)

The historical standalone-controller-oriented E2E path has been fully replaced by a **mock-based, in-memory integration test** strategy centered on the harness engine.

### What changed

* Legacy quarantined E2E assets (including old `e2e_test.go` / `e2e_suite_test.go` and related legacy support assets) were removed after replacement coverage was proven.
* `test/e2e/README.md` was updated to document the new official integration testing path.
* The `test-e2e` Makefile path was modernized to run the fast mock-based integration test instead of provisioning a Kind cluster.

### Why this is better

* **Much faster** (milliseconds instead of heavy cluster startup)
* **Deterministic**
* **No Kubernetes cluster dependency**
* **Low flakiness**
* Better aligned with kube-slint’s current identity as a **library/harness**, not a standalone operator runtime

---

## 4) Mock-based integration coverage (Phase 3 actualization)

The new in-memory integration path now covers core evaluation behaviors, including:

* **Happy Path**
* **Missing Metric handling**
* **Fetch Error handling**
* **Delta computation path (`ComputeDelta`)**
* **Multi-metric mixed pass/fail result handling**

This provides a significantly stronger and more maintainable foundation than the prior legacy path.

---

## Breaking Changes / Compatibility Notes

### Removed: `presets/`

Projects relying on historical `presets/...` imports will fail to compile.

**Recommended migration:**

* Replace hardcoded preset helpers with JSON/YAML `SessionConfig.Specs` definitions and `spec.UnsafePromKey(...)` (or equivalent current spec declarations) in test/harness-driven flows.

### Removed: `scripts/check-slo-metrics.sh`

Manual bash-based metric checking is no longer part of the supported workflow.

**Recommended workflow:**

* Use harness/session execution and inspect structured output and diagnostic logs from the evaluation pipeline.

---

## Known Limitations (Deferred / Backlog)

The following items are intentionally deferred and are **not blockers** for `v1.0.0-rc.1`:

* **Kustomize parameterization / remote UX debt**

  * hardcoded labels and non-parameterized resource assumptions still limit drop-in remote Kustomize consumption.
* **Post-release harness refactors**

  * deferred cleanup/refactor tasks (e.g. fetcher modularization and related internal debt) remain tracked in backlog.

---

## Upgrade / Adoption Guidance

For new users and consumers:

* Prefer the **mock-based integration harness path** under `test/e2e` for deterministic validation.
* Use **JSON/YAML SLI specs** rather than legacy code presets.
* Treat the Kustomize remote assets as usable building blocks, but expect local customization until parameterization improvements land.

---

## What’s next

Planned next tracks after this RC:

* Kustomize parameterization / consumer UX improvements (Stage D follow-up debt)
* Additional post-release cleanup/refactor items already tracked in `PROGRESS_LOG.md`

---

## Tag

* **Tag:** `v1.0.0-rc.1`

This RC is intended to mark the first fully cleaned, consumer-validated, library-centric release milestone of kube-slint.
