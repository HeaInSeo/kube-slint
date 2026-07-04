# Dangerous Option Naming Policy

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

Options that weaken kube-slint's security boundary must be difficult to enable
by accident. This document defines naming and documentation rules for those
options.

## Rule

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

The name should describe the exact risk, not the implementation detail.

Good:

- `dangerouslyAllowExternalMetricsURL`
- `dangerouslySkipTLSVerify`
- `dangerouslyAllowClusterWideRBAC`

Avoid:

- `allowExternal`
- `insecure`
- `skipVerify`
- `clusterMode`
- `unsafe`

## Documentation Requirements

Every dangerous option must document:

- The default safe behavior.
- The risk being accepted.
- The narrow use case where the option may be appropriate.
- Whether credentials, tokens, cluster resources, or CI secrets may be exposed.
- Which tests cover the default rejection and opt-in behavior.

## Runtime Reporting Requirements

When a dangerous option is enabled, kube-slint should make the choice visible
in a non-secret output path, such as:

- debug logs
- effective config summary
- gate diagnostics
- CI step summary

The output must not reveal tokens, Authorization headers, kubeconfig secrets,
or other credential material.

## Option Review Checklist

Before adding a dangerous option:

- [ ] The default behavior rejects or avoids the dangerous path.
- [ ] The option name starts with `dangerously`.
- [ ] The option name states the risk directly.
- [ ] There is a negative test for default rejection.
- [ ] There is a positive test for explicit opt-in.
- [ ] Documentation explains the security impact.
- [ ] CI or static guardrails prevent the default from regressing.

## Initial Dangerous Option Candidates

| Candidate | Purpose | Default |
|---|---|---|
| `dangerouslyAllowExternalMetricsURL` | Permit metrics URL outside cluster-local DNS. | false |
| `dangerouslySkipTLSVerify` | Permit TLS verification bypass. | false |
| `dangerouslyAllowClusterWideRBAC` | Permit generated cluster-wide binding. | false |
| `dangerouslyAllowKubeSystemNamespace` | Permit target namespace `kube-system`. | false |
| `dangerouslyAllowUnsafeCleanup` | Permit cleanup without kube-slint ownership labels. | false |

## Compatibility Note

Existing public fields may need compatibility handling. If a legacy option has
an insecure name, new work should:

- keep source compatibility where required;
- document the legacy option as deprecated or compatibility-only;
- introduce the dangerous name as the preferred contract;
- add tests proving the default behavior remains safe.
