# Release Policy

Date: 2026-07-04
Status: Draft policy for quality roadmap Sprint 3

## Purpose

kube-slint is intended to run in CI. External users need reproducible release
artifacts instead of relying on `go run` from a source checkout.

## Release Artifacts

Target release artifacts:

- versioned `slint-gate` binary;
- checksums;
- release notes;
- changelog entry;
- optional container image;
- optional SBOM and provenance attestation.

## Versioning

Use semantic versioning:

```text
MAJOR.MINOR.PATCH
```

Guidance:

- PATCH: bug fix, docs fix, non-breaking hardening.
- MINOR: additive policy/summary fields, new integrations, new checks.
- MAJOR: breaking schema, CLI, action input, or gate output contract changes.

## GitHub Action Direction

Long-term target usage:

```yaml
- uses: HeaInSeo/kube-slint/actions/slint-gate@v1
  with:
    summary: artifacts/slint/sli-summary.json
    policy: .slint/policy.yaml
    fail-on: FAIL_OR_NOGRADE
```

Target behavior:

- action downloads or uses a release binary;
- action does not require `go run` for normal external users;
- action exposes gate result outputs;
- action uploads `slint-gate-summary.json` when requested.

## Checksums

Every release binary should have a checksum file.

Recommended:

```text
slint-gate_<version>_checksums.txt
```

## Container Image

Open decision:

- publish `ghcr.io/heainseo/slint-gate:<version>`;
- require digest pinning in docs for production CI examples;
- keep source-based action path for local repo testing only.

## Breaking Change Policy

Breaking changes include:

- changing summary schema semantics;
- changing policy schema semantics;
- changing gate output field names or result values;
- changing default `fail-on` behavior;
- requiring cluster-wide RBAC for default measurement;
- changing dangerous option defaults from safe to permissive.

Breaking changes require:

- release note callout;
- migration guide;
- schema/version bump where applicable;
- compatibility test or explicit removal note.

## Minimum Version Policy

Open decisions:

- minimum supported Go version;
- minimum supported Kubernetes version;
- supported OS/architecture matrix.

These should be finalized in `docs/support/*` during the 10-point backlog.
