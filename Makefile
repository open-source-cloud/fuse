.PHONY: run run-debug install-gotestsum test test-report testdox clean install-lint lint lint-fix swagger build build-debug

GOTESTSUM := $(shell go env GOPATH)/bin/gotestsum
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

run:
	go build -o bin/fuse cmd/fuse/main.go
	./bin/fuse server -o -p 9090 -l debug

# Install gotestsum into GOPATH/bin (same path as GOTESTSUM)
install-gotestsum:
	go install gotest.tools/gotestsum@latest

test: install-gotestsum
	$(GOTESTSUM) --junitfile test-report.xml --format testdox -- ./pkg/... ./internal/...

test-benchmark:
	go test -bench=. -benchmem ./...

build:
	go build -o bin/fuse cmd/fuse/main.go

build-debug:
	go build -tags pprof -o bin/fuse cmd/fuse/main.go

run-debug: build-debug
	./bin/fuse server -o -p 9090 -l debug

clean:
	rm -rf bin/

# Install golangci-lint (built with the active Go toolchain; required when go.mod > golangci-lint's embed Go)
install-lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

# Run linter
lint: install-lint
	$(GOLANGCI_LINT) run ./... --timeout=5m

# Run linter with auto-fix
lint-fix: install-lint
	$(GOLANGCI_LINT) run ./... --fix --timeout=5m

format:
	go fmt ./...

# Generate Swagger documentation
swagger:
	swag init -g cmd/fuse/main.go -o docs/

dkb:
	docker build -t fuse-app:dev .

dkx:
	docker stop fuse-local
	docker rm fuse-local
	docker run --name fuse-local --env-file .env -p 9090:9090 fuse-app:dev
