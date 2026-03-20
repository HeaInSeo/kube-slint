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
SLINT_GATE_PY="${REPO_ROOT}/hack/slint_gate.py"
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

if [ ! -f "${SLINT_GATE_PY}" ]; then
  echo "ERROR: slint_gate.py not found: ${SLINT_GATE_PY}"
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "ERROR: python3 not found in PATH"
  echo "  This helper uses python3 to normalize JSON before diffing."
  exit 1
fi

if ! python3 -c "import yaml" >/dev/null 2>&1; then
  echo "ERROR: missing Python dependency: pyyaml"
  echo "  This helper reuses hack/slint_gate.py, which reads the RC policy YAML."
  echo "  Install it with: python3 -m pip install pyyaml"
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

python3 - "${CURRENT_BASELINE}" "${CURRENT_PRETTY}" <<'PY'
import json
import sys
src, dst = sys.argv[1], sys.argv[2]
with open(src, "r", encoding="utf-8") as f:
    data = json.load(f)
with open(dst, "w", encoding="utf-8") as f:
    json.dump(data, f, ensure_ascii=False, indent=2, sort_keys=True)
    f.write("\n")
PY

python3 - "${CANDIDATE_FILE}" "${CANDIDATE_PRETTY}" <<'PY'
import json
import sys
src, dst = sys.argv[1], sys.argv[2]
with open(src, "r", encoding="utf-8") as f:
    data = json.load(f)
with open(dst, "w", encoding="utf-8") as f:
    json.dump(data, f, ensure_ascii=False, indent=2, sort_keys=True)
    f.write("\n")
PY

if diff -u "${CURRENT_PRETTY}" "${CANDIDATE_PRETTY}" > "${DIFF_FILE}"; then
  DIFF_STATUS="no_changes"
else
  DIFF_STATUS="changes_detected"
fi

python3 "${SLINT_GATE_PY}" \
  --measurement-summary "${CURRENT_BASELINE}" \
  --policy "${RC_POLICY}" \
  --baseline "${CURRENT_BASELINE}" \
  --output "${CURRENT_GATE_FILE}" >/dev/null

python3 "${SLINT_GATE_PY}" \
  --measurement-summary "${CANDIDATE_FILE}" \
  --policy "${RC_POLICY}" \
  --baseline "${CURRENT_BASELINE}" \
  --output "${CANDIDATE_GATE_FILE}" >/dev/null

python3 - "${CURRENT_BASELINE}" "${CANDIDATE_FILE}" "${CURRENT_PRETTY}" "${CANDIDATE_PRETTY}" "${DIFF_FILE}" "${REPORT_FILE}" "${CURRENT_GATE_FILE}" "${CANDIDATE_GATE_FILE}" <<'PY'
import json
import sys
from pathlib import Path

current_path, candidate_path, current_pretty, candidate_pretty, diff_path, report_path, current_gate_path, candidate_gate_path = sys.argv[1:]

with open(current_path, "r", encoding="utf-8") as f:
    current = json.load(f)
with open(candidate_path, "r", encoding="utf-8") as f:
    candidate = json.load(f)
with open(current_gate_path, "r", encoding="utf-8") as f:
    current_gate = json.load(f)
with open(candidate_gate_path, "r", encoding="utf-8") as f:
    candidate_gate = json.load(f)

def result_map(data):
    out = {}
    for item in data.get("results", []):
        if isinstance(item, dict) and isinstance(item.get("id"), str):
            out[item["id"]] = {
                "status": item.get("status"),
                "value": item.get("value"),
            }
    return out

cur = result_map(current)
can = result_map(candidate)
all_metrics = sorted(set(cur) | set(can))

changed = []
for metric in all_metrics:
    c = cur.get(metric, {})
    n = can.get(metric, {})
    if c.get("status") != n.get("status") or c.get("value") != n.get("value"):
        changed.append(
            (
                metric,
                c.get("status"),
                c.get("value"),
                n.get("status"),
                n.get("value"),
            )
        )

cur_rel = current.get("reliability", {}) if isinstance(current.get("reliability"), dict) else {}
can_rel = candidate.get("reliability", {}) if isinstance(candidate.get("reliability"), dict) else {}
current_gate_result = current_gate.get("gate_result", "unknown")
candidate_gate_result = candidate_gate.get("gate_result", "unknown")
gate_changed = "yes" if current_gate_result != candidate_gate_result else "no"

lines = [
    "# Baseline Update Report",
    "",
    "## Paths",
    "",
    f"- Current baseline: `{current_path}`",
    f"- Candidate: `{candidate_path}`",
    f"- Normalized diff: `{diff_path}`",
    f"- Approval target: `{current_path}`",
    "",
    "## Summary",
    "",
    f"- Changed metrics: `{len(changed)}`",
    f"- Current gate result: `{current_gate_result}`",
    f"- Candidate gate result: `{candidate_gate_result}`",
    f"- Gate result changed: `{gate_changed}`",
    f"- Reliability change: `{cur_rel.get('collectionStatus')}/{cur_rel.get('evaluationStatus')}` -> `{can_rel.get('collectionStatus')}/{can_rel.get('evaluationStatus')}`",
    "",
    "## Changed Metrics",
    "",
]

if changed:
    lines.append("| Metric | Current | Candidate |")
    lines.append("|---|---|---|")
    for metric, cs, cv, ns, nv in changed:
        lines.append(f"| `{metric}` | `{cs}` / `{cv}` | `{ns}` / `{nv}` |")
else:
    lines.append("- No metric changes detected.")

lines += [
    "",
    "## Reviewer Checklist",
    "",
    "- Confirm the candidate came from an approved hello-operator summary run.",
    "- Review the changed metrics above.",
    "- Review the normalized diff file for full JSON changes.",
    "- If approved, replace the repository baseline with the candidate file.",
]

Path(report_path).write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

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
