# Quality Roadmap Implementation Handoff

Date: 2026-07-04
Status: Frozen handoff

## Purpose

This document turns the completed quality roadmap planning sprint into
implementation-ready work. The quality/docs workstream is complete; the tasks
below are for the development agent.

Runtime behavior has not changed as part of this handoff document.

## Priority 0 Task 1: ServiceURLFormat Default-Deny Validator

Ticket:

```text
Title:
  Reject external ServiceURLFormat by default.

Background:
  ServiceURLFormat controls the metrics scrape URL. If it resolves outside the
  cluster-local .svc boundary, kube-slint could send Authorization material to
  an external host.

Required behavior:
  - Validate ServiceURLFormat before creating a curl pod.
  - Accept cluster-local service DNS only by default.
  - Reject unsupported schemes.
  - Validate service and namespace interpolation values.
  - Return a security/config error before scraping on invalid input.

Rejected behavior:
  - Sending Authorization material to public or external private DNS names.
  - Accepting https://%s.%s.evil.example/metrics.
  - Accepting ftp://%s.%s.svc/metrics.
  - Creating a curl pod before URL validation passes.

Acceptance criteria:
  - External host rejects by default.
  - .svc and .svc.cluster.local behavior is explicitly tested.
  - malformed service and namespace values reject.
  - invalid URL cannot produce PASS.

Test cases:
  - docs/test-strategy.md security URL cases.
  - unit tests for URL builder/validator.
  - optional kind E2E E2E-6.

Docs to update:
  - docs/security-model.md
  - docs/test-strategy.md
  - README.md if user-facing config changes.

Security impact:
  Prevents ServiceAccount token exfiltration through metrics URL config.
```

## Priority 0 Task 2: Dangerous Option Compatibility Plan

Ticket:

```text
Title:
  Introduce dangerous option naming for security boundary bypasses.

Background:
  Existing names such as TLSInsecureSkipVerify do not visibly communicate risk.
  The roadmap requires dangerous options to start with dangerously.

Required behavior:
  - Define compatibility behavior for existing insecure knobs.
  - Add or plan dangerous names for risky behavior.
  - Keep defaults safe.
  - Document migration and deprecation where needed.

Rejected behavior:
  - Adding ambiguous options such as allowExternal or insecure.
  - Silently changing compatibility behavior without migration notes.

Acceptance criteria:
  - Dangerous options use names from docs/security-model.md.
  - Legacy fields are documented as compatibility-only or deprecated.
  - Tests cover default safe behavior and explicit opt-in behavior.

Test cases:
  - default insecure TLS rejected or warned according to final compatibility decision.
  - dangerous opt-in allows the behavior only when explicitly set.

Docs to update:
  - docs/security-model.md
  - README.md / README(Kor).md

Security impact:
  Reduces accidental activation of security boundary bypasses.
```

## Priority 0 Task 3: Summary Bad Fixtures

Ticket:

```text
Title:
  Add executable summary invalid fixture tests.

Background:
  Invalid measurement summaries must not silently produce PASS.

Required behavior:
  - Convert planned summary fixtures into executable testdata.
  - Validate schemaVersion, generatedAt, result IDs, statuses, JSON shape, and
    non-finite values according to the final policy.

Rejected behavior:
  - Duplicate result IDs accepted as last-write-wins.
  - Unknown result status ignored.
  - malformed JSON accepted.

Acceptance criteria:
  - All summary fixture rows in docs/test-strategy.md have tests.
  - Invalid summary produces reject or NO_GRADE, never PASS.

Test cases:
  - summary/missing-schema-version.json
  - summary/wrong-schema-version.json
  - summary/duplicate-result-id.json
  - summary/unknown-result-status.json
  - summary/invalid-generated-at.json
  - summary/nan-metric-value.json
  - summary/malformed-json.json

Docs to update:
  - docs/gate-contract.md
  - docs/test-strategy.md

Security impact:
  Prevents malformed or adversarial measurement artifacts from bypassing gate.
```

## Priority 0 Task 4: Policy Bad Fixtures

