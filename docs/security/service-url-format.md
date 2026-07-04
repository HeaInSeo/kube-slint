# ServiceURLFormat Security Policy

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

`ServiceURLFormat` controls the metrics URL used by kube-slint when scraping
`/metrics`. If this format points outside the cluster-local trust boundary,
the curl pod can send Authorization material to an attacker-controlled host.

This policy defines the target default behavior for ServiceURLFormat
validation.

## Trust Boundary

Default mode treats these as trusted for scraping:

- service DNS names ending in `.svc`
- service DNS names ending in `.svc.cluster.local`

Default mode treats these as untrusted:

- public DNS names
- private but non-cluster-local DNS names
- raw IP addresses
- URLs whose host is assembled so that service or namespace appears under an
  external suffix
- schemes other than `http` or `https`

## Allowed Default Shapes

Allowed examples:

```text
https://<service>.<namespace>.svc:8443/metrics
https://<service>.<namespace>.svc.cluster.local:8443/metrics
http://<service>.<namespace>.svc:<port>/metrics
http://<service>.<namespace>.svc.cluster.local:<port>/metrics
```

`https` is preferred. `http` may be acceptable for in-cluster metrics when the
cluster network is the intended trust boundary and the ServiceAccount scope is
minimal. This remains an explicit open decision for implementation.

## Rejected Default Shapes

Rejected examples:

```text
https://evil.example.com/collect?svc=%s&ns=%s
https://%s.%s.evil.com/metrics
ftp://%s.%s.svc/metrics
https://10.0.0.10/metrics
https://%s-%s.default.svc.evil.example/metrics
```

## Validation Requirements

Before scraping:

- parse the formatted URL with a structured URL parser;
- reject unsupported schemes;
- validate service and namespace interpolation values before URL construction;
- reject empty service or namespace values;
- require DNS-label-compatible service and namespace names;
- require the final host to be cluster-local unless a dangerous opt-in is set;
- reject external hosts before creating the curl pod;
- never send Authorization material to an external host in default mode.

## Dangerous Opt-In

If external metrics URLs are supported, the option must be named:

```yaml
dangerouslyAllowExternalMetricsURL: true
```

Open implementation decision:

- External URL with Authorization header removed.
- External URL fully rejected unless a dangerous opt-in is set.
- External URL allowed only for unauthenticated scrape mode.

Default recommendation:

External URL should remain rejected until token and Authorization behavior is
specified and tested.

## Test Matrix

| Case | Expected default result |
|---|---|
| `https://svc.ns.svc:8443/metrics` | accept |
| `https://svc.ns.svc.cluster.local:8443/metrics` | accept |
| `http://svc.ns.svc:8080/metrics` | accept or documented reject |
| `https://evil.example.com/collect?svc=%s&ns=%s` | reject |
| `https://%s.%s.evil.com/metrics` | reject |
| `ftp://%s.%s.svc/metrics` | reject |
| service name contains `/` | reject |
| namespace name contains `..` | reject |
| missing service | reject |
| missing namespace | reject |

## Acceptance Criteria

- [ ] External host is rejected by default.
- [ ] Without a dangerous option, token or Authorization material cannot be
  sent outside cluster-local DNS.
- [ ] Malformed service and namespace values are rejected.
- [ ] URL validator test cases are documented and implemented.
- [ ] User-facing error explains the security boundary and suggested fix.

## Developer Ticket

Use the `Default-deny external ServiceURLFormat` ticket in
`docs/quality-roadmap-ticket-backlog.md`.
