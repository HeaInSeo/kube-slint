# kube-slint Competition Readiness Sprint

Date: 2026-06-27

This document records the short sprint plan for preparing kube-slint for open-source competition submission. It does not change product behavior or source-of-truth contracts.

## Source-of-Truth Basis

- `docs/DECISIONS.md`: kube-slint is a shift-left operational SLI guardrail, not a generic operator correctness framework.
- `docs/project-status.yaml`: current stage is post-RC hardening after Go CLI migration.
- `docs/CODEX_OPERATING_RULES.md`: keep measurement, policy evaluation, and correctness testing separate.
- `README.md`: public entry point currently points users to the self-contained `examples/kind-hello-operator/make demo` quickstart.
- `docs/DECISIONS.md` D-005: the separate GitHub repository `HeaInSeo/hello-operator` is the canonical consumer DX validation repo.

## Current Evidence

- Public API cleanup: `pkg/slint` now owns the consumer-facing session implementation; `test/e2e/harness` remains as a compatibility wrapper for the historical import path.
- Gate strictness risk: `.github/actions/slint-gate/action.yml` defaults `fail-on` to `FAIL`; `FAIL_OR_NOGRADE` and `FAIL_WARN_OR_NOGRADE` are already supported.
- Self-contained demo path exists: `examples/kind-hello-operator/Makefile` provides `make demo`, `make demo-keep`, and gate execution.
- Canonical consumer path exists outside this repository: `github.com/HeaInSeo/hello-operator` validates kube-slint as an external consumer.
- Demo gate strictness: `examples/kind-hello-operator` and competition-facing examples now use `FAIL_OR_NOGRADE` so missing measurement is not treated as promotion approval.
- Remote canonical consumer proof: `hello-operator` GitHub HEAD `f1a34a556e1a0bb39c824ea5dc7ff1f9942e017e` passed real kind SLI E2E and `slint-gate --fail-on FAIL_OR_NOGRADE` against this kube-slint worktree after narrowing the consumer test to policy-evaluated, non-skipped SLI specs.
- Token exposure risk: `pkg/slo/fetch/curlpod/client.go` injects the bearer token into the curl pod shell command.
- Compliance gap: `LICENSE` exists, but `THIRD_PARTY_LICENSES.md`, `NOTICE`, and `SECURITY.md` are absent.
- Release tag evidence: local and remote Git refs include `v1.2.0`.
- GitHub integration state: remote `origin/main` is ahead of local `main` by 6 workflow-only commits upgrading GitHub Actions versions.

## Worktree Integration Notes

Current local uncommitted files observed on 2026-06-27:

- `go.sum`
- `test/consumer-onboarding/external-onboarding-validation/go.sum`
- `test/consumer-onboarding/kubebuilder-default-sli/go.sum`
- `slint-gate-summary.json`

Assessment:

- The three `go.sum` changes are checksum-only additions. They may be legitimate output from another agent running Go tooling, but they should be validated by the owner or by the exact command that produced them before committing.
- `slint-gate-summary.json` is a generated `NO_GRADE` gate artifact with `MEASUREMENT_INPUT_MISSING`. It should not be integrated as source unless a specific fixture needs it.
- Remote workflow commits should be merged or rebased before sprint implementation to avoid working from stale CI definitions.

Recommended integration order:

1. Preserve or stash local uncommitted changes before updating local `main`.
2. Fast-forward local `main` to `origin/main`.
3. Re-run `git status --short`.
4. Decide whether the checksum-only `go.sum` additions are intentional.
5. Remove or ignore generated `slint-gate-summary.json` unless it is explicitly needed as a test fixture.

## Sprint Goal

Make kube-slint credible as a competition submission by reducing first-impression structural risk, satisfying open-source disclosure expectations, and making the demo/gate story easy to verify.

## Sprint Schedule

### Day 0 - 2026-06-27: Repository sync and scope lock

Deliverables:

- Confirm remote `main` and tag state.
- Decide whether to keep the existing `go.sum` checksum additions.
- Sync local `main` with GitHub workflow updates.
- Keep this sprint limited to public API cleanup, compliance docs, demo strictness, and submission documentation.

Exit criteria:

- Local branch is not behind `origin/main`.
- Generated artifacts are not accidentally staged.
- Sprint scope is documented here.

Status:

- Done: local `main` fast-forwarded to `origin/main` on 2026-06-27.
- Done: generated `slint-gate-summary.json` remains untracked and is not part of this sprint change.

### Day 1-2 - 2026-06-28 to 2026-06-29: Public API cleanup

Deliverables:

- Completed: moved consumer-facing session implementation out of `test/e2e/harness` into `pkg/slint`.
- Completed: left `test/e2e/harness` as a compatibility wrapper.
- Completed: kept `slint.NewSession`, `slint.SessionConfig`, `slint.DefaultSpecs`, and existing README examples source-compatible.

Exit criteria:

- Done: `pkg/slint` no longer imports `test/e2e/harness`.
- Done: focused tests for `pkg/slint`, `test/e2e/harness`, and `internal/gate` pass.
- Done: full feasible suite passed with `go test ./...`.

