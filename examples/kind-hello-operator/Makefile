CLUSTER_NAME       ?= slint-demo
NAMESPACE          ?= hello-system
IMAGE_NAME         ?= hello-operator:dev
# Set to "podman" when Docker is not available (requires cgroup v2 on the host).
CONTAINER_ENGINE   ?= docker
# Set to "podman" to use the rootless Podman provider for kind.
KIND_PROVIDER      ?=

KIND      = $(if $(KIND_PROVIDER),KIND_EXPERIMENTAL_PROVIDER=$(KIND_PROVIDER) kind,kind)

.PHONY: demo
demo: cluster-up build-image load-image deploy run-test gate show-result teardown  ## Full automated demo (cluster → measure → gate → teardown)

.PHONY: demo-keep
demo-keep: cluster-up build-image load-image deploy run-test gate show-result  ## Full demo, keeps the cluster running afterwards

.PHONY: cluster-up
cluster-up:  ## Create kind cluster (reuses existing)
	KIND_PROVIDER=$(KIND_PROVIDER) bash setup.sh

.PHONY: build-image
build-image:  ## Build hello-operator image
	$(CONTAINER_ENGINE) build -t $(IMAGE_NAME) operator/

.PHONY: load-image
load-image:  ## Load image into kind cluster
	$(KIND) load docker-image $(IMAGE_NAME) --name $(CLUSTER_NAME)

.PHONY: deploy
deploy:  ## Apply manifests and wait for hello-operator to be ready
	kubectl apply -f manifests/
	kubectl -n $(NAMESPACE) rollout status deployment/hello-operator --timeout=90s

.PHONY: run-test
run-test:  ## Run E2E test (creates a short-lived ServiceAccount token automatically)
	mkdir -p artifacts
	SLINT_SA_TOKEN=$$(kubectl -n $(NAMESPACE) create token kube-slint --duration=1h) \
	  go test -tags kind -v -timeout 120s -run TestHelloOperatorSLI ./e2e/

.PHONY: gate
gate:  ## Evaluate policy gate against the latest sli-summary.json
	go run ../../cmd/slint-gate \
	  --measurement-summary artifacts/sli-summary.json \
	  --policy .slint/policy.yaml \
	  --fail-on FAIL_OR_NOGRADE

.PHONY: show-result
show-result:  ## Print a summary of the gate result
	@echo ""
	@echo "════════════════════════════════════════════"
	@echo "  kube-slint demo complete"
	@echo ""
	@echo "  SLI summary : artifacts/sli-summary.json"
	@echo "  Gate result : slint-gate-summary.json"
	@if command -v jq >/dev/null 2>&1 && [ -f slint-gate-summary.json ]; then \
	  printf "  gate_result : %s\n" "$$(jq -r '.gate_result' slint-gate-summary.json)"; \
	fi
	@echo "════════════════════════════════════════════"
	@echo ""

.PHONY: teardown
teardown:  ## Delete the kind cluster
	$(KIND) delete cluster --name $(CLUSTER_NAME)

.PHONY: clean
clean:  ## Remove generated artifacts
	rm -f artifacts/sli-summary*.json slint-gate-summary.json

.PHONY: help
help:  ## Show this help
	@echo "Usage: make <target>"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
