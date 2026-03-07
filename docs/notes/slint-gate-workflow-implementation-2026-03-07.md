# slint-gate workflow implementation note (2026-03-07)

## Implemented files

- `.github/workflows/slint-gate.yml`
- `hack/slint_gate.py`
- `.slint/policy.example.yaml` (example only)

## Default input paths

- policy: `.slint/policy.yaml` (workflow input default)
- measurement summary: `artifacts/sli-summary.json` (workflow input default)
- baseline summary: optional path (empty by default)

## Baseline optional handling

- baseline path empty -> `baseline_status=absent_first_run`
- baseline file missing -> `baseline_status=unavailable`
- baseline file parse failure -> `baseline_status=corrupt`
- first-run with threshold pass and no baseline -> default `gate_result=WARN`

## gate_result and workflow exit

- `gate_result=FAIL` -> workflow step exits non-zero (job fail)
- `gate_result=PASS|WARN|NO_GRADE` -> workflow succeeds
- script/runtime crash is treated as workflow failure (normal CI behavior)

## Scope intentionally excluded in this version

- remote baseline storage
- PR comment automation
- strict config failure mode (`policy missing/invalid` still defaults to `NO_GRADE`)
- advanced policy schema extensions beyond threshold/regression/reliability minimal subset
