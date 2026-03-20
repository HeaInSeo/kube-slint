# AGENTS.md

## Repo Identity

`kube-slint` is the product repo for a shift-left operational SLI guardrail framework/library/harness/gate toolchain for Kubernetes Operator development.

It is not a standalone operator repo and not a generic operator correctness test framework.

## Source Of Truth Priority

1. `docs/DECISIONS.md`
2. `docs/project-status.yaml`
3. `docs/CODEX_OPERATING_RULES.md`
4. `docs/PROGRESS_LOG.md`
5. `README.md`
6. `docs/notes/*`
7. `docs/old/*`, `docs/current/*` as reference-only history unless a higher-priority document explicitly promotes them

## Change Principles

- No product feature development unless the task explicitly requires it.
- Prefer the smallest diff that reduces ambiguity.
- Keep measurement, policy evaluation, and correctness testing conceptually separate.
- Do not let legacy standalone-operator assumptions re-enter docs, CI messaging, or test interpretation.
- If a setting or contract is uncertain, preserve current behavior and record an assumption or TODO instead of guessing.

## Test And Docs Update Rules

- If repo identity, workflow expectations, or operator-consumer contract changes, update `docs/CODEX_OPERATING_RULES.md` and the relevant source-of-truth documents in the same change.
- Update `docs/PROGRESS_LOG.md` only when actual repo work status changes; do not use it as the sole contract source.
- Treat `docs/project-status.yaml` as the only machine-readable status input for automation.
- Do not reinterpret `test/e2e` as real-cluster operator deployment E2E without explicit evidence in code and docs.

## Hard Boundaries

- No controller/operator runtime resurrection work without explicit approval.
- No workflow behavior changes unless the task is specifically about workflow behavior.
- No library semantics changes, no harness API changes, no gate output schema changes in documentation-only tasks.
- Do not edit sibling repos from this repo's task context.

## Read First

Read these before making non-trivial changes:

1. `docs/DECISIONS.md`
2. `docs/project-status.yaml`
3. `docs/CODEX_OPERATING_RULES.md`
4. `docs/PROGRESS_LOG.md`
5. `README.md`
6. `hack/slint_gate.py`
7. `.github/workflows/*`
8. `test/e2e/README.md` and `test/e2e/harness/*`

## Reporting Format

- Start with confirmed facts from repo evidence.
- Separate current identity, intended target state, and open risks.
- Name the exact source-of-truth document used for each important claim.
- If behavior was not changed, say so explicitly.

## Agent Roles

- Exploration agent: read-only repo scan, stale-doc detection, source-of-truth conflict detection.
- Implementation agent: minimal edits that align docs/config with accepted decisions.
- Docs/CI agent: maintain operating docs, README wording, workflow-facing terminology, and status document consistency.

Parallel agents are limited to read-heavy exploration and analysis. All write changes must be integrated by the main thread.
