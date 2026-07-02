# Post-RC Hardening Sprint

Date: 2026-07-02

## Goal

Reduce the practical risks identified in the post-RC security and gate review without changing kube-slint's product identity or collapsing measurement failure into correctness test failure.

## Sprint Length

Two weeks, split into four focused tracks. Dates are planning anchors, not release promises.

## Track 1: Secret Containment and RBAC

Target: 2026-07-02 to 2026-07-05

- Remove bearer token interpolation from curlpod command args.
- Redact command logs and command-bearing errors.
- Change generated RBAC from `ClusterRoleBinding` to namespaced `RoleBinding`.
- Update README/RBAC wording that still says ClusterRole is the default.

Definition of done:

- No generated curlpod command contains a literal bearer token from `SessionConfig.Token`.
- Runner logs mask `Bearer`, `token=`, `password=`, `passwd=`, and `secret=` values.
- `slint-gate init --emit-rbac` emits namespace-scoped RBAC by default.

## Track 2: Gate Safety and Validation

Target: 2026-07-06 to 2026-07-09

- Validate CLI `--fail-on` values.
- Validate policy `schema_version`.
- Validate policy `fail_on` values.
- Make `NO_GRADE` outrank `WARN` in aggregate gate result, while preserving first-run baseline warning semantics where intended.
- Escape GitHub Step Summary markdown table cells.

Definition of done:

- Unknown CLI fail mode exits non-zero before writing a misleading success.
- Invalid policy enum values produce `policy_status=invalid` and `gate_result=NO_GRADE`.
- A mixed `WARN` + `NO_GRADE` evaluation reports `NO_GRADE`.

## Track 3: Measurement Lifecycle and Run Identity

Target: 2026-07-10 to 2026-07-12

- Separate port-forward process lifecycle context from `PreFetch` timeout context.
- Propagate start snapshot failure into reliability/gate output.
- Generate default RunID with nanosecond resolution and a short random suffix.
- Sanitize Kubernetes label values used for run selectors and keep the original RunID in annotations or summary metadata.
- Tighten port-forward readiness to require HTTP 200.

Definition of done:

- `portforward.Fetcher` remains alive after `Session.Start()` returns.
- Start snapshot failure cannot silently produce a trusted PASS.
- Default parallel sessions do not share a second-level RunID.

## Track 4: Policy Expressiveness and Parser Hardening

Target: 2026-07-13 to 2026-07-15

- Add direction-aware regression policy design and implementation plan.
- Increase Prometheus text scanner token limit.
- Decide whether ownerRef existence checks should be cross-kind or reduced to ownerReferences-present semantics.
- Document Docker image digest pinning guidance without forcing a breaking default.

Definition of done:

- Regression direction has an accepted schema proposal or a minimal backward-compatible implementation.
- Prometheus parser handles long but bounded metric lines.
- ownerRef metric semantics are no longer misleading.

## Initial Implementation Order

1. Runner redaction.
2. curlpod token command removal.
3. RBAC template reduction.
4. CLI/policy enum validation.
5. Gate severity ordering.
6. Port-forward lifecycle.
7. RunID and label sanitization.

## Out of Scope

- Sibling repo edits.
- New Kubernetes controller code.
- Real-cluster workflow behavior changes not directly tied to the hardening items above.
- Replacing `slint-gate-summary.json` schema.
