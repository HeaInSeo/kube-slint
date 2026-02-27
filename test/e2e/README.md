# Legacy E2E Tests

The `test/e2e_test.go` and `test/e2e_suite_test.go` files are legacy integration tests originally designed when `kube-slint` functioned as a standalone controller. Since `kube-slint` has transitioned into an instrumentation library, these tests are currently broken (as they expect a standalone deployment and metrics endpoints that no longer exist in this repository).

They have been strictly quarantined using the `//go:build legacy_e2e` build tag to prevent them from executing during standard `go test ./...` and CI pipelines. 

To run them locally (not recommended unless refactoring them), you must explicitly provide the tag:
`go test -tags legacy_e2e ./test/e2e/...`
