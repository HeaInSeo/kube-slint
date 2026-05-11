#!/usr/bin/env bash
# setup.sh — bootstrap a kind cluster for the hello-operator kube-slint example
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER:-slint-demo}"
KUBE_VERSION="${KUBE_VERSION:-v1.30.0}"

if ! command -v kind &>/dev/null; then
  echo "error: kind not found. Install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
  exit 1
fi

if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "kind cluster '${CLUSTER_NAME}' already exists — reusing."
else
  echo "Creating kind cluster '${CLUSTER_NAME}' (k8s ${KUBE_VERSION})..."
  kind create cluster --name "${CLUSTER_NAME}" --image "kindest/node:${KUBE_VERSION}"
fi

echo ""
echo "Cluster ready. Next steps:"
echo "  docker build -t hello-operator:dev operator/"
echo "  kind load docker-image hello-operator:dev --name ${CLUSTER_NAME}"
echo "  kubectl apply -f manifests/"
echo "  kubectl -n hello-system rollout status deployment/hello-operator"
echo "  export SLINT_SA_TOKEN=\$(kubectl -n hello-system create token kube-slint --duration=1h)"
echo "  mkdir -p artifacts"
echo "  go test -tags kind -v -timeout 120s -run TestHelloOperatorSLI ./e2e/"
echo "  go run ../../cmd/slint-gate --measurement-summary artifacts/sli-summary.json --policy .slint/policy.yaml --fail-on FAIL"