Ticket:

```text
Title:
  Add executable policy invalid fixture tests.

Background:
  Invalid policies must not silently downgrade or pass gate evaluation.

Required behavior:
  - Convert planned policy fixtures into executable testdata.
  - Reject unsupported schema_version, unknown operators, duplicate threshold
    names, missing metrics, negative tolerance, non-finite values, and unknown
    fail_on values.

Rejected behavior:
  - Unknown policy enum ignored.
  - Duplicate threshold names accepted.
  - invalid policy producing PASS.

Acceptance criteria:
  - All policy fixture rows in docs/test-strategy.md have tests.
  - Invalid policy produces reject or NO_GRADE, never PASS.

Test cases:
  - policy/missing-policy-version.yaml
  - policy/wrong-policy-version.yaml
  - policy/unknown-operator.yaml
  - policy/duplicate-threshold-name.yaml
  - policy/missing-metric.yaml
  - policy/negative-tolerance.yaml
  - policy/nan-threshold-value.yaml
  - policy/unknown-fail-on.yaml

Docs to update:
  - docs/gate-contract.md
  - docs/test-strategy.md

Security impact:
  Prevents unsafe CI policy interpretation.
```

## Priority 0 Task 5: Security Bad Fixtures

Ticket:

```text
Title:
  Add executable security bad fixture tests.

Background:
  Security-sensitive defaults must be regression-tested before release.

Required behavior:
  - Add tests for external ServiceURLFormat, unsupported URL scheme,
    ClusterRoleBinding default regression, privileged curl pod, hostPath curl
    pod, kube-system target, and unsafe cleanup.

Rejected behavior:
  - Default generated RBAC uses ClusterRoleBinding.
  - Generated curl pod is privileged or mounts hostPath.
  - Cleanup can target resources without kube-slint ownership metadata.

Acceptance criteria:
  - Security fixture rows in docs/test-strategy.md have tests or
    explicit deferred implementation notes.
  - Existing quality guardrails remain passing.

Test cases:
  - security/external-service-url.yaml
  - security/external-service-url-template-injection.yaml
  - security/clusterrolebinding-default.yaml
  - security/privileged-curlpod.yaml
  - security/hostpath-curlpod.yaml
  - security/cleanup-without-owner-label.yaml

Docs to update:
  - docs/security-model.md
  - docs/test-strategy.md

Security impact:
  Prevents reintroduction of high-risk defaults.
```

## Priority 0 Task 6: Quality Guardrails Ownership

Ticket:

```text
Title:
  Keep quality guardrails required for source-of-truth and security drift.

Background:
  D-015 accepts quality guardrails as CI-backed drift detection.

Required behavior:
  - Keep .github/workflows/quality-guardrails.yml active on docs/security/gate
    relevant paths.
  - Keep hack/quality-guardrails.sh fast and dependency-light.
  - Extend only for accepted contracts or documentation drift checks.

Rejected behavior:
  - Enforcing future unimplemented runtime behavior as if it shipped.
  - Removing checks that protect accepted decisions without a replacement.

Acceptance criteria:
  - bash hack/quality-guardrails.sh passes locally.
  - Workflow path filters cover docs, security, gate, action, and guardrail
    files.

Test cases:
  - local script execution.
  - shell syntax check.
  - workflow YAML parse.

Docs to update:
  - docs/quality-guardrails.md
  - docs/CODEX_OPERATING_RULES.md

Security impact:
  Keeps accepted security and identity contracts from drifting silently.
```

## Frozen Open Decisions

The following are intentionally not decided by the quality/docs workstream:

- `.svc` only versus `.svc` plus `.svc.cluster.local` default allowlist.
- HTTP allowance for cluster-local metrics.
- External unauthenticated metrics URL support.
- NaN/Inf handling as invalid input versus measurement failure.
- Required-baseline absence as `NO_GRADE` versus `FAIL`.
- Unknown field compatibility rules.
- Legacy insecure option migration behavior.

These need implementation-owner decisions before code changes.
