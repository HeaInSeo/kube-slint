#!/usr/bin/env bash
#
# Prepare a baseline update candidate without modifying the repository baseline.
# Usage:
#   bash hack/prepare-baseline-update.sh /path/to/sli-summary.json

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CURRENT_BASELINE="${REPO_ROOT}/docs/baselines/hello-operator-sli-summary.json"
RC_POLICY="${REPO_ROOT}/.slint/policy.yaml"
INPUT_SUMMARY="${1:-}"

if [ -z "${INPUT_SUMMARY}" ]; then
  echo "ERROR: missing input summary path"
  echo "  Usage: bash hack/prepare-baseline-update.sh /path/to/sli-summary.json"
  exit 1
fi

if [ ! -f "${INPUT_SUMMARY}" ]; then
  echo "ERROR: summary file not found: ${INPUT_SUMMARY}"
  exit 1
fi

if [ ! -f "${CURRENT_BASELINE}" ]; then
  echo "ERROR: current baseline file not found: ${CURRENT_BASELINE}"
  exit 1
fi

if [ ! -f "${RC_POLICY}" ]; then
  echo "ERROR: RC policy file not found: ${RC_POLICY}"
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "ERROR: jq not found in PATH"
  echo "  Install: https://jqlang.github.io/jq/download/"
  exit 1
fi

WORK_DIR="$(mktemp -d /tmp/kube-slint-baseline-update.XXXXXX)"
CANDIDATE_FILE="${WORK_DIR}/candidate.json"
CURRENT_PRETTY="${WORK_DIR}/current.pretty.json"
CANDIDATE_PRETTY="${WORK_DIR}/candidate.pretty.json"
DIFF_FILE="${WORK_DIR}/baseline.diff"
REPORT_FILE="${WORK_DIR}/baseline-report.md"
CURRENT_GATE_FILE="${WORK_DIR}/current-gate.json"
CANDIDATE_GATE_FILE="${WORK_DIR}/candidate-gate.json"

cp "${INPUT_SUMMARY}" "${CANDIDATE_FILE}"

# Normalize JSON for diffing (sorted keys, indented).
jq --sort-keys '.' "${CURRENT_BASELINE}" > "${CURRENT_PRETTY}"
jq --sort-keys '.' "${CANDIDATE_FILE}" > "${CANDIDATE_PRETTY}"

if diff -u "${CURRENT_PRETTY}" "${CANDIDATE_PRETTY}" > "${DIFF_FILE}"; then
  DIFF_STATUS="no_changes"
else
  DIFF_STATUS="changes_detected"
fi

# Evaluate gate for current baseline and candidate using the Go CLI.
go run "${REPO_ROOT}/cmd/slint-gate" \
  --measurement-summary "${CURRENT_BASELINE}" \
  --policy             "${RC_POLICY}" \
  --baseline           "${CURRENT_BASELINE}" \
  --output             "${CURRENT_GATE_FILE}" >/dev/null

go run "${REPO_ROOT}/cmd/slint-gate" \
  --measurement-summary "${CANDIDATE_FILE}" \
  --policy             "${RC_POLICY}" \
  --baseline           "${CURRENT_BASELINE}" \
  --output             "${CANDIDATE_GATE_FILE}" >/dev/null

# Generate markdown report from gate results and metric diffs.
CURRENT_GATE_RESULT="$(jq -r '.gate_result'   "${CURRENT_GATE_FILE}")"
CANDIDATE_GATE_RESULT="$(jq -r '.gate_result' "${CANDIDATE_GATE_FILE}")"
GATE_CHANGED="$([ "${CURRENT_GATE_RESULT}" = "${CANDIDATE_GATE_RESULT}" ] && echo "no" || echo "yes")"

CUR_COLLECT="$(jq -r '.reliability.collectionStatus // "unknown"' "${CURRENT_BASELINE}")"
CAN_COLLECT="$(jq -r '.reliability.collectionStatus // "unknown"' "${CANDIDATE_FILE}")"
CUR_EVAL="$(jq -r    '.reliability.evaluationStatus // "unknown"' "${CURRENT_BASELINE}")"
CAN_EVAL="$(jq -r    '.reliability.evaluationStatus // "unknown"' "${CANDIDATE_FILE}")"

# Build changed-metric table via jq.
CHANGED_ROWS="$(jq -r --slurpfile cur "${CURRENT_PRETTY}" --slurpfile can "${CANDIDATE_PRETTY}" -n '
  def result_map(data):
    (data[0].results // []) | map(select(.id != null))
    | map({(.id): {status: .status, value: .value}}) | add // {};
  ($cur | result_map(.)) as $cm |
  ($can | result_map(.)) as $nm |
  (($cm | keys) + ($nm | keys)) | unique | sort |
  map(
    . as $k |
    {
      metric: $k,
      cur_status: ($cm[$k].status // "—"),
      cur_value:  ($cm[$k].value  // "—" | tostring),
      can_status: ($nm[$k].status // "—"),
      can_value:  ($nm[$k].value  // "—" | tostring)
    } |
    select(.cur_status != .can_status or .cur_value != .can_value) |
    "| `\(.metric)` | `\(.cur_status)` / `\(.cur_value)` | `\(.can_status)` / `\(.can_value)` |"
  ) | .[]
')"

{
  echo "# Baseline Update Report"
  echo ""
  echo "## Paths"
  echo ""
  echo "- Current baseline: \`${CURRENT_BASELINE}\`"
  echo "- Candidate: \`${CANDIDATE_FILE}\`"
  echo "- Normalized diff: \`${DIFF_FILE}\`"
  echo "- Approval target: \`${CURRENT_BASELINE}\`"
  echo ""
  echo "## Summary"
  echo ""
  echo "- Current gate result: \`${CURRENT_GATE_RESULT}\`"
  echo "- Candidate gate result: \`${CANDIDATE_GATE_RESULT}\`"
  echo "- Gate result changed: \`${GATE_CHANGED}\`"
  echo "- Reliability change: \`${CUR_COLLECT}/${CUR_EVAL}\` -> \`${CAN_COLLECT}/${CAN_EVAL}\`"
  echo ""
  echo "## Changed Metrics"
  echo ""
  if [ -n "${CHANGED_ROWS}" ]; then
    echo "| Metric | Current | Candidate |"
    echo "|---|---|---|"
    echo "${CHANGED_ROWS}"
  else
    echo "- No metric changes detected."
  fi
  echo ""
  echo "## Reviewer Checklist"
  echo ""
  echo "- Confirm the candidate came from an approved hello-operator summary run."
  echo "- Review the changed metrics above."
  echo "- Review the normalized diff file for full JSON changes."
  echo "- If approved, replace the repository baseline with the candidate file."
} > "${REPORT_FILE}"

echo "Baseline update candidate prepared."
echo ""
echo "Current baseline : ${CURRENT_BASELINE}"
echo "Candidate copy   : ${CANDIDATE_FILE}"
echo "Diff file        : ${DIFF_FILE}"
echo "Report file      : ${REPORT_FILE}"
echo "Diff status      : ${DIFF_STATUS}"
echo ""
echo "Reviewer flow:"
echo "  1. Inspect the report: ${REPORT_FILE}"
echo "  2. Inspect the candidate JSON: ${CANDIDATE_FILE}"
echo "  3. Inspect the normalized diff: ${DIFF_FILE}"
echo "  4. If approved, replace the repository baseline with:"
echo "     cp ${CANDIDATE_FILE} ${CURRENT_BASELINE}"
echo "  5. Record the reason for the update in docs/PROGRESS_LOG.md or the approval PR."
