#!/usr/bin/env bash
set -euo pipefail

failures=0

fail() {
  echo "::error::$*"
  failures=$((failures + 1))
}

pass() {
  echo "ok: $*"
}

require_file() {
  local path="$1"
  if [[ -f "$path" ]]; then
    pass "found $path"
  else
    fail "missing required file: $path"
  fi
}

require_grep() {
  local pattern="$1"
  local path="$2"
  local label="$3"
  if grep -Eq -- "$pattern" "$path"; then
    pass "$label"
  else
    fail "$label"
  fi
}

reject_grep() {
  local pattern="$1"
  local path="$2"
  local label="$3"
  if grep -Eq -- "$pattern" "$path"; then
    fail "$label"
  else
    pass "$label"
  fi
}

check_source_of_truth() {
  echo "== source-of-truth guardrails =="
  require_file docs/DECISIONS.md
  require_file docs/project-status.yaml
  require_file docs/CODEX_OPERATING_RULES.md
  require_file docs/PROGRESS_LOG.md
  require_file README.md

  require_grep 'D-001: kube-slint identity = shift-left operational quality guardrail' docs/DECISIONS.md \
    "decision log keeps shift-left guardrail identity"
  require_grep 'D-002: measurement failure != test failure' docs/DECISIONS.md \
    "decision log separates measurement failure from test failure"
  require_grep 'D-008: slint-gate is a separate policy evaluation layer' docs/DECISIONS.md \
    "decision log keeps slint-gate as separate policy layer"
  require_grep 'D-015: Quality roadmap contracts are CI-guarded planning inputs' docs/DECISIONS.md \
    "decision log accepts quality roadmap CI-guarded planning contracts"
  require_grep 'status_path = Path\("docs/project-status.yaml"\)' .github/workflows/roadmap-status.yml \
    "roadmap-status workflow reads docs/project-status.yaml"
}

check_canonical_docs() {
  echo "== canonical docs guardrails =="
  require_file docs/quality-roadmap.md
  require_file docs/quality-roadmap-implementation-handoff.md
  require_file docs/security-model.md
  require_file docs/gate-contract.md
  require_file docs/test-strategy.md
  require_file docs/release-devex-plan.md
  require_file docs/quality-guardrails.md

  require_grep 'Progress: 100%' docs/quality-roadmap.md \
    "quality roadmap reports complete planning progress"
  require_grep 'operator-first, dataplane-aware shift-left operational SLI' docs/quality-roadmap.md \
    "quality roadmap uses accepted product framing"
  require_grep 'Priority 0 Task 1: ServiceURLFormat Default-Deny Validator' docs/quality-roadmap-implementation-handoff.md \
    "implementation handoff includes ServiceURLFormat validator task"
}

check_public_api_doc_sync() {
  echo "== public API doc-comment sync guardrails =="
  # Found by pre-release-adversarial-review (2026-07-08): SessionConfig's
  # StrictnessMode doc comment listed only 3 of the 4 modes propagation.go
  # actually implements, hiding RequiredSLIs from the public API surface.
  require_grep 'RequiredSLIs' pkg/slint/session.go \
    "SessionConfig.StrictnessMode doc comment lists all implemented modes"
}

check_security_contract() {
  echo "== security guardrails =="
  require_file SECURITY.md

  require_grep 'curl pod path, the pod reads its own mounted' SECURITY.md \
    "SECURITY.md documents in-pod token read path"
  require_grep 'ServiceAccount token from' SECURITY.md \
    "SECURITY.md documents in-pod ServiceAccount token read"
  require_grep 'bearer token should not appear in kubectl command arguments' SECURITY.md \
    "SECURITY.md documents command/log token containment"
  reject_grep 'token is currently command-line visible|passed to curl as an `Authorization: Bearer \.\.\.` header in the pod command' SECURITY.md \
    "SECURITY.md has no stale token-in-command limitation"

  require_grep 'Default-Deny Patterns' docs/security-model.md \
    "security model documents default-deny patterns"
  require_grep 'Any option that allows behavior rejected by the default security policy must' docs/security-model.md \
    "security model defines dangerous option rule"
  require_grep 'begin with `dangerously`' docs/security-model.md \
    "security model requires dangerously prefix"
  require_grep 'Default mode accepts only cluster-local metrics hosts' docs/security-model.md \
    "security model keeps cluster-local ServiceURLFormat requirement"
  require_grep 'never send Authorization material to an external host' docs/security-model.md \
    "ServiceURLFormat policy blocks external Authorization material"
  require_grep 'Token material never appears in command logs' docs/security-model.md \
    "token handling policy blocks command-log token exposure"
}

