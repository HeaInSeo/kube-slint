# Post-RC Hardening Design

Date: 2026-07-02

## Confirmed Facts

- `docs/DECISIONS.md` defines kube-slint as a shift-left operational SLI guardrail, not an operator correctness framework.
- `docs/DECISIONS.md` D-002 says measurement failure is not automatically test failure.
- `docs/DECISIONS.md` D-008 says `slint-gate` is a separate policy evaluation layer over measurement outputs.
- `docs/project-status.yaml` lists post-RC hardening as the next milestone.
- `pkg/slo/fetch/curlpod/client.go` currently embeds the bearer token in the curl pod shell command.
- `pkg/kubeutil/runner.go` logs the full command line before execution.
- The external review found that `cmd/slint-gate/init.go` emitted `ClusterRole` and `ClusterRoleBinding`.
- The external review found that `internal/gate/gate.go` allowed unknown `fail_on` policy values and let `WARN` outrank `NO_GRADE`.

## Current Identity

kube-slint remains a library/harness/gate toolchain for applying operational SLI guardrails during Kubernetes Operator development. It measures what happens during an existing test session and evaluates the resulting artifacts through a separate gate.

This hardening work does not turn kube-slint into a standalone operator, a generic correctness test framework, or an always-fail-on-measurement-error test runner.

## Intended Target State

1. Secret-bearing values are not embedded in Kubernetes Pod command args or command logs.
2. Generated RBAC defaults are namespace-scoped unless a future task explicitly introduces cluster-wide behavior.
3. Port-forward lifecycle is tied to the measurement session, not to a short scrape timeout context.
4. Start snapshot collection failures are represented in machine-readable reliability/gate output.
5. `NO_GRADE` outranks `WARN` for aggregate gate results, except for explicitly documented first-run baseline behavior.
6. CLI and policy enums reject unknown values instead of silently falling back.
7. Run IDs used in Kubernetes label selectors are generated and sanitized for label-value safety.
8. Regression checks use metric direction before treating a baseline difference as a regression.

## Non-Goals

- No controller/operator runtime resurrection.
- No harness API rewrite unless required to close the lifecycle bug.
- No gate output schema replacement.
- No real-cluster E2E reinterpretation of `test/e2e`.
- No feature expansion beyond security, lifecycle, and gate correctness hardening.

## Design Decisions

### Secret Handling

The curl pod should read its bearer token from the mounted ServiceAccount token file inside the pod:

```sh
TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
curl -H "Authorization: Bearer ${TOKEN}" ...
```

`SessionConfig.Token` remains available for compatibility until a dedicated deprecation decision exists, but the default curlpod path should prefer the in-pod token when `ServiceAccountName` is configured.

Command execution logs must pass through redaction before logging or returning command-bearing errors. Existing `pkg/slo/evidence.RedactString` is the preferred helper.

### RBAC Scope

`slint-gate init --emit-rbac` should emit:

- `ServiceAccount`
- namespaced `Role`
- namespaced `RoleBinding`

Default rules should avoid cluster-wide binding and remove unused verbs where current code evidence does not require them.

### Measurement Failure Semantics

Start snapshot failure should not be hidden as a stderr-only warning. It should be visible in summary reliability data and should become `NO_GRADE` in gate evaluation when it prevents a trustworthy policy decision.

This preserves D-002 because the E2E test can continue, while the gate can still fail CI when the caller uses `FAIL_OR_NOGRADE` or `FAIL_WARN_OR_NOGRADE`.

### Gate Severity

Aggregate severity should be:

```text
FAIL > NO_GRADE > WARN > PASS
```

First-run baseline absence may remain `WARN` when regression is optional/first-run friendly, but corrupt or unavailable required inputs should remain `NO_GRADE`.

### Validation

The following values should be validated:

- CLI `--fail-on`
- action `fail-on`
- policy `schema_version`
- policy `fail_on`
- policy reliability levels
- future regression direction values

Unknown values should produce `policy_status=invalid`, `gate_result=NO_GRADE`, or a CLI usage error depending on where they are found.

## Open Risks

- Moving token sourcing to in-pod ServiceAccount token changes behavior for users who rely on an externally supplied token that differs from the scraper ServiceAccount token.
- Role/RoleBinding defaults can break users who expected a single generated ClusterRoleBinding to work across namespaces.
- Direction-aware regression requires a policy schema extension; the first patch should validate existing behavior before changing the schema.
- Start snapshot failure propagation may require a small internal interface extension so `Session.Start()` can pass collection status into `Session.End()`.

## Verification Plan

- Unit tests for command redaction and curlpod command construction.
- Unit tests for RBAC template output.
- Unit tests for `fail-on` validation and `NO_GRADE` severity priority.
- `go test ./pkg/kubeutil ./pkg/slo/fetch/curlpod ./cmd/slint-gate ./internal/gate`.
- If toolchain permits, `go test ./...`.

## Sprint Plan (2026-07-03)

A second-round review against the code that landed the items above (`fa4bab7`,
`55b625c`) confirmed most of them are done, and found a small set of residual
gaps. This sprint closes the highest-risk residual gaps in dependency order.

1. **R1 — `CollectionStatus=Failed` must never silently resolve to `PASS`.**
   Today, `runReliability` only warns when `reliability.required: true`. A
   policy with no threshold rules and `reliability.required: false` (or
   unset-and-false) can reach `computeGateResult` with `failed=false,
   hasWarn=false, hasNoGrade=false` even though the underlying measurement
   never completed, and get `PASS`. Fix: collection failure becomes an
   unconditional `NO_GRADE`, independent of `reliability.required`.

