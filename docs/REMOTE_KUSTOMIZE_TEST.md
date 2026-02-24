# Remote Kustomize Test

To ensure that the `kube-slint` repository works as a remote Kustomize resource, consumers can simply reference the `config/default` directory using a pinned tag or commit SHA.

## How to Test Remote Usage

You can test this functionality locally using `kustomize build` by pointing to the GitHub repository directly.

> **CRITICAL:** Do NOT use branches like `?ref=main`. You must pin the remote resource to a specific tag or commit SHA to ensure reproducible builds.
> Additionally, because the base configuration (`config/default`) purposely omits hardcoded namespaces (Strategy A), consumers should use a Kustomization overlay with the `namespace:` field declared to map the resources into their desired target namespace.

```bash
# Required approach: Pinned to a specific commit SHA or Tag:
kustomize build github.com/HeaInSeo/kube-slint//config/default?ref=ca156d34b0efde18bb54fcf1e9d07727e5e4dce3
```

This ensures consumers can deploy the observability stack seamlessly without needing to clone the operator repository locally.
