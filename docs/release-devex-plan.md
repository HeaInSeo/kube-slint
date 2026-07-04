# Release and DevEx Plan

Date: 2026-07-04
Status: Canonical release, GitHub Action, README IA, and UX planning source

## Purpose

This document consolidates the release policy, GitHub Action target contract,
README structure plan, and failure UX catalog.

## Release Policy

Target release artifacts:

- versioned `slint-gate` binary;
- checksums;
- release notes;
- changelog entry;
- optional container image;
- optional SBOM and provenance attestation.

Use semantic versioning:

```text
MAJOR.MINOR.PATCH
```

Breaking changes include:

- summary schema semantics changes;
- policy schema semantics changes;
- gate output field/result changes;
- default `fail-on` behavior changes;
- requiring cluster-wide RBAC for default measurement;
- changing dangerous option defaults from safe to permissive.

## GitHub Action Target

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
- action uploads `slint-gate-summary.json` when requested;
- Default `fail-on` includes `NO_GRADE`;
- invalid policy or summary cannot produce `PASS`;
- step summary must not print secrets.

## README Structure Plan

Required first-screen message:

```text
kube-slint does not replace your tests.
It measures what happens during them.

It helps catch Kubernetes operational regressions before they reach production.
```

Recommended README flow:

1. Product one-liner.
2. What kube-slint is not.
3. How it works diagram.
4. Quickstart.
5. Add to E2E test.
6. Run `slint-gate`.
7. Understand gate results.
8. Security defaults.
9. GitHub Action.
10. Docs index.

README must not:

- imply kube-slint replaces tests;
- imply measurement failure is app correctness failure;
- promote ClusterRoleBinding as the default;
- show dangerous options without explaining the risk;
- position MCP as core functionality.

## Failure UX Catalog

Failure messages should include:

- reason;
- meaning;
- fix;
- result;
- machine-readable reason code where available.

Failure messages must not include:

- ServiceAccount token;
- Authorization header;
- kubeconfig credentials;
- CI secrets;
- raw credential material.

Failure categories:

| Situation | Category | Result |
|---|---|---|
| metrics Service missing | measurement unavailable | `NO_GRADE` or config error |
| RBAC denied | measurement unavailable | `NO_GRADE` with permission hint |
| invalid policy | invalid input | reject or `NO_GRADE` |
| invalid summary | invalid input | reject or `NO_GRADE` |
| regression detected | policy violation | `FAIL` when configured |
| first-run baseline missing | first-run adoption | `WARN` or policy-defined `NO_GRADE` |
| external URL blocked | security reject | reject before scraping |
| cleanup partial failure | cleanup warning | `WARN` or explicit cleanup error |

## Open Decisions

- Binary download source and checksum verification format.
- Whether the action supports both `summary` and `measurement-summary`.
- Whether action supports container mode.
- Whether action writes PR comments or only step summary.
- Minimum supported Go and Kubernetes versions.
