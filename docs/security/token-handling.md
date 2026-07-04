# ServiceAccount Token Handling Policy

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

kube-slint may need a Kubernetes ServiceAccount token to scrape a protected
metrics endpoint from a curl pod. This document defines how token material
should be contained.

## Current Contract

The default curl pod path reads the token inside the pod:

```sh
TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
curl -H "Authorization: Bearer ${TOKEN}" ...
```

The token should not appear in:

- kubectl command arguments;
- generated PodSpec command strings after shell expansion;
- kube-slint command logs;
- command-bearing errors;
- `sli-summary.json`;
- `slint-gate-summary.json`;
- GitHub Step Summary output.

## Required Redaction Coverage

Redaction must cover at least:

- `Authorization: Bearer <value>`
- `Bearer <value>`
- `token=<value>`
- `password=<value>`
- `passwd=<value>`
- `secret=<value>`
- `"token": "<value>"`
- `token: <value>`
- `--token <value>`
- `--client-key-data <value>`
- `--certificate-authority-data <value>`

## Trust Boundary

Authorization material may be used only for the cluster-local scrape path in
default mode. It must not be sent to:

- public DNS names;
- external private DNS names;
- raw IP addresses;
- URLs derived from user-provided format strings that do not resolve to
  cluster-local service DNS.

## Recommended User Setup

- Use a dedicated ServiceAccount for kube-slint scraping.
- Scope permissions to the test namespace.
- Prefer namespace-scoped Role and RoleBinding.
- Run measurement in isolated CI/test namespaces.
- Treat generated summaries and logs as operational artifacts.
- Do not commit raw CI logs or generated summaries unless scrubbed.

## Forbidden Defaults

Default behavior must not:

- require users to create a token that is never used;
- expose literal token values in PodSpec commands;
- expose tokens in logs or errors;
- send Authorization headers to external hosts;
- require ClusterRoleBinding for normal measurement.

## Acceptance Criteria

- [ ] Token material never appears in command logs.
- [ ] Token material never appears in command-bearing errors.
- [ ] Token material never appears in generated gate summaries.
- [ ] External ServiceURLFormat cannot receive Authorization material by
  default.
- [ ] Redaction tests cover Bearer, key-value, JSON, YAML, and CLI flag forms.

## Open Decisions

- Whether external unauthenticated metrics URLs are ever supported.
- Whether legacy `SessionConfig.Token` remains indefinitely, is deprecated, or
  is replaced by a more explicit auth mode.
- Whether debug logs should include a boolean such as
  `serviceAccountTokenMounted=true`.
