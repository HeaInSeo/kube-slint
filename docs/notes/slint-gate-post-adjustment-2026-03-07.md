# slint-gate post-adjustment note (2026-03-07)

## 1) pull_request trigger scope adjustment

Problem observed:
- `slint-gate` workflow had `pull_request` without path filters.
- Result: unrelated PRs could run gate and produce repetitive `NO_GRADE`.

Adjustment:
- Added `pull_request.paths` with the same scope as `push.paths`:
  - `.github/workflows/slint-gate.yml`
  - `hack/slint_gate.py`
  - `.slint/**`
  - `docs/notes/slint-gate-*.md`
  - `docs/DECISIONS.md`
  - `docs/PROGRESS_LOG.md`

Rationale:
- Keep gate visibility focused on guardrail-relevant changes.
- Reduce noise and avoid unnecessary `NO_GRADE` runs on unrelated PRs.

## 2) measurement summary expected shape (current implementation)

`hack/slint_gate.py` currently expects:
- top-level `results` array
- each result item includes:
  - `id` (string)
  - `value` (number)
- optional:
  - `reliability.collectionStatus`

If missing/corrupt:
- `measurement_status` becomes `missing` or `corrupt`
- gate result defaults to `NO_GRADE`

## 3) default summary path vs hello-operator

Current default in workflow/script:
- `artifacts/sli-summary.json`

hello-operator observed pattern:
- uses `ArtifactsDir` like `/tmp/sli-results`
- filename pattern includes run/test context (not fixed `sli-summary.json`)

Implication:
- For hello-operator integration, path override is practically required.
- Existing `workflow_dispatch` input (`measurement_summary_path`) is sufficient for now.

## 4) scope kept intentionally small

Not changed in this post-adjustment:
- gate policy model
- gate result semantics (`PASS/WARN/FAIL/NO_GRADE`)
- baseline storage strategy
- strict policy/config failure mode
