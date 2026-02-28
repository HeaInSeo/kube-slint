# Kustomize Parameterization Options Matrix (UX Debt Stage D)
*Date: 2026-02-28*

## 1. Problem Decomposition (Stage D UX Debt)

In Phase 4-b (Kustomize Remote Consumer test), we proved that grabbing `kube-slint` monitoring definitions via direct GitHub remote fetches (`github.com/HeaInSeo/kube-slint//config/samples/prometheus?ref=...`) technically works (resources are applied).

However, it results in a **silent failure of observability**:
1. **Hardcoded Labels:** `config/samples/prometheus/monitor.yaml` hardcodes `app.kubernetes.io/name: kube-slint` both in metadata and `spec.selector.matchLabels`.
2. **Selector Mismatch:** External consumers building their own operators will have different labels (e.g., `app.kubernetes.io/name: their-operator`).
3. **Silent Failure:** The applied `ServiceMonitor` will successfully exist in the cluster but will fail to match any pods because it's looking for `kube-slint` instead of the consumer's pods, causing Prometheus to scrape nothing.
4. **UX Void:** Consumers are forced to guess how to override these nested fields, breaking the "drop-in" reusability promise.

## 2. Options Matrix

| Option | Approach | Pros | Cons | Viability / Remote UX | Recommended? |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1. Pure Kustomize Patch (Local Override)** | Keep base as-is, provide explicit docs/templates for `patchesStrategicMerge` in consumer repos. | Lowest effort, zero changes to upstream base files. | High cognitive load for consumer. They must maintain a verbose YAML patch for a remote resource. | Poor-to-Medium. | **Yes (Short-term MVP)** |
| **2. Kustomize `replacements` & `commonLabels`** | Use modern Kustomize parametrization features in base (`config/samples/`). | Simplifies consumer Kustomization file. Standardized behavior. | Kustomize `replacements` syntax is infamously clunky and hard to debug. | Medium. | No (Over-engineered for now) |
| **3. Sample Split (Template + Examples)** | Refactor `config/samples/` to isolate generic scaffolds and provide a localized consumer example directory showing the exact override. | Highly educational. Reduces guessing. Safe backward compatibility. | Requires restructuring `config/` slightly. | Good. | **Yes (Paired with Opt 1)** |
| **4. Helm Chart Transition** | Abandon Kustomize for remote distribution; provide a `helm install kube-slint-monitoring --set selector=foo`. | Industry standard for parametrization. Best UX. | Massive rewrite. Breaks existing Kustomize consumers. High maintenance. | Excellent, but heavy. | No (Long-term only) |
| **5. JSONNET / CUE** | Use advanced templating engine. | Ultimate flexibility. | Steep learning curve. Forces new tooling on users. | Poor. | Never |

## 3. Selected Short-Term MVP (P4-1)

**Decision: Option 1 & 3 Hybrid (Explicit Patching via Documentation & Consumer Example).**

Instead of diving into Kustomize's messy `replacements` feature or moving to Helm right away, the most robust Minimum Viable Parameterization (MVP) is to:
1. Treat the hardcoded labels in `config/samples/prometheus/monitor.yaml` as a base default, not a hard truth.
2. Update the integration test asset `test/consumer-onboarding/kustomize-remote-consumer` to actually perform a *Strategic Merge Patch* on the ServiceMonitor. This clearly documents to the consumer *where* and *how* to inject their target operator's name.
3. Prove that this small diff completely solves the silent failure without breaking backward compatibility or requiring structural overhauls of `kube-slint`'s own config.

## 4. Execution Plan (Small Diff)

1. **`test/consumer-onboarding/kustomize-remote-consumer/kustomization.yaml`**: Add a `patches` block pointing to a new `override-monitor.yaml`.
2. **`override-monitor.yaml`**: Construct the minimal JSON/YAML patch to overwrite `spec.selector.matchLabels`.
3. Ensure this compiles beautifully with `kubectl kustomize`.