### Day 3 - 2026-06-30: License and security disclosure

Deliverables:

- Completed: add `THIRD_PARTY_LICENSES.md` covering direct and material indirect Go dependencies.
- Completed: add `NOTICE` for attribution clarity under Apache-2.0 distribution expectations.
- Completed: add `SECURITY.md` documenting supported reporting channel, token exposure surface, and recommended short-lived token usage.

Exit criteria:

- Done: submission reviewers can identify project license and third-party dependency licenses from top-level files.
- Done: token handling risk is explicitly documented without claiming it is fully eliminated.

### Day 4 - 2026-07-01: Promotion gate strictness

Deliverables:

- Completed: update competition-facing examples to use `FAIL_OR_NOGRADE`.
- Completed: update `examples/kind-hello-operator` gate path to demonstrate strict promotion behavior.
- Completed: keep CLI default behavior aligned with `D-002`; measurement failure is not automatically test failure.

Exit criteria:

- Done: documentation distinguishes local/dev measurement tolerance from promotion-gate strictness.
- Done: missing measurement in the demo gate does not silently look like approval.

### Day 5-6 - 2026-07-02 to 2026-07-03: Demo proof and canonical consumer validation

Deliverables:

- Completed: verify the separate GitHub `hello-operator` consumer against this kube-slint change on remote equipment.
- Pending: verify `cd examples/kind-hello-operator && make demo` as a self-contained quickstart path.
- Completed: add `docs/demo.md` with PASS and intentional FAIL/NO_GRADE reproduction steps.
- Completed: capture the exact artifacts reviewers should inspect: `artifacts/sli-summary.json` and `slint-gate-summary.json`.

Exit criteria:

- Done for documentation: a reviewer has a runnable demo path that shows measurement output and gate output.
- Done for documentation: the demo clearly shows kube-slint blocking promotion when policy or measurement conditions require it.
- Done: remote real-cluster execution proof for `hello-operator` canonical consumer validation.
- Pending: self-contained kind example proof on the submission machine.

Remote verification notes:

- Environment: `seoy@100.123.80.48`, kind `tilt-study`, node image `kindest/node:v1.30.0`.
- Reason for v1.30.0: the remote host uses cgroup v1; newer kind node images rejected kubelet startup on that host.
- Consumer-side temp patches used only in `/tmp/hello-operator-verify`: Dockerfile Go image aligned to Go 1.25, Dockerfile build path changed to `./cmd/main.go`, `.dockerignore` allowed source directories, and the SLI E2E spec set was narrowed to non-skipped metrics.
- Gate result: `PASS` with `measurement_status=ok`, `evaluation_status=evaluated`, and `fail-on=FAIL_OR_NOGRADE`.
- Self-contained `examples/kind-hello-operator && make demo` did not complete on the same remote host. The kind node reached kubelet startup but kubelet repeatedly failed with `failed to initialize top level QOS containers: root container [kubelet kubepods] doesn't exist` under rootful Podman/cgroup-v1. Treat this as a remote environment limitation until reproduced on a Docker or cgroup-v2 runner.

### Day 7 - 2026-07-04: Submission document cleanup

Deliverables:

- Update `docs/competition-submission.md` to avoid overclaiming.
- Confirm `v1.2.0` references match public GitHub tags/releases or change wording to "current candidate".
- Remove or soften future-plan items that look like uncommitted product promises.
- Add `docs/judging-guide.md` if needed for a 3-minute reviewer path.

Exit criteria:

- A judge can identify what is implemented now, what is demoed, and what is future work.
- Historical folders remain clearly lower-authority than accepted decisions and current submission docs.

### Day 8-9 - 2026-07-05 to 2026-07-06: Verification and release check

Deliverables:

- Completed for public API cleanup: focused tests for `pkg/slint`, `test/e2e/harness`, and `internal/gate`.
- Completed for public API cleanup: full feasible suite with `go test ./...`.
- Completed: canonical `hello-operator` gate path after this kube-slint change on remote equipment.
- Pending: self-contained example gate path after demo strictness changes.
- Confirm GitHub Actions are green after workflow update integration.
- Prepare release or tag notes only if actual submitted state changed materially.

Exit criteria:

- Test commands and results are recorded in the final PR or release notes.
- Known residual risks are documented rather than hidden.

## Deferred Work

- RBAC default reduction from `ClusterRoleBinding` to namespace-scoped `RoleBinding` should be treated as a separate behavior/security change. It is important, but it can affect existing users and tests.
- Reworking curl pod token handling to avoid command-line token exposure is a deeper implementation change. For this sprint, document the risk and recommended operation first.
- Moving or deleting `docs/old`, `docs/current`, or `docs/notes` should not be done as a cosmetic cleanup unless source-of-truth references are preserved.

## Reporting Template

Use this structure for sprint status updates:

- Confirmed facts from repo evidence
- Current identity
- Intended target state for the sprint
- Completed changes
- Verification results
- Open risks and deferred work
