# kube-slint Demo Guide

This guide is the self-contained reviewer-facing demo path for kube-slint.

The canonical external consumer validation repo is the separate GitHub repository
`github.com/HeaInSeo/hello-operator`, as recorded in `docs/DECISIONS.md` D-005.
Use that repository for consumer DX validation. Use this in-repo example when a
judge needs a compact demo that can run from one checkout.

It demonstrates three outcomes:

- `PASS`: real kind cluster, the in-repo hello-operator example emits metrics, policy gate approves.
- `FAIL`: the same measurement is evaluated against a deliberately strict policy.
- `NO_GRADE`: the measurement file is missing and the promotion gate rejects it with `FAIL_OR_NOGRADE`.

## Prerequisites

- kind v0.22 or newer
- Docker
- kubectl
- Go 1.22 or newer

## PASS Demo

From the repository root:

```sh
cd examples/kind-hello-operator
make demo
```

The demo creates a kind cluster, builds and loads the in-repo hello-operator example, deploys it, runs the kind-tagged E2E test, evaluates the gate, prints the result, and deletes the cluster.

Expected artifacts:

- `examples/kind-hello-operator/artifacts/sli-summary.json`
- `examples/kind-hello-operator/slint-gate-summary.json`

Expected gate result:

```sh
jq -r '.gate_result' slint-gate-summary.json
# PASS
```

Use `make demo-keep` instead of `make demo` if you want to inspect the cluster before teardown.

## Manual PASS Path

Use this path when recording a demo because it leaves each step visible:

```sh
cd examples/kind-hello-operator

make cluster-up
make build-image
make load-image
make deploy
make run-test
make gate
make show-result
```

The `gate` target uses:

```sh
go run ../../cmd/slint-gate \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --exit-on FAIL_OR_NOGRADE
```

`FAIL_OR_NOGRADE` is intentional for promotion-gate demos: a missing or unevaluable measurement is not treated as approval.

## FAIL Demo

Run the PASS path first so that `artifacts/sli-summary.json` exists.

Then create a deliberately strict temporary policy:

```sh
sed 's/value: 1/value: 999999/' .slint/policy.yaml > /tmp/kube-slint-fail-policy.yaml
```

Re-run the gate against the same measurement:

```sh
go run ../../cmd/slint-gate \
  --summary artifacts/sli-summary.json \
  --policy /tmp/kube-slint-fail-policy.yaml \
  --output slint-gate-summary.fail.json \
  --exit-on FAIL_OR_NOGRADE
```

Expected result:

```sh
jq -r '.gate_result' slint-gate-summary.fail.json
# FAIL
```

This shows kube-slint blocking promotion because the measured SLI does not satisfy policy.

## NO_GRADE Demo

Run the gate with a missing measurement file:

```sh
go run ../../cmd/slint-gate \
  --summary artifacts/does-not-exist.json \
  --policy .slint/policy.yaml \
  --output slint-gate-summary.nograde.json \
  --exit-on FAIL_OR_NOGRADE
```

Expected behavior:

- `gate_result` is `NO_GRADE`.
- The command exits non-zero because `FAIL_OR_NOGRADE` rejects missing measurement for promotion gates.

Inspect the summary:

```sh
jq '{gate_result, evaluation_status, measurement_status, reasons}' slint-gate-summary.nograde.json
```

## Cleanup

If you used `make demo-keep` or the manual path:

```sh
make teardown
```

Generated summaries are local artifacts and should not be committed unless intentionally converted into scrubbed fixtures.
