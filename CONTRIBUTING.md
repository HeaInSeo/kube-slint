# Contributing to kube-slint

Thank you for your interest in contributing!

## Getting started

```bash
git clone https://github.com/HeaInSeo/kube-slint.git
cd kube-slint
go mod download
go test ./...
```

Requirements: Go 1.25+, `make`, `jq` (for action smoke tests).

## Development workflow

1. Fork the repository and create a branch: `git checkout -b feat/my-change`
2. Make your changes, add or update tests.
3. Run `go test ./...` — all tests must pass.
4. Run `go vet ./...` and `go build ./...` — no errors.
5. Open a pull request against `main`.

## What to contribute

- Bug fixes — include a failing test that reproduces the bug.
- New SLI compute modes — add to `pkg/slo/engine` and `pkg/slo/spec`.
- New fetcher backends — implement `fetch.Fetcher` in `pkg/slo/fetch/`.
- Documentation or example improvements.

## Code style

- Standard Go formatting (`gofmt`).
- No comments explaining *what* code does — only *why* a non-obvious choice was made.
- No feature flags or backward-compat shims; just change the code.

## Tests

- Unit tests live alongside the package they test (`_test.go`).
- Example/consumer spec files use `//go:build ignore` to stay out of `go test ./...`.
- Integration tests that require a live cluster are tagged `//go:build e2e`.

## Reporting issues

Use the GitHub issue templates. Please include:
- kube-slint version or commit SHA
- Go version (`go version`)
- Minimal reproduction steps or a failing test

## License

By contributing you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).
