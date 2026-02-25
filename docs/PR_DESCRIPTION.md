# PR Description

## Background
The repository has been transitioned from a standalone Kubebuilder Operator to a Go library and Observability stack framework. However, the `Makefile` and `README.md` still contained references, targets, and instructions for building, running, and deploying an operator (e.g., `cmd/main.go`, `config/manager`). This caused downstream confusion and failed builds for developers attempting to standardly test the repository. 

Additionally, we needed to ensure the Kustomize entrypoints (specifically `config/observability` and `config/default`) are stable and capable of being consumed remotely by other projects.

## Changes
- **Makefile Cleanup:**
  - Standardized targets for library development (`build`, `test`, `fmt`, `vet`).
  - Converted obsolete deployment and image build targets (`run`, `docker-build`, `deploy`, `build-installer`, `install`, etc.) into `echo` no-op statements with friendly guidance pointing out that the repository is now a library.
  - Removed `setup-envtest` dependency from local tests since `pkg/` tests do not require a live API server (resolving 401 GCS issues).
- **Kustomize Entrypoint Tuning:**
  - Validated that `config/default` behaves as a safe overarching entrypoint for the observability stack.
  - Demonstrated usage of remote resource references (`github.com/HeaInSeo/kube-slint//config/default?ref=<tag or commitSHA>`).
- **README Updates:**
  - Explicitly stated the "operator runtime removed" transition.
  - Separated instructions into two sections: Deploying the Observability Stack (via Kustomize) vs. Instrumenting your Operator (via Go modules).
  - Clarified that Kustomize handles deployment targets, while `slint` acts as the computational engine.

## Verification
I ran the following commands to ensure stability:
- `make build` -> Success (compiles `pkg/` contents).
- `make test` -> Success (runs unit tests without ENVTEST failures).
- `make fmt` and `make vet` -> Success.
- `./bin/kustomize build config/default` -> Success (outputs correct configuration alongside placeholder ConfigMap, requiring consumer overlay for exact injection).

## Compatibility
Existing targets widely used by downstream systems or returning developers (`make run`, `make docker-build`) will now safely exit 0 while printing a deprecation notice, preventing sudden pipeline explosion errors. 

## TODO
- Downstream projects utilizing this Observability Stack must resolve and determine their `<PINNED_REF>` when pointing to `github.com/HeaInSeo/kube-slint//config/default`.