2. **R4 — PreFetch/scrape context timeout must not cap the pod-wait timeout
   it's supposed to bound.** `Session.Start()` wraps `PreFetch` in a context
   built from `ScrapeTimeout` (2m), then passes that context into
   `CurlPod.Run(ctx, WaitPodDoneTimeout=5m, LogsTimeout=2m)`, which derives its
   own wait/log sub-timeouts from `ctx` via `context.WithTimeout` — so the
   inherited 2m deadline silently overrides the 5m wait budget. Once R1 ships,
   every prefetch that legitimately needs more than 2m to schedule will surface
   as a hard gate `NO_GRADE` instead of being swallowed, so this is bundled
   with R1 rather than deferred. Fix: derive the outer context for pod-backed
   fetch calls (both `PreFetch` and the steady-state `Fetch`) from
   `WaitPodDoneTimeout + LogsTimeout + margin`, not `ScrapeTimeout`.

3. **R2 — regression checks must be direction-aware.** `evalRegressionCheck`
   currently flags any `abs(deltaPercent) > tolerance`, so an improvement
   (e.g. a 30% latency reduction) is flagged as a "regression" exactly like a
   real regression. Fix: reuse the paired threshold rule's `operator`
   (`<=`/`<` = lower-is-better, `>=`/`>` = higher-is-better) to decide which
   direction counts as a regression; unknown/`==` operators keep the existing
   symmetric check. This avoids the policy schema extension flagged as an open
   risk above, since the direction signal already exists on the threshold rule.

4. **N1 — orphan sweep selector must use the sanitized RunID.** `sweep.go`'s
   `slint-run-id!=<runID>` exclusion selector uses the raw RunID while the pod
   labels themselves (and the plain cleanup path) use
   `SanitizeKubernetesLabelValue(runID)`. For any RunID that sanitization
   actually changes, the sweep selector becomes syntactically invalid and
   orphan sweep stops working for that RunID. Fix: sanitize consistently.

5. **N2 — dead `Token` requirement / implicit `automountServiceAccountToken`.**
   `curlpod.Client.RunOnce` now discards the caller-supplied token and reads
   the pod's own mounted ServiceAccount token instead, but
   `validateSessionConfigOrFail` still hard-fails the test if
   `SessionConfig.Token` is empty, forcing new users to mint a token that is
   never used. Separately, the generated PodSpec relies on the
   ServiceAccount's default `automountServiceAccountToken`, so a SA with
   automount disabled breaks the new token path with no clear signal. Fix:
   drop the `Token`-required validation (field stays for compatibility per the
   design decision above) and set `automountServiceAccountToken: true`
   explicitly on the curl pod override.

Deferred to a later pass (unchanged from the mapping table below): N3
(redaction pattern coverage for JSON-shaped tokens), N5 (`Session.End`
unconditional `Stop()` contract), N6 (workflow demo-fixture default
labeling), R6 (`internal/gate` → `pkg/gate`, engine stdout hygiene).

## Sprint Plan follow-up (N4, R5)

Closed in a follow-up pass, same day:

6. **N4 — `POLICY_INVALID` diagnostic hint didn't mention `schema_version`.**
   `validatePolicy` strictly requires `schema_version: "slint.policy.v1"`
   (rejecting missing/old values as `policy_status=invalid`), but
   `cmd/slint-gate/diagnose.go`'s `POLICY_INVALID` hint only talked about YAML
   syntax and unsupported operators. A user upgrading from a policy.yaml
   without `schema_version` would see `POLICY_INVALID` with no hint pointing
   at the actual cause. Fix: added an explicit `schema_version: slint.policy.v1`
   hint (plus `fail_on`/`reliability.min_level` hints) to the diagnostic entry.

7. **R5 — example RBAC was still cluster-scoped.** The
   `slint-gate init --emit-rbac` template already emits a namespaced
   `Role`/`RoleBinding`, but `examples/kind-hello-operator/manifests/rbac.yaml`
   still defined a `ClusterRole`/`ClusterRoleBinding`, contradicting the
   namespace-scoped-by-default decision above and the fixed init template.
   Fix: converted the example to a namespaced `Role`/`RoleBinding` in
   `hello-system`, keeping the existing ServiceAccount name (`kube-slint`)
   referenced by `e2e_test.go`/`Makefile` unchanged.

## Sprint Plan follow-up (R3)

8. **R3 — curlpod and portforward fetchers disagreed on metric semantics.**
   `pkg/slint/fetcher_curlpod.go`'s `parsePrometheusText` synthesized a
   bare-name key summing every labeled series for a metric (so specs can
   reference `reconcile_total` instead of every
   `reconcile_total{controller="..."}` combination), but
   `pkg/slo/fetch/portforward/fetcher.go` called `promtext.ParseTextToMap`
   directly and never got that aggregate — so a bare-name-keyed spec that
   worked under curlpod silently went `missing`/`NO_GRADE` under portforward
   for the exact same target. Fix: moved the aggregation into a shared
   `promtext.Aggregate`/`ParseTextToMapWithAggregates` (pkg/slo/fetch/promtext),
   used by both fetchers. While moving it, also closed two aggregation traps
   the original curlpod-only version didn't guard against:
   - a real unlabeled series with the same name as an aggregated one is left
     untouched instead of being summed into (avoids double counting when an
     exporter emits both a plain total and a per-label breakdown);
   - series carrying an `le` (histogram bucket) or `quantile` (summary
     quantile) label are excluded from aggregation, since those are
     cumulative/positional and summing them is meaningless.

   Not in scope for this pass: full `# TYPE` comment parsing (the `le`/
   `quantile` label heuristic covers the common case without it) and the
   `strings.Fields`-based parser's inability to handle label values
   containing spaces (mapping table item P0-FETCH-004 / F4) — both remain
   open, deferred alongside N3/N5/N6/R6.
