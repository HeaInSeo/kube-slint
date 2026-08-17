# kube-slint Constitution

<!--
  ②-form (D-12), authority revision AR-2026-08-17.1: this file does NOT own
  cross-repo invariants. It consumes the task Authority Snapshot and indexes
  only THIS repo's own enforced constraints. SoT for those is the rules
  themselves (.semgrep/rules/ and the Makefile gates), not this prose.
-->

## Cross-repo authority — verified revision-pinned repository mirror

Cross-repo platform meaning is selected by the external Authority Router. For
`AR-2026-08-17.1` the scoped authority chain is:

- platform invariants: `Platform Spec Wiki — CURRENT / 1. constitution`
- platform structure / responsibility / call direction:
  `Platform Spec Wiki — CURRENT / 2. architecture`
- repository-portable invariant mirror: `HeaInSeo/NodeVault` —
  `docs/PLATFORM_MASTER_DESIGN.md §4.1–§4.10`
- mirror verification record: `HeaInSeo/NodeVault` —
  `docs/AUTHORITY_MIRROR_VERIFICATION.md`

kube-slint does **not** treat NodeVault §4 as an independent platform canonical.
A task may consume the repository mirror for cross-repo invariant meaning only
when **all** of the following are true:

1. the task `Authority Snapshot` declares `AR-2026-08-17.1`;
2. the NodeVault verification record says `SYNC STATUS: VERIFIED`;
3. the mirror blob SHA matches the blob SHA recorded by that verification record;
4. every scoped/domain/component authority required by the kube-slint task is
   explicitly present in the task `Authority Snapshot`;
5. no semantic conflict with the current Authority Router/upstream authority has
   been detected.

If any condition is missing, `STALE`, `UNKNOWN`, mismatched, or conflicting, stop
with `AUTHORITY_CONFLICT`; do not choose a source by timestamp, filename, or
search rank. **Revision equality alone is not sufficient.**

The current repository verification record covers platform invariants only.
kube-slint work that depends on platform structure/ownership/call-direction or a
specific operational-SLI/security-policy contract must carry the exact CURRENT
architecture and relevant scoped/component contract directly in the task
`Authority Snapshot`.

## Process discipline (repo-operational — owned by this repo)

- **Deterministic gates are the guarantee** (aspirational until a ruleset
  exists). The gates below (tests, golangci-lint, custom Semgrep rules) RUN in
  CI, but **kube-slint currently has no branch ruleset, so none of them block
  merge**. Until a ruleset requires them, this is stated intent, not an enforced
  fact. LLM/agent review is advisory regardless.
- **Repo agent guidance is not duplicated here.** See `AGENTS.md` and
  `CLAUDE.md` for this repo's agent operating instructions; this constitution
  does not restate them.
- **Spec-anchored change**; **test-first** (behavioral changes ship with tests
  that fail before / pass after).
- **Local verify (before a PR):** `make test lint semgrep`.
- **Branch protection**: ⚠ NONE yet — kube-slint has no branch ruleset, so
  merge is not gated on any check and direct pushes are possible. Governance
  gap, tracked platform-wide.

## Repo-local enforced constraints (derived index — NOT canonical)

> Derived index of THIS repo's own gates. Not canonical — SoT is the gate
> itself (`.semgrep/rules/` for the Semgrep rules; the `Makefile` for the rest).

> **⚠ §1.10 status of this section:** these gates RUN in CI but are **NOT
> merge-enforced** — kube-slint has no branch ruleset, so a failing check does
> not block merge. They are therefore marked **PROPOSED** (runs, not enforced),
> not IMPLEMENTED, until a ruleset requires them.

**Custom Semgrep rules** (PROPOSED — runs via `make semgrep` but not
merge-enforced (no ruleset); SoT = `.semgrep/rules/`,
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

- **golangci-lint** (PROPOSED — runs via `make lint`, not merge-enforced; no ruleset): lint gate.
- **tests** (PROPOSED — runs via `make test`, not merge-enforced; no ruleset): unit/integration test suite (the
  target also runs `manifests generate fmt vet`).

## §1.10 — "do not record what you did not observe"

**Authority: CURRENT platform invariant under `AR-2026-08-17.1`. Enforcement in
this repo: PROPOSED.** None of this repo's seven Semgrep rules generally enforce
this invariant: they cover token handling, RBAC scope, URL validation, TLS
verification, PodSpec JSON injection, write TOCTOU, and cleanup ownership. The
platform invariant's authority status and this repo's local enforcement status
are separate axes.

## Governance

Cross-repo semantics cannot be amended by editing this constitution, a
repository mirror, or its verification record alone. They follow the task's
current Authority Snapshot; a new platform authority revision must be accepted
before repository mirrors are synchronized and independently re-verified.

**Version**: 2.1.0 | **Ratified**: 2026-08-02 | **Last Amended**: 2026-08-17
