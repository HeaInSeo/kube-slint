# GitHub Action Integration

Date: 2026-07-04
Status: Draft target contract for quality roadmap Sprint 3

## Purpose

The GitHub Action should make kube-slint usable as a CI gate without forcing
users to understand internal repository layout.

## Current Repo Action

The current local action runs `slint-gate` from source and is suitable for this
repository's own workflows.

Long-term external use should prefer a release binary.

## Target Usage

```yaml
- name: Run kube-slint gate
  uses: HeaInSeo/kube-slint/actions/slint-gate@v1
  with:
    summary: artifacts/slint/sli-summary.json
    policy: .slint/policy.yaml
    baseline: docs/baselines/sli-summary.json
    output: slint-gate-summary.json
    fail-on: FAIL_OR_NOGRADE
```

## Inputs

| Input | Required | Default | Meaning |
|---|---:|---|---|
| `summary` or `measurement-summary` | no | `artifacts/sli-summary.json` | Measurement summary path. |
| `policy` | no | `.slint/policy.yaml` | Policy path. |
| `baseline` | no | empty | Optional baseline summary path. |
| `output` | no | `slint-gate-summary.json` | Gate output path. |
| `fail-on` | no | `FAIL_OR_NOGRADE` | CI failure threshold. |
| `github-step-summary` | no | `true` | Render step summary. |
| `upload-artifact` | no | `true` | Upload gate output. |

## Outputs

| Output | Meaning |
|---|---|
| `gate-result` | `PASS`, `WARN`, `FAIL`, or `NO_GRADE`. |
| `evaluation-status` | Evaluation completeness. |
| `summary-path` | Absolute path to gate summary JSON. |

## Required Guardrails

- Unknown `fail-on` values reject.
- Default `fail-on` includes `NO_GRADE`.
- Invalid policy or summary cannot produce `PASS`.
- Step summary must not print secrets.
- Action docs must distinguish measurement failure from test failure.

## Release-Binary Target

Target flow:

1. Resolve requested kube-slint version.
2. Download release binary.
3. Verify checksum.
4. Run `slint-gate`.
5. Upload output artifact.
6. Fail CI according to `fail-on`.

## Open Decisions

- Input name compatibility: keep `measurement-summary`, add `summary`, or
  support both.
- Binary download source and checksum verification format.
- Whether action supports container mode.
- Whether action should comment on PRs or only write step summary.
