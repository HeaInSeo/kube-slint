# RBAC Model

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

kube-slint should not need cluster-wide Kubernetes permissions for normal
measurement and gate workflows. This document defines the default RBAC model.

## Default Model

Default generated RBAC uses:

- `ServiceAccount`
- namespaced `Role`
- namespaced `RoleBinding`

Default generated RBAC must not use:

- `ClusterRole`
- `ClusterRoleBinding`

## Required Permissions

The harness path needs only the permissions required to create and observe the
temporary curl pod and read the target Service/Endpoints.

Expected namespace-scoped permissions:

| Resource | Verbs | Purpose |
|---|---|---|
| `pods` | `get`, `list`, `create`, `delete` | Create and clean up curl pod. |
| `pods/log` | `get` | Read scrape result from curl pod logs. |
| `services` | `get` | Find target metrics Service. |
| `endpoints` | `get` | Confirm Service endpoint shape where needed. |

## Dangerous Cluster-Wide Opt-In

If cluster-wide RBAC is ever generated, it must require an explicit dangerous
option:

```yaml
dangerouslyAllowClusterWideRBAC: true
```

Default docs and examples must not imply that ClusterRoleBinding is the normal
setup path.

## Namespace Restrictions

Default behavior should reject or warn before running against high-risk
namespaces such as:

- `kube-system`
- `kube-public`
- `kube-node-lease`

If supported, overriding this must require a dangerous option such as:

```yaml
dangerouslyAllowKubeSystemNamespace: true
```

## Acceptance Criteria

- [ ] `slint-gate init --emit-rbac` emits ServiceAccount, Role, and
  RoleBinding by default.
- [ ] Default RBAC output does not include ClusterRoleBinding.
- [ ] Tests prevent default ClusterRoleBinding regression.
- [ ] README and security docs describe namespace-scoped RBAC as default.
- [ ] Any cluster-wide path is documented as advanced and dangerous.

## Related Documents

- `docs/curlpod-security.md`
- `docs/security-defaults.md`
- `docs/dangerous-options.md`
