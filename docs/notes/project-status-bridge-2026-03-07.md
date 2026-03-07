# project-status bridge note (2026-03-07)

- `docs/project-status.yaml` is the machine-readable source for Actions/summary jobs.
- `docs/PROGRESS_LOG.md` remains human-readable narrative documentation.
- Actions should read YAML status keys directly instead of parsing markdown prose.

Planned consumers:
- `slint-gate`
- `roadmap-status`
- `baseline-update`
