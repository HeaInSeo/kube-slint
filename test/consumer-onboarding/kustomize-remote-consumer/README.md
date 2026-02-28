# Kustomize Direct Remote Consumption (Explicit Local Override Tutorial)

## Context
You can consume `kube-slint`'s observability layer directly via Kustomize remote resource fetching (`github.com/HeaInSeo/kube-slint//config/samples/prometheus?ref=<SHA>`).

However, the upstream `ServiceMonitor` contains a **hardcoded label** (`app.kubernetes.io/name: kube-slint`). If you simply fetch it as a drop-in without overrides, Prometheus will experience a **Silent Failure** — it won't scrape your operator because the labels won't match. 

To solve this, you must apply an **Explicit Local Override (Kustomize Patch)**.
*(Note: This is a short-term UX mitigation. Long-term structural parameterization of the base manifests is currently deferred.)*

## Tutorial: How to Patch for your Operator

You need two files in your deployment Kustomize directory.

### 1. The Patch File (`patch-monitor.yaml`)
Create this file to inject your operator's specific name into the `matchLabels` selector.

**Before (Upstream Base):** `app.kubernetes.io/name: kube-slint`  
**After (Your Overridden Target):** `app.kubernetes.io/name: my-custom-operator`

```yaml
# patch-monitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: controller-manager-metrics-monitor
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: my-custom-operator
spec:
  selector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: my-custom-operator
```

### 2. The `kustomization.yaml`
Import the remote resource and explicitly apply your patch file using the `patches:` block.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: your-target-namespace

resources:
# IMPORTANT: ALWAYS pin a specific SHA or tag. Do not use ?ref=main
- github.com/HeaInSeo/kube-slint//config/samples/prometheus?ref=0f48feb356823dfa12cef8f0500236983b291953

patches:
- path: patch-monitor.yaml
```

### 3. Verification
Render the manifests to confirm your `my-custom-operator` label has successfully mutated the upstream `ServiceMonitor`:

```bash
kubectl kustomize .
```
