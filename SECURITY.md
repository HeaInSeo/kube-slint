# Security Policy

## Reporting Vulnerabilities

Please report suspected vulnerabilities through a private GitHub security advisory for this repository, or contact the repository owner directly if GitHub advisory access is unavailable.

Do not open a public issue for secrets, token exposure, privilege escalation, or vulnerability details before a fix or mitigation is available.

## Supported Scope

Security support covers the current `main` branch and the latest public release tag.

## Token Handling

kube-slint can scrape an operator `/metrics` endpoint by creating a short-lived
curl pod. In the current curl pod path, the pod reads its own mounted
ServiceAccount token from
`/var/run/secrets/kubernetes.io/serviceaccount/token` and uses it only inside
the pod to call the metrics endpoint.

Operational implications:

- The bearer token should not appear in kubectl command arguments, generated
  PodSpec command strings, CI logs, or kube-slint command-bearing errors.
- Anyone who can exec into or otherwise inspect the running curl container may
  be able to access the mounted ServiceAccount token while the pod exists.
- The curl pod is intended to be short-lived and is labeled for cleanup, but cleanup is best-effort.

Recommended operation:

- Use a dedicated ServiceAccount for kube-slint scraping.
- Scope the ServiceAccount to the namespace and permissions needed for the
  measurement path.
- Run kube-slint measurement in isolated CI or test namespaces.
- Treat generated `sli-summary.json`, `slint-gate-summary.json`, pod logs, and CI logs as artifacts that may contain operational metadata.
- Do not commit generated gate summaries or raw logs unless they are intentionally scrubbed fixtures.

## Current Limitations

- The curl pod still receives a mounted ServiceAccount token, so namespace and
  RBAC scope remain important.
- The default RBAC scaffolding should still be reviewed before use in shared
  clusters.
- Measurement failure remains separate from correctness test failure by design; use `FAIL_OR_NOGRADE` or `FAIL_WARN_OR_NOGRADE` for promotion gates that must reject missing measurement.
