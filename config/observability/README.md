# Observability (Kustomize Remote)

This directory is the entrypoint for adding observability resources.

## Remote resource (recommended)

Use a pinned ref (tag or commit SHA) to ensure reproducible installs.
Commit SHA is recommended.

Example:

```yaml
resources:
- github.com/<ORG>/<OBS_REPO>//kustomize/overlays/default?ref=<PINNED_REF>
```

## Patches

Optional patches can be added in `patches/override.yaml` and enabled by
uncommenting the `patches:` section in `kustomization.yaml`.
