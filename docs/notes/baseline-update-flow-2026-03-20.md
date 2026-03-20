# Baseline Update Flow (2026-03-20)

Post-RC baseline update stays approval-driven.

Role split:

- repository-stored baseline (`docs/baselines/hello-operator-sli-summary.json`) remains the official RC baseline
- artifact-backed summary is only a candidate input source for baseline update review

Current minimum helper:

- `bash hack/prepare-baseline-update.sh /path/to/sli-summary.json`
- `make baseline-update-prepare BASELINE_SUMMARY=/path/to/sli-summary.json`

Canonical example:

- `make baseline-update-prepare BASELINE_SUMMARY=/opt/go/src/github.com/HeaInSeo/hello-operator/artifacts/sli-summary.json`

Artifact candidate example:

- `make baseline-update-prepare BASELINE_SUMMARY=/tmp/downloaded-artifact/sli-summary.json`

What it does:

- copies the input summary into a temporary candidate file
- prepares a normalized diff against `docs/baselines/hello-operator-sli-summary.json`
- writes a reviewer-friendly `baseline-report.md`
- evaluates current baseline and candidate with the RC policy to show gate-result changes
- prints the current baseline path, candidate path, diff path, and report path
- tells the reviewer which file would be replaced after approval

What it does not do:

- it does not overwrite the repository baseline
- it does not approve the change
- it does not update policy or RC contract
- it does not treat an artifact summary as the source of truth

Approval model:

1. candidate generation is assisted
2. baseline comparison is assisted
3. approval remains manual
4. repository baseline replacement happens only after approval

Recommended entrypoint:

- use `make baseline-update-prepare BASELINE_SUMMARY=/path/to/sli-summary.json`
- this keeps the local approval flow discoverable without adding automatic baseline replacement

Minimum reviewer flow:

1. run:
   - `make baseline-update-prepare BASELINE_SUMMARY=/opt/go/src/github.com/HeaInSeo/hello-operator/artifacts/sli-summary.json`
2. inspect:
   - `baseline-report.md`
   - `baseline.diff`
   - candidate JSON path printed by the helper
3. confirm approval target:
   - `docs/baselines/hello-operator-sli-summary.json`
4. only after approval:
   - replace the repository baseline with the printed `cp` command