check_rbac_contract() {
  echo "== rbac guardrails =="
  require_grep 'kind: RoleBinding' cmd/slint-gate/init.go \
    "slint-gate init emits RoleBinding in default RBAC template"
  reject_grep 'kind: ClusterRoleBinding' cmd/slint-gate/init.go \
    "slint-gate init default RBAC template does not emit ClusterRoleBinding"
  require_grep 'assert.NotContains\(t, body, "ClusterRoleBinding"\)' cmd/slint-gate/init_test.go \
    "unit test guards against default ClusterRoleBinding regression"
  require_grep 'Default generated RBAC must not use' docs/security-model.md \
    "RBAC model documents default cluster-wide RBAC rejection"

  # Found by pre-release-adversarial-review (2026-07-08): Session.Cleanup()
  # and SweepOrphansWithResult() issued `kubectl delete`/`kubectl get`
  # against a caller-supplied namespace with no kube-system/kube-public/
  # kube-node-lease rejection, even though docs/security-model.md documents
  # that rejection as an unconditional default. Any file that shells out to
  # kubectl delete against a session/config namespace must also reference
  # the shared guard so this can't silently regress per-file.
  require_grep 'kubeutil\.IsDangerousNamespace' pkg/slint/session.go \
    "Session.Cleanup() enforces the kube-system namespace guard"
  require_grep 'kubeutil\.IsDangerousNamespace' pkg/slint/sweep.go \
    "SweepOrphansWithResult() enforces the kube-system namespace guard"
}

check_secret_redaction_contract() {
  echo "== secret redaction guardrails =="
  require_grep 'Bearer\\s\+' pkg/slo/evidence/redact.go \
    "redaction covers Bearer token shape"
  require_grep 'token\|password\|passwd\|secret\|serviceaccounttoken\|clientsecret' pkg/slo/evidence/redact.go \
    "redaction covers common secret key names"
  require_grep 'Authorization.*Bearer \[REDACTED\]' pkg/slo/evidence/redact_test.go \
    "redaction tests cover Authorization bearer headers"
}

check_curlpod_security_contract() {
  # As of 2026-07-08 (PodSpec/JSON-injection fix), the --overrides payload is
  # built via encoding/json.Marshal of a typed struct (podOverride in
  # client.go) rather than a raw fmt.Sprintf JSON template, so these check
  # for the Go struct-literal field assignments, not literal JSON text.
  echo "== curlpod securityContext guardrails =="
  require_grep 'AutomountServiceAccountToken: true' pkg/slo/fetch/curlpod/client.go \
    "curlpod explicitly mounts ServiceAccount token"
  require_grep 'AllowPrivilegeEscalation: false' pkg/slo/fetch/curlpod/client.go \
    "curlpod disables privilege escalation"
  require_grep 'Capabilities:.*Drop: \[\]string{"ALL"}' pkg/slo/fetch/curlpod/client.go \
    "curlpod drops Linux capabilities"
  require_grep 'RunAsNonRoot: *true' pkg/slo/fetch/curlpod/client.go \
    "curlpod runs as non-root"
  require_grep 'SeccompProfile:.*Type: "RuntimeDefault"' pkg/slo/fetch/curlpod/client.go \
    "curlpod uses RuntimeDefault seccomp"
  require_grep 'isValidDNSLabel\(serviceAccountName\)' pkg/slo/fetch/curlpod/client.go \
    "curlpod validates ServiceAccountName before use (PodSpec-injection guard)"
}

check_kubectl_delete_pod_resource_naming() {
  echo "== kubectl delete pod/pods naming guardrails =="
  # Convention: a delete targeting one named pod uses singular "pod"; a
  # delete targeting a label selector (potentially multiple pods) uses
  # plural "pods". Found by the third pre-release-adversarial-review pass
  # (2026-07-09): two selector-based deletes used singular "pod" and one
  # name-based delete used plural "pods" - an unexplained divergence
  # between the closest-analogous cleanup paths in the same packages.
  require_grep '"kubectl", "delete", "pods",' pkg/slo/fetch/curlpod/client.go \
    "curlpod CleanupByLabel (selector-based) uses plural 'pods'"
  require_grep 'kubectl.*delete.*pods.*-l.*labelSelector' pkg/slint/session.go \
    "session.go's label-selector delete uses plural 'pods'"
  require_grep 'kubectl.*delete.*pod.*name.*-n' pkg/slint/sweep.go \
    "sweep.go's name-based delete uses singular 'pod'"
}

