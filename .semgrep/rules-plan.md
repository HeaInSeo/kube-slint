# kube-slint Semgrep / Custom Rule Plan

Date: 2026-07-04
Status: Draft plan; not yet CI-enforced

## Purpose

Generic Go linters do not know kube-slint's security boundary. This plan
defines custom static checks that can later be implemented with Semgrep or a
small repository-specific scanner.

Do not enable these rules as blocking CI until each rule has positive and
negative examples and the current codebase is either compliant or explicitly
exempted.

## Rule: kube-slint-no-direct-service-url-format

Purpose:

Prevent direct construction of metrics URLs from `ServiceURLFormat` without a
validator.

Risk:

An external URL can receive Authorization material.

Positive example:

```go
metricsURL := fmt.Sprintf(c.ServiceURLFormat, metricsSvcName, ns)
```

Negative example:

```go
metricsURL, err := serviceurl.BuildValidated(c.ServiceURLFormat, metricsSvcName, ns)
if err != nil {
    return "", err
}
```

Initial action:

- Warn only.
- Convert to blocking after ServiceURLFormat validator exists.

## Rule: kube-slint-no-bearer-token-in-curl-args

Purpose:

Prevent literal token interpolation into command arguments or PodSpec command
strings.

Risk:

Token appears in kubectl args, PodSpec, process list, logs, or CI errors.

Positive example:

```go
fmt.Sprintf(`curl -H "Authorization: Bearer %s" "%s"`, token, metricsURL)
```

Negative example:

```go
`TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)";
curl -H "Authorization: Bearer ${TOKEN}" "${METRICS_URL}";`
```

Initial action:

- Blocking after examples are added.

## Rule: kube-slint-no-insecure-skip-verify

Purpose:

Prevent insecure TLS defaults in production/default paths.

Risk:

Metrics scraping can be downgraded or intercepted.

Positive example:

```go
TLSInsecureSkipVerify: true
```

Negative example:

```go
DangerouslySkipTLSVerify: cfg.DangerouslySkipTLSVerify
```

Initial action:

- Warn only because current compatibility paths may still expose legacy fields.

## Rule: kube-slint-no-clusterrolebinding-default

Purpose:

Prevent default generated RBAC from returning to cluster-wide binding.

Risk:

Normal measurement path gets unnecessary cluster-wide privileges.

Positive example:

```yaml
kind: ClusterRoleBinding
```

Negative example:

```yaml
kind: RoleBinding
```

Initial action:

- Blocking for default scaffolding paths.

## Rule: kube-slint-no-stat-before-write

Purpose:

Detect race-prone `os.Stat` before file creation patterns.

Risk:

TOCTOU behavior can overwrite or race generated policy/artifact files.

Positive example:

```go
if _, err := os.Stat(path); os.IsNotExist(err) {
    return os.WriteFile(path, data, 0644)
}
```

Negative example:

```go
f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
```

Initial action:

- Warn only.

## Rule: kube-slint-no-unsafe-cleanup

Purpose:

Prevent cleanup/delete operations without kube-slint ownership selectors.

Risk:

kube-slint deletes resources it did not create.

Positive example:

```go
kubectl delete pod -n ns --all
```

Negative example:

```go
kubectl delete pod -n ns -l app.kubernetes.io/managed-by=kube-slint,slint-run-id=runID
```

Initial action:

- Blocking after cleanup ownership policy is finalized.

## Implementation Checklist

- [ ] Add positive and negative fixture snippets for each rule.
- [ ] Decide Semgrep versus custom shell/Go scanner.
- [ ] Run rules in advisory mode first.
- [ ] Document all intentional exceptions.
- [ ] Promote low-noise rules to CI blocking.
