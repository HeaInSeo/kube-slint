# roadmap-status workflow implementation note (2026-03-07)

## Implemented path

- `.github/workflows/roadmap-status.yml`

## Input source contract

- Reads exactly one automation status source:
  - `docs/project-status.yaml`
- Does not parse prose docs:
  - `docs/PROGRESS_LOG.md`
  - other markdown narratives

## Output intent

- Writes a concise status view to `GITHUB_STEP_SUMMARY`:
  - current stage
  - roadmap percent + roadmap status
  - current focus list
  - next milestone
  - capabilities summary

## Validation behavior

- Fails with clear error when:
  - `docs/project-status.yaml` is missing
  - required fields are missing
  - `capabilities` shape or enum values are invalid

## Scope boundary

- This workflow is status visibility only.
- No `slint-gate` behavior was added.
- No `baseline-update` behavior was added.
