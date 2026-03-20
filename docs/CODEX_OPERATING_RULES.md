# kube-slint Codex Operating Rules

## 1. Operating Goal

This repo is the product-side source for `kube-slint`.

Codex work here should keep the repo aligned as a shift-left operational SLI guardrail framework/library/harness/gate toolchain for Kubernetes Operator development. Do not drift back toward "sample operator" or "generic correctness test framework" messaging.

## 2. Source Of Truth Hierarchy

1. `docs/DECISIONS.md`
2. `docs/project-status.yaml`
3. This file
4. `docs/PROGRESS_LOG.md`
5. `README.md`
6. `docs/notes/*`
7. Historical material under `docs/old/*` and `docs/current/*`

Rules:

- Product/contract decisions come from `docs/DECISIONS.md`.
- Machine-readable automation state comes only from `docs/project-status.yaml`.
- Narrative progress and work history live in `docs/PROGRESS_LOG.md`.
- README is an external entry point, not the final authority when conflicts exist.

## 3. Repo Operating Model

- `tmux` window = one repo.
- `worktree` = one parallel change unit.
- Parallel agents may scan and analyze in parallel, but actual writes must be merged by the main thread after source conflicts are resolved.

Recommended `tmux` layout:

- Window 1: `kube-slint`
- Window 2: `hello-operator`
- Window 3: scratch or coordination

## 4. Agent Responsibilities

- Exploration agent
  - Read `docs/DECISIONS.md`, `docs/project-status.yaml`, workflows, gate code, and harness code first.
  - Find stale wording, source conflicts, and legacy assumptions.
  - No write changes.
- Implementation agent
  - Make minimal edits that align docs/config with accepted decisions.
  - Avoid functionality changes during operating-rules work.
- Docs/CI agent
  - Maintain README wording, operating docs, progress/status consistency, and workflow-facing terminology.
  - Ensure docs do not imply that correctness tests and guardrail evaluation are the same path.

## 5. When To Avoid Product Changes

Do documentation/reporting-only work when:

- The task is repo identity alignment.
- The task is source-of-truth clarification.
- The task is Codex/agent operating rules.
- The task is audit, roadmap, or reporting cleanup.
- The evidence is insufficient to safely change runtime behavior.

In those cases, stop at documentation, assumptions, and explicit TODOs.

## 6. Progress Log And Audit Rules

- Update `docs/PROGRESS_LOG.md` when stage/state materially changes.
- Do not use `docs/PROGRESS_LOG.md` as automation input.
- Record unresolved ambiguity in notes or TODO language instead of silently deciding.
- If an audit or report is superseded, lower its authority explicitly rather than deleting historical context.

## 7. Repeated Skill Candidates

Design-first skill candidates for future Codex workflows:

- `repo-scan`: source-of-truth scan and stale-doc detection
- `workflow-audit`: CI/workflow terminology and contract audit
- `consumer-friction-check`: consumer integration friction review against `hello-operator`
- `progress-log-update`: disciplined stage/status update workflow

If formal skills are added later, keep them skeleton-only until the workflow is repeated enough to justify automation.

## 8. Today’s Consolidation Result

- `kube-slint` is fixed as the product repo, not a standalone operator repo.
- `hello-operator` is fixed as the canonical consumer validation repo.
- `docs/DECISIONS.md` and `docs/project-status.yaml` are the top authority for Codex work here.
- Historical and draft material remains useful, but below the accepted decision/status layer.

## 9. Recommended Procedure

1. Read `AGENTS.md`.
2. Read `docs/DECISIONS.md` and `docs/project-status.yaml`.
3. Confirm whether the task is product behavior work or documentation/reporting work.
4. If parallel exploration is needed, keep it read-only.
5. Integrate final writes in one main thread.
6. Report facts, target state, changed files, and unresolved risks explicitly.

## 10. Codex Config Caution

`.codex/config.toml` is intentionally left without active keys until a verified Codex config schema is available locally. Do not add guessed keys.
