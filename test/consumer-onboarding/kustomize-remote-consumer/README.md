# Kustomize Consumer UX Probe (Phase 4-b)

## Purpose
This directory contains a minimal validation asset to verify the User Experience (UX) of an external operator consuming `kube-slint`'s observability stack via Kustomize Remote Resources.

## Validated Paths
1. `github.com/HeaInSeo/kube-slint//config/default?ref=<SHA>`
2. `github.com/HeaInSeo/kube-slint//config/samples/prometheus?ref=<SHA>`

## Kustomize UX Findings (4 Categories)

### 1. Document UX
The main `README.md` correctly identifies the need for `//` (double slash) to separate the repo URL from the subdirectory, and explicitly warns against using `?ref=main` to enforce reproducible pinning. However, the README then instructs the user to "copy/adapt" files from `config/samples/` because `config/default` is an empty placeholder. This creates cognitive dissonance: why use Remote Kustomize if the consumer has to manually copy the files locally anyway?

### 2. Ref Pinning & Path Validity
The exact string `github.com/HeaInSeo/kube-slint//config/default?ref=...` works flawlessly at the tool level (`kubectl kustomize`). Kustomize correctly clones the repo at that SHA and injects the overlay's `namespace`.

### 3. Asset Placement / Structural Issues
Because this project pivoted from a Standalone Operator to a Library, the `config/` directory is in an awkward transitional state.
- `config/default` only yields a `kube-slint-observability-placeholder` ConfigMap.
- `config/samples/prometheus` yields a valid `ServiceMonitor`, but it is hardcoded to `app.kubernetes.io/name: kube-slint`.
If a consumer imports this remotely, their Prometheus will scrape nothing because their operator's label is different. To be a true "Remote Base", Kustomize components need to be parameterized (e.g., using vars, nameReferences, or Helm charts) rather than hardcoded.

### 4. Debugging & Error UX (P4-1 MVP UX Fix Applied)
If a novice user ignores the text and blindly runs `kustomize build` on the remote resource, they will not get any build errors. The manifests will deploy successfully across the cluster, but no metrics will be scraped. The failure is silent (at the infrastructure level) due to label mismatching.

**P4-1 Solution (Mock Drop-In UX):**
To mitigate this UX debt without completely rebuilding `config/samples/` into a Helm chart, we enforce the **Option 1 (Explicit Local Override)** strategy in this consumer testing harness.
By specifying `patch-monitor.yaml` via standard Kustomize `patchesStrategicMerge`, the consumer overrides the hardcoded `app.kubernetes.io/name` labels with their own operator labels (`my-custom-operator`). This acts as the official templated pattern for parameterization.

## Conclusion
The technical mechanism of Kustomize Remote Resources works. The silent failure caused by structural hardcoding is now explicitly circumvented via provided override patterns. In the medium-to-long term, more advanced `replacements` or `helm` strategies may be adopted, but this MVP guarantees functionally sound consumption today.