check_cli_dispatch_error_printing() {
  echo "== CLI dispatch error printing guardrails =="
  local file="cmd/slint-gate/main.go"
  # Every dispatch case in main() must print the error it received before
  # calling os.Exit(1) - a case that exits silently with no diagnostic is a
  # real regression (the `inspect` dispatch bug found by the second
  # pre-release-adversarial-review pass, 2026-07-08: every sibling case
  # printed "slint-gate <cmd>: %v" before exiting, but `inspect` didn't).
  # This fails if `os.Exit(1)` is the ONLY statement inside an
  # `err != nil { ... }` block anywhere in the file (no Fprintf between).
  if grep -Pzoq 'err != nil \{\n\t+os\.Exit\(1\)\n\t+\}' "$file"; then
    fail "a CLI dispatch case in $file calls os.Exit(1) without printing the error first"
  else
    pass "every CLI dispatch case in $file prints its error before os.Exit(1)"
  fi
}

check_gate_contract() {
  echo "== gate workflow guardrails =="
  require_grep 'default: FAIL_OR_NOGRADE' .github/actions/slint-gate/action.yml \
    "GitHub Action default fail-on includes NO_GRADE"
  require_grep 'exit-on:[[:space:]]+FAIL_OR_NOGRADE' .github/workflows/slint-gate.yml \
    "slint-gate workflow uses FAIL_OR_NOGRADE"
  require_grep 'FAIL > NO_GRADE > WARN > FIRST_RUN_WARNING > PASS' docs/gate-contract.md \
    "gate contract documents conservative gate priority"
  require_grep 'Invalid schema version cannot produce PASS' docs/gate-contract.md \
    "summary schema contract blocks invalid schema PASS"
  require_grep 'Missing or unsupported `schema_version` is rejected' docs/gate-contract.md \
    "policy schema contract rejects missing/unsupported version"
  require_grep 'Invalid policy or summary input must not produce `PASS`' docs/gate-contract.md \
    "gate semantics block invalid input PASS"

  local inspect_in_github
  inspect_in_github=$(grep -REn 'slint-gate inspect|cmd/slint-gate.*inspect' .github 2>/dev/null || true)
  if [[ -z "$inspect_in_github" ]]; then
    pass ".github workflows/actions do not use inspect as an enforcing gate step"
  else
    fail ".github workflows/actions must call the gate evaluator for enforcement, not inspect: $inspect_in_github"
  fi
  require_grep 'Inspect readiness is not a gate verdict' docs/gate-contract.md \
    "gate contract separates inspect readiness from gate verdicts"
}

check_metric_key_api_contract() {
  echo "== metric key API guardrails =="
  require_grep 'func InputKey' pkg/slo/spec/spec.go \
    "source-neutral InputKey helper exists"
  require_grep 'spec.InputKey\("memstats.Alloc"\)' README.md \
    "README uses InputKey for non-Prometheus JSON/expvar input"
  require_grep 'spec.InputKey\("memstats.Alloc"\)' "README(Kor).md" \
    "README(Kor).md uses InputKey for non-Prometheus JSON/expvar input"
  require_grep 'spec.PromMetric\("workqueue_adds_total", spec.Labels' test/consumer-onboarding/external-onboarding-validation/main.go \
    "consumer onboarding uses PromMetric for labeled Prometheus metric"
  require_grep 'spec.InputKey\("up"\)' test/consumer-onboarding/kubebuilder-default-sli/integration_test.go \
    "kubebuilder onboarding uses InputKey for simple source-neutral input"
  reject_grep 'UnsafePromKey\("up"\)|UnsafePromKey\(`up`\)|UnsafePromKey\("operator_up"\)|UnsafePromKey\("events_processed_total"\)|UnsafePromKey\("error_rate"\)' test/consumer-onboarding/kubebuilder-default-sli/integration_test.go \
    "kubebuilder onboarding does not use UnsafePromKey for simple input keys"
  reject_grep 'UnsafePromKey\("workqueue_adds_total\{name=' test/consumer-onboarding/external-onboarding-validation/main.go \
    "external onboarding does not use UnsafePromKey for ordinary labeled Prometheus metrics"
}

check_test_strategy() {
  echo "== test strategy guardrails =="
  require_grep 'summary/missing-schema-version.json' docs/test-strategy.md \
    "test strategy includes missing summary schema fixture"
  require_grep 'policy/unknown-operator.yaml' docs/test-strategy.md \
    "test strategy includes unknown policy operator fixture"
  require_grep 'security/external-service-url.yaml' docs/test-strategy.md \
    "test strategy includes external ServiceURLFormat fixture"
  require_grep 'E2E-6 \| External ServiceURLFormat configured \| reject before scraping' docs/test-strategy.md \
    "kind E2E matrix includes external ServiceURLFormat rejection"
  require_grep 'invalid input produces `PASS`' docs/test-strategy.md \
    "E2E acceptance rejects invalid-input PASS"
}

