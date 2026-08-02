# kube-slint Constitution

<!--
  ②-form (D-12): this file does NOT own cross-repo invariants. It references the
  platform canonical constitution and indexes only THIS repo's own enforced
  constraints. SoT for those is the rules themselves (.semgrep/rules/ and the
  Makefile gates), not this prose.
-->

## Cross-repo invariants live in the Platform Spec Wiki (canonical)

Cross-repo invariants — reproducibility, `casHash`, `stableRef`, the artifact
dual-axis (`lifecycle_phase` / `integrity_health`), the sori boundary, and
"do not record what you did not observe" (§1.10) — are owned solely by the
**Platform Spec Wiki `1. constitution`**. This document does not restate or fork
them; on any conflict, the wiki §1 wins.

## Process discipline (repo-operational — owned by this repo)

- **Deterministic gates are the guarantee.** Merge is decided by deterministic
  checks (tests, golangci-lint, the custom Semgrep rules). LLM/agent review is
  **advisory**: a passing review never merges alone, a failing gate is never
  overridden.
- **Repo agent guidance is not duplicated here.** See `AGENTS.md` and
  `CLAUDE.md` for this repo's agent operating instructions; this constitution
  does not restate them.
- **Spec-anchored change**; **test-first** (behavioral changes ship with tests
  that fail before / pass after).
- **Local verify (before a PR):** `make test lint semgrep`.
- **Branch protection**: `main` lands via PR with required checks; no direct
  pushes.

## Repo-local enforced constraints (derived index — NOT canonical)

> Derived index of THIS repo's own gates. Not canonical — SoT is the gate
> itself (`.semgrep/rules/` for the Semgrep rules; the `Makefile` for the rest).

**Custom Semgrep rules** (IMPLEMENTED — `make semgrep`; SoT = `.semgrep/rules/`,
fixtures validated by `make semgrep-test`):

- **`kube-slint-no-bearer-token-in-curl-args`** — forbids interpolating a token
  into a command string (`fmt.Sprintf` with a `Bearer %s` format); the token
  must be read in-pod from the mounted ServiceAccount token file, never placed
  in args, PodSpec command strings, or logs.
- **`kube-slint-no-clusterrolebinding-default`** — default generated RBAC must
  stay namespace-scoped (ServiceAccount + Role + RoleBinding); never emit
  `ClusterRole`/`ClusterRoleBinding`.
- **`kube-slint-no-direct-service-url-format`** — forbids building the metrics
  scrape URL via a raw `fmt.Sprintf($X.ServiceURLFormat, ...)`; must go through
  `curlpod.ValidateMetricsURL` (the default-deny host/scheme/value check).
- **`kube-slint-no-insecure-skip-verify`** — forbids setting
  `TLSInsecureSkipVerify`/`InsecureSkipVerify` to `true` directly; skipping TLS
  verification must be the explicit, visibly-named `DangerouslySkipTLSVerify`
  opt-in, default false.
- **`kube-slint-no-raw-json-splice-in-podspec`** — forbids building a
  PodSpec/`--overrides` JSON payload with a `fmt.Sprintf` format string
  containing a JSON key literal (sibling-key injection risk); marshal a typed
  struct via `encoding/json` instead.
- **`kube-slint-no-stat-before-write`** — forbids the `os.Stat`-then-write
  (`os.WriteFile`/`os.Create`) check-then-act TOCTOU pattern; use an atomic
  primitive (`os.OpenFile` with `O_CREATE|O_EXCL`).
- **`kube-slint-no-unsafe-cleanup`** — forbids a `kubectl delete pods <name>`
  construction with no label selector in the same command
  (cleanup-without-ownership-check).

**Other gates:**

- **golangci-lint** (IMPLEMENTED — `make lint`): lint gate.
- **tests** (IMPLEMENTED — `make test`): unit/integration test suite (the
  target also runs `manifests generate fmt vet`).

## §1.10 — "do not record what you did not observe"

**Status: PROPOSED (not enforced in this repo).** §1.10 is a cross-repo
invariant owned by the wiki. None of this repo's seven Semgrep rules enforce it:
they cover token handling, RBAC scope, URL validation, TLS verification, PodSpec
JSON injection, write TOCTOU, and cleanup ownership — none constrain recording
only observed values. Marked PROPOSED, not IMPLEMENTED, until a deterministic
gate exists.

**Version**: 1.0.0 | **Ratified**: 2026-08-02 | **Last Amended**: 2026-08-02
