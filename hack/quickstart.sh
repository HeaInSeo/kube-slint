#!/usr/bin/env bash
#
# hack/quickstart.sh — zero-dependency kube-slint quickstart demo
#
# Demonstrates the full slint-gate flow without a live cluster:
#   1. slint-gate init  → generates .slint/policy.yaml
#   2. pre-crafted sli-summary.json  (simulates a healthy operator run)
#   3. slint-gate       → evaluates policy and prints PASS
#
# Usage:
#   make quickstart
# or:
#   bash hack/quickstart.sh [--keep]   # --keep: don't delete workspace on exit

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATE="${REPO_ROOT}/bin/slint-gate"
KEEP=false

for arg in "$@"; do
  [[ "${arg}" == "--keep" ]] && KEEP=true
done

if [[ ! -x "${GATE}" ]]; then
  echo "Building slint-gate..."
  (cd "${REPO_ROOT}" && go build -o bin/slint-gate ./cmd/slint-gate)
fi

WORKSPACE="$(mktemp -d /tmp/kube-slint-quickstart.XXXXXX)"
cleanup() { "${KEEP}" || rm -rf "${WORKSPACE}"; }
trap cleanup EXIT

POLICY="${WORKSPACE}/.slint/policy.yaml"
SUMMARY="${WORKSPACE}/artifacts/sli-summary.json"
OUTPUT="${WORKSPACE}/slint-gate-summary.json"

mkdir -p "${WORKSPACE}/artifacts"

# ── Step 1: generate policy.yaml ──────────────────────────────────────────────
echo ""
echo "──────────────────────────────────────────────────────────────────────────"
echo " kube-slint quickstart"
echo "──────────────────────────────────────────────────────────────────────────"
echo ""
echo "[1/3] Generating policy.yaml via slint-gate init..."
"${GATE}" init --output "${POLICY}" 2>&1 | grep -v "^$" | sed 's/^/  /'

# ── Step 2: create a pre-crafted sli-summary.json ─────────────────────────────
echo ""
echo "[2/3] Creating sample sli-summary.json (simulates a healthy operator run)..."
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || python3 -c 'from datetime import datetime, timezone; print(datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"))')"

cat > "${SUMMARY}" <<JSON
{
  "schemaVersion": "slo.v3",
  "generatedAt": "${NOW}",
  "config": {
    "runId": "quickstart-demo",
    "startedAt": "${NOW}",
    "finishedAt": "${NOW}",
    "mode": { "location": "outside", "trigger": "none" },
    "tags": { "suite": "quickstart" },
    "format": "v4.4"
  },
  "reliability": {
    "collectionStatus": "Complete",
    "evaluationStatus": "Complete"
  },
  "results": [
    { "id": "reconcile_total_delta",        "value": 5, "unit": "count", "kind": "delta_counter" },
    { "id": "reconcile_error_delta",        "value": 0, "unit": "count", "kind": "delta_counter" },
    { "id": "workqueue_depth_end",          "value": 0, "unit": "items", "kind": "gauge"         },
    { "id": "rest_client_429_delta",        "value": 0, "unit": "count", "kind": "delta_counter" },
    { "id": "rest_client_5xx_delta",        "value": 0, "unit": "count", "kind": "delta_counter" }
  ]
}
JSON
echo "  ✓ ${SUMMARY}"

# ── Step 3: run slint-gate ─────────────────────────────────────────────────────
echo ""
echo "[3/3] Running slint-gate..."
"${GATE}" \
  --policy "${POLICY}" \
  --measurement-summary "${SUMMARY}" \
  --output "${OUTPUT}"

GATE_RESULT="$(python3 -c "import json,sys; d=json.load(open('${OUTPUT}')); print(d['gate_result'])" 2>/dev/null \
  || grep -o '"gate_result":"[^"]*"' "${OUTPUT}" | cut -d'"' -f4)"

echo ""
echo "──────────────────────────────────────────────────────────────────────────"
if [[ "${GATE_RESULT}" == "PASS" ]]; then
  echo " ✓ Gate Result: PASS"
  echo ""
  echo " Next steps:"
  echo "   1. Copy .slint/policy.yaml to your project (or run 'slint-gate init')"
  echo "   2. Add harness.NewSession(...) + sess.End(ctx) to your E2E test"
  echo "   3. Run: make slint-gate && ./bin/slint-gate"
  echo "   See: README.md for the full integration guide"
else
  echo " Gate Result: ${GATE_RESULT}"
  echo " (Unexpected result in quickstart — please file a bug)"
  exit 1
fi
echo "──────────────────────────────────────────────────────────────────────────"
echo ""

"${KEEP}" && echo "Workspace kept at: ${WORKSPACE}" || true
