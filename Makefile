.PHONY: run run-debug install-gotestsum test test-report testdox clean install-lint lint lint-fix swagger build build-debug dockerfile-lint sonar-local

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

# Lint Dockerfiles (consecutive RUN merge, etc.). Mirrors Sonar Docker rules; requires Docker.
HADOLINT_IMAGE := hadolint/hadolint:2.12.0-alpine
dockerfile-lint:
	docker run --rm -v "$$(pwd):/work" -w /work --entrypoint /bin/hadolint $(HADOLINT_IMAGE) \
		--config /work/.hadolint.yaml Dockerfile Dockerfile.dev

# Full SonarCloud analysis (same project as GitHub checks). Requires SONAR_TOKEN from
# https://sonarcloud.io/account/security and Docker.
sonar-local:
	@test -n "$$SONAR_TOKEN" || (echo "Set SONAR_TOKEN (SonarCloud > My Account > Security)" >&2; exit 1)
	docker run --rm \
		-e SONAR_TOKEN \
		-v "$$(pwd):/usr/src" \
		sonarsource/sonar-scanner-cli:latest \
		-Dsonar.organization=open-source-cloud \
		-Dsonar.projectKey=open-source-cloud_fuse \
		-Dsonar.sources=. \
		-Dsonar.exclusions=**/vendor/**,**/bin/**,**/.git/** \
		-Dsonar.host.url=https://sonarcloud.io

dkb:
	docker build -t fuse-app:dev .

dkx:
	docker stop fuse-local
	docker rm fuse-local
	docker run --name fuse-local --env-file .env -p 9090:9090 fuse-app:dev
