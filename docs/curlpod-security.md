# curlpod Security Model

kube-slint uses a short-lived curl Pod to scrape `/metrics` from inside the cluster.
This document lists required RBAC, recommended NetworkPolicy, and Pod identity labels.

## Pod identity labels

Every curlpod carries two fixed labels set by the harness:

| Label | Value | Purpose |
|---|---|---|
| `app.kubernetes.io/managed-by` | `kube-slint` | Identifies all pods owned by kube-slint |
| `slint-run-id` | `<RunID>` | Ties the pod to a specific measurement session |

These labels enable `CleanupByLabel` to delete orphaned pods from previous runs and make pods auditable in `kubectl get pods -l app.kubernetes.io/managed-by=kube-slint`.

## Minimum RBAC

The ServiceAccount used by the curlpod needs only `get` on the target Service's endpoint.
The harness itself (running outside the cluster) needs permission to create/delete Pods and read their logs.

### ServiceAccount for the curlpod

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: slint-metrics-reader
  namespace: <operator-namespace>
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: slint-metrics-reader
  namespace: <operator-namespace>
rules:
  - apiGroups: [""]
    resources: ["services", "endpoints"]
    verbs: ["get", "list"]
```

### ServiceAccount for the harness (CI runner)

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: slint-harness
  namespace: <operator-namespace>
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["create", "delete", "get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
```

## NetworkPolicy example

Allow the curlpod to reach the operator's metrics port and block all other egress:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: slint-curlpod-egress
  namespace: <operator-namespace>
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: kube-slint
  policyTypes:
    - Egress
  egress:
    - ports:
        - port: 8443   # operator metrics port (adjust as needed)
          protocol: TCP
      to:
        - podSelector:
            matchLabels:
              app: <operator-app-label>
```

## Cleanup guarantee

`CurlPod.Run()` always calls `DeletePodNoWait` after collecting logs, even on error paths.
If deletion fails (e.g. network partition, RBAC revoked mid-run), the failure is logged as a warning:

```
kube-slint [curlpod]: cleanup warning — failed to delete pod <ns>/<name>: <err>
 (pod may require manual cleanup; selector: app.kubernetes.io/managed-by=kube-slint,slint-run-id=<id>)
```

Use the printed label selector to find and delete orphaned pods manually:

```bash
kubectl delete pod -n <namespace> -l app.kubernetes.io/managed-by=kube-slint
```
