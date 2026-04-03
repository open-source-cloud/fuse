// Package e2e contains HTTP integration tests (build tag e2e), one test file per
// example workflow under examples/workflows, REST smoke tests in apis_e2e_test.go,
// and helpers for the FUSE API.
//
// Run API-backed tests: go test -tags=e2e ./tests/e2e -v
//
// Run helper unit tests only: go test ./tests/e2e
package e2e