check_test_capture_helper_consolidation() {
  echo "== test capture helper consolidation guardrails =="
  # os.Pipe()-based output capture must go through the canonical, already-
  # fixed helpers (cmd/slint-gate/inspect_test.go's captureStdout,
  # pkg/gate/gate_test.go's captureStderr) rather than reinventing a
  # synchronous-drain-after-fn() version, which deadlocks on output larger
  # than the OS pipe buffer. Found 5 duplicate instances of the buggy
  # pattern by the third pre-release-adversarial-review pass (2026-07-09);
  # all were consolidated onto these two canonical helpers.
  local unexpected
  unexpected=$(grep -rl 'os\.Pipe()' --include='*_test.go' cmd/slint-gate pkg/gate 2>/dev/null \
    | grep -v -E '^(cmd/slint-gate/inspect_test\.go|pkg/gate/gate_test\.go)$' || true)
  if [[ -z "$unexpected" ]]; then
    pass "no new os.Pipe()-based capture helpers outside the two canonical ones"
  else
    fail "new os.Pipe() usage outside the canonical capture helpers (reuse captureStdout/captureStderr instead of reinventing): $unexpected"
  fi
}

check_flag_deprecation_docs() {
  echo "== flag deprecation docs guardrails =="
  # --summary is the current/preferred flag; --measurement-summary is the
  # deprecated alias (cmd/slint-gate/main.go). Both READMEs' flag tables
  # previously called --summary its own deprecated alias (self-referential
  # and backwards) - found by the second pre-release-adversarial-review pass.
  reject_grep '`--summary` still works as a deprecated alias' README.md \
    "README.md does not call --summary its own deprecated alias"
  reject_grep '`--summary`는 하위호환용 deprecated alias' "README(Kor).md" \
    "README(Kor).md does not call --summary its own deprecated alias"
  require_grep '`--measurement-summary` still works as a deprecated alias' README.md \
    "README.md correctly names --measurement-summary as the deprecated alias"
  require_grep '`--measurement-summary`는 하위호환용 deprecated alias' "README(Kor).md" \
    "README(Kor).md correctly names --measurement-summary as the deprecated alias"
  # `ci github-actions` generates a ready-to-paste Actions step - it must
  # emit the preferred `summary:` action input, not the deprecated
  # `measurement-summary:` alias. Found by the third pre-release-adversarial-
  # review pass (2026-07-09): every other onboarding surface had already
  # migrated to --summary, but this one generator was missed.
  reject_grep 'measurement-summary: %s' cmd/slint-gate/ci_github_actions.go \
    "ci github-actions generates the preferred summary: action input, not measurement-summary:"
}

check_identity_wording() {
  echo "== product identity guardrails =="
  require_grep 'does not replace your tests\. It measures what happens during them\.' README.md \
    "README keeps test-vs-measurement message"
  require_grep 'does not replace your tests' docs/release-devex-plan.md \
    "release/devex plan preserves test-vs-measurement first-screen message"
  reject_grep 'generic Kubernetes YAML linter|Prometheus replacement|functional test framework replacement' README.md \
    "README does not describe kube-slint as a generic linter, Prometheus replacement, or test replacement"
}

check_release_and_ux_contract() {
  echo "== release and ux guardrails =="
  require_grep 'action downloads or uses a release binary' docs/release-devex-plan.md \
    "release/devex plan targets release-binary based GitHub Action"
  require_grep 'Default `fail-on` includes `NO_GRADE`' docs/release-devex-plan.md \
    "GitHub Action integration keeps NO_GRADE in default fail-on"
  require_grep 'Failure messages must not include' docs/release-devex-plan.md \
    "failure catalog includes secret exclusion rule"
}

check_source_of_truth
check_canonical_docs
check_public_api_doc_sync
check_security_contract
check_rbac_contract
check_secret_redaction_contract
check_curlpod_security_contract
check_kubectl_delete_pod_resource_naming
check_cli_dispatch_error_printing
check_gate_contract
check_metric_key_api_contract
check_test_strategy
check_test_capture_helper_consolidation
check_flag_deprecation_docs
check_identity_wording
check_release_and_ux_contract

if (( failures > 0 )); then
  echo "quality guardrails failed: ${failures} issue(s)"
  exit 1
fi

echo "quality guardrails passed"
