# kind + hello-operator Example

End-to-end demonstration of kube-slint measuring a real operator running inside a kind cluster.

## Quick path

```bash
# Prerequisites: kind ≥ v0.22, Docker, Go 1.25+
make demo
```

`make demo` runs the full cycle — cluster creation, image build, deploy, E2E test, gate evaluation, and teardown — in one command. Use `make demo-keep` to leave the cluster running for inspection after the run.

## What this example shows

| Step | What happens |
|---|---|
| 1. Deploy `hello-operator` | A minimal Go service that emits Prometheus counters on `:8080/metrics` |
| 2. `sess.Start()` | kube-slint launches a curl pod to capture the pre-workload snapshot |
| 3. Workload runs | `hello-operator` fires reconcile loops in the background |
| 4. `sess.End()` | kube-slint captures the post-workload snapshot, computes deltas, writes `artifacts/sli-summary.json` |
| 5. `slint-gate` | Evaluates `sli-summary.json` against `.slint/policy.yaml`; exits 0 by default, exits 1 with `--fail-on FAIL` |

## Prerequisites

- [kind](https://kind.sigs.k8s.io/) ≥ v0.22
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Docker](https://docs.docker.com/get-docker/) (for building the image)
- Go 1.25+

## Manual steps

If you prefer to run steps individually instead of `make demo`:

```bash
# 1. Create the kind cluster
./setup.sh

# 2. Build and load hello-operator
# Build context is the operator/ directory — no repo-root dependency required.
docker build -t hello-operator:dev operator/
kind load docker-image hello-operator:dev --name slint-demo

# 3. Deploy hello-operator + RBAC
kubectl apply -f manifests/

# 4. Wait for the pod to be ready
kubectl -n hello-system rollout status deployment/hello-operator

# 5. Get a bearer token for the kube-slint ServiceAccount
export SLINT_SA_TOKEN=$(kubectl -n hello-system create token kube-slint --duration=1h)

# 6. Run the E2E test
# The -tags kind flag is required — the test file is guarded by //go:build kind
# to keep it out of the default `go test ./...` run.
mkdir -p artifacts
SLINT_SA_TOKEN=$SLINT_SA_TOKEN go test -tags kind -v -timeout 120s -run TestHelloOperatorSLI \
  github.com/HeaInSeo/kube-slint/examples/kind-hello-operator/e2e

# 7. Evaluate policy gate (run from examples/kind-hello-operator/)
go run ../../cmd/slint-gate \
  --measurement-summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --fail-on FAIL

# 8. Tear down
kind delete cluster --name slint-demo
```

## How to get the ServiceAccount token

The harness uses `kubectl` to create a temporary curl pod in your namespace. `SessionConfig.Token`
is forwarded to `curl` as an `Authorization: Bearer <token>` header when scraping the metrics
endpoint. For operators that require token-authenticated `/metrics`, use a short-lived
ServiceAccount token. (hello-operator serves plain HTTP without auth, so any non-empty token works.)

Use one of these approaches:

### Option A — `kubectl create token` (recommended for CI/local dev)

```bash
# Short-lived token (1 hour) — pass via env var
export SLINT_SA_TOKEN=$(kubectl -n hello-system create token kube-slint --duration=1h)
```

### Option B — Token from a running pod

If your E2E test runs inside a Kubernetes pod with the `kube-slint` ServiceAccount mounted:

```go
token, err := slint.ReadServiceAccountToken(slint.DefaultTokenPath)
// or
token, err := slint.ReadServiceAccountTokenFromEnv("SLINT_SA_TOKEN", slint.DefaultTokenPath)
```

### Option C — Long-lived Secret (Kubernetes < 1.24 or legacy)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kube-slint-token
  namespace: hello-system
  annotations:
    kubernetes.io/service-account.name: kube-slint
type: kubernetes.io/service-account-token
```

```bash
export SLINT_SA_TOKEN=$(kubectl -n hello-system get secret kube-slint-token \
  -o jsonpath='{.data.token}' | base64 -d)
```

## How to use a non-HTTPS operator

By default kube-slint connects to `https://<service>.<namespace>.svc:8443/metrics`.
`hello-operator` exposes plain HTTP on port 8080. In your `SessionConfig` set:

```go
sess := slint.NewSession(slint.SessionConfig{
    ...
    ServiceURLFormat: slint.ServiceURLHTTP, // "http://%s.%s.svc:8080/metrics"
})
```

For a custom port or path:

```go
ServiceURLFormat: "http://%s.%s.svc:9090/metrics",
```

## File structure

```
kind-hello-operator/
  operator/
    main.go        -- hello-operator metrics server, stdlib-only (no build tag)
    Dockerfile     -- builds the hello-operator image (context: operator/)
  manifests/
    namespace.yaml
    deployment.yaml  -- Deployment + Service
    rbac.yaml        -- kube-slint ServiceAccount + ClusterRole
  e2e/
    e2e_test.go    -- example E2E test (//go:build kind — excluded from go test ./...)
  .slint/
    policy.yaml    -- gate policy for hello-operator metrics
  setup.sh         -- kind cluster bootstrap (cluster creation only)
  README.md
```

## Policy gate

The included `.slint/policy.yaml` fails CI if:
- `hello_reconcile_delta` is less than 1 (operator ran no reconcile loops)
- `hello_workqueue_depth_end` exceeds 5 (operator workqueue is backed up)

These IDs match `results[].id` in `sli-summary.json`, not raw Prometheus metric names.

Run the gate separately after any E2E suite that writes `artifacts/sli-summary.json`:

```yaml
# .github/workflows/e2e.yml
- uses: ./.github/actions/slint-gate
  with:
    measurement-summary: artifacts/sli-summary.json
    policy: examples/kind-hello-operator/.slint/policy.yaml
    fail-on: FAIL_OR_NOGRADE
```
