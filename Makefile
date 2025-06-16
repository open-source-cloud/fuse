.PHONY: run test test-report testdox clean lint lint-fix

GOTESTSUM := $(shell go env GOPATH)/bin/gotestsum
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

run:
	go build -o bin/fuse cmd/fuse/main.go
	./bin/fuse server -o -p 9090 -l debug

test:
	$(GOTESTSUM) --junitfile test-report.xml --format testdox -- ./pkg/... ./internal/...

test-benchmark:
	go test -bench=. -benchmem ./...

build:
	go build -o bin/fuse cmd/fuse/main.go

clean:
	rm -rf bin/

# Install golangci-lint if not installed
install-lint:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.0.2

# Run linter
lint: install-lint
	$(GOLANGCI_LINT) run ./... --timeout=5m

# Run linter with auto-fix
lint-fix: install-lint
	$(GOLANGCI_LINT) run ./... --fix --timeout=5m

