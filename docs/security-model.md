# Security Model

Date: 2026-07-04
Status: Canonical security planning contract

## Purpose

This document consolidates kube-slint's security defaults, dangerous option
naming, ServiceURLFormat policy, token handling, RBAC model, and static
guardrail plan.

kube-slint is a shift-left operational SLI guardrail. Its security posture
protects the measurement path without turning kube-slint into a cluster
management tool.

## Default-Deny Patterns

| Pattern | Default policy |
|---|---|
| External metrics URL | reject |
| Authorization header to external host | reject |
| `InsecureSkipVerify` | reject or explicit dangerous opt-in |
| `ClusterRoleBinding` | reject in default path |
| privileged pod | reject |
| `hostPath` | reject |
| `kube-system` target namespace | reject by default |
| cleanup without owner label | reject |
| unknown policy enum | reject |
| unknown result status | reject |

## Dangerous Option Naming

Any option that allows behavior rejected by the default security policy must
begin with `dangerously`.

Examples:

```yaml
dangerouslyAllowExternalMetricsURL: true
dangerouslySkipTLSVerify: true
dangerouslyAllowClusterWideRBAC: true
dangerouslyAllowKubeSystemNamespace: true
dangerouslyAllowUnsafeCleanup: true
```

Dangerous options must document the default safe behavior, the risk being
accepted, the narrow use case, and the tests covering default rejection and
explicit opt-in.

## ServiceURLFormat Policy

Default mode accepts only cluster-local metrics hosts.

Allowed default shapes:

```text
https://<service>.<namespace>.svc:8443/metrics
https://<service>.<namespace>.svc.cluster.local:8443/metrics
http://<service>.<namespace>.svc:<port>/metrics
```

Rejected default shapes:

```text
https://evil.example.com/collect?svc=%s&ns=%s
https://%s.%s.evil.com/metrics
ftp://%s.%s.svc/metrics
https://10.0.0.10/metrics
```

Required behavior:

- validate ServiceURLFormat before creating a curl pod;
- parse the formatted URL with a structured URL parser;
- validate service and namespace values before URL construction;
- reject unsupported schemes;
- reject external hosts by default;
- never send Authorization material to an external host in default mode.

## Token Handling

The default curl pod path reads the token inside the pod:

```sh
TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
curl -H "Authorization: Bearer ${TOKEN}" ...
```

Token material must not appear in:

- kubectl command arguments;
- generated PodSpec command strings after shell expansion;
- kube-slint command logs;
- command-bearing errors;
- `sli-summary.json`;
- `slint-gate-summary.json`;
- GitHub Step Summary output.

Token material never appears in command logs.

Redaction must cover at least:

- `Authorization: Bearer <value>`;
- `Bearer <value>`;
- `token=<value>`;
- `password=<value>`;
- `passwd=<value>`;
- `secret=<value>`;
- JSON, YAML, and CLI flag-shaped secret values.

## RBAC Model

Default generated RBAC uses:

- `ServiceAccount`;
- namespaced `Role`;
- namespaced `RoleBinding`.

Default generated RBAC must not use:

- `ClusterRole`;
- `ClusterRoleBinding`.

Expected namespace-scoped permissions:

| Resource | Verbs | Purpose |
|---|---|---|
| `pods` | `get`, `list`, `create`, `delete` | Create and clean up curl pod. |
| `pods/log` | `get` | Read scrape result from curl pod logs. |
| `services` | `get` | Find target metrics Service. |
| `endpoints` | `get` | Confirm Service endpoint shape where needed. |

## Static Guardrail Plan

Custom Semgrep or repository-specific checks should cover:

- `kube-slint-no-direct-service-url-format`;
- `kube-slint-no-bearer-token-in-curl-args`;
- `kube-slint-no-insecure-skip-verify`;
- `kube-slint-no-clusterrolebinding-default`;
- `kube-slint-no-stat-before-write`;
- `kube-slint-no-unsafe-cleanup`.

Do not enable these as blocking CI until each rule has positive and negative
examples and the current codebase is compliant or explicitly exempted.

## Current CI-Guarded Items

The quality guardrail workflow currently checks:

- `SECURITY.md` matches current in-pod token handling;
- default RBAC does not reintroduce ClusterRoleBinding;
- redaction still covers Bearer and common secret names;
- curlpod securityContext remains non-privileged;
- ServiceURLFormat external-host handling remains a Priority 0 policy.

## Acceptance Checklist

- [ ] External metrics URL rejected by default.
- [ ] Token forwarding to external hosts impossible in default mode.
- [ ] Namespace-scoped RBAC remains default.
- [ ] Cluster-wide RBAC requires explicit dangerous opt-in.
- [ ] Privileged pod and hostPath are rejected in generated/default resources.
- [ ] Cleanup requires kube-slint ownership metadata.
- [ ] Invalid summary/policy cannot produce PASS.
