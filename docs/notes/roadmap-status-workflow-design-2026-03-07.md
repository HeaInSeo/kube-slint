# roadmap-status workflow design note (2026-03-07)

## Scope

- This note defines the design contract only.
- Workflow implementation is intentionally deferred.

## Input Source

- `roadmap-status` reads exactly one machine-readable source:
  - `docs/project-status.yaml`
- It does not parse:
  - `docs/PROGRESS_LOG.md`
  - other prose markdown docs

## Output Contract

- Primary output target:
  - `GITHUB_STEP_SUMMARY`

## Display Fields

- current stage
- roadmap percent
- current focus
- next milestone
- capabilities summary

## Human vs Machine Document Boundary

- `docs/PROGRESS_LOG.md` remains human-readable narrative documentation.
- Automation status must come from `docs/project-status.yaml`.

## Out of Scope (separate phases)

- `slint-gate` workflow/job behavior
- `baseline-update` workflow/job behavior
- Cross-workflow orchestration among `roadmap-status`, `slint-gate`, and `baseline-update`
