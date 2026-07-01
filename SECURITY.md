# Security Policy

## Reporting Vulnerabilities

Please report suspected vulnerabilities through a private GitHub security advisory for this repository, or contact the repository owner directly if GitHub advisory access is unavailable.

Do not open a public issue for secrets, token exposure, privilege escalation, or vulnerability details before a fix or mitigation is available.

## Supported Scope

Security support covers the current `main` branch and the latest public release tag.

## Token Handling

kube-slint can scrape an operator `/metrics` endpoint by creating a short-lived curl pod. In the current curl pod path, the bearer token is passed to curl as an `Authorization: Bearer ...` header in the pod command.

Operational implications:

- Anyone who can read the generated curl pod spec may be able to see the token while the pod exists.
- Anyone who can read process arguments inside the curl container may be able to see the token while curl is running.
- The curl pod is intended to be short-lived and is labeled for cleanup, but cleanup is best-effort.

Recommended operation:

- Use short-lived ServiceAccount tokens, for example tokens created with `kubectl create token ... --duration=1h` or a shorter duration appropriate for CI.
- Scope the ServiceAccount to the namespace and permissions needed for the measurement path.
- Run kube-slint measurement in isolated CI or test namespaces.
- Treat generated `sli-summary.json`, `slint-gate-summary.json`, pod logs, and CI logs as artifacts that may contain operational metadata.
- Do not commit generated gate summaries or raw logs unless they are intentionally scrubbed fixtures.

## Current Limitations

- The curl pod token is currently command-line visible inside the generated pod spec.
- The default RBAC scaffolding should be reviewed before use in shared clusters.
- Measurement failure remains separate from correctness test failure by design; use `FAIL_OR_NOGRADE` or `FAIL_WARN_OR_NOGRADE` for promotion gates that must reject missing measurement.
