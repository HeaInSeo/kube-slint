# Remote Kustomize Test

To ensure that the `kube-slint` repository works as a remote Kustomize resource, consumers can simply reference the `config/default` directory using a pinned tag or commit SHA.

## How to Test Remote Usage

You can test this functionality locally using `kustomize build` by pointing to the GitHub repository directly:

```bash
# Example using main branch:
kustomize build github.com/HeaInSeo/kube-slint//config/default?ref=main

# Recommended approach: Pinned to a specific commit SHA or Tag:
# kustomize build github.com/HeaInSeo/kube-slint//config/default?ref=<COMMIT_SHA>
```

This ensures consumers can deploy the observability stack seamlessly without needing to clone the operator repository locally.
