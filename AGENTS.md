## Learned User Preferences

- Prefer dev tooling to be installable from the Makefile so `make test` and `make lint` work on fresh machines without manual binary setup.

## Learned Workspace Facts

- The module targets Go 1.26; local toolchains, CI workflows, and the Dockerfile should match.
- golangci-lint must be built with a Go version at least as new as `go.mod`; use `make install-lint` (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0`). Prebuilt install scripts can embed an older Go and fail with a version-mismatch error when loading `.golangci.yml`.
- `make test` runs `install-gotestsum` first (`go install gotest.tools/gotestsum@latest` into `GOPATH/bin`).
- With ergo v3.2+, sending to sibling processes during actor `Init` is supported; workflow init runs inline in `WorkflowHandler.Init` instead of an `ActorInit` / `actor:init` self-message.
- Async completion from goroutines outside `HandleMessage` (e.g. timer callbacks) should deliver messages via `Node().Send`, not `Process.Send`, when the worker can be in Sleep (ergo restricts `Process.Send` to Init/Running/Terminated).
- `make examples-ci` and `scripts/run-example-workflows.sh` use `CI=true` to skip `github-request-example` and workflows that reference `fuse/pkg/logic/timer` where the default server path does not complete async timer flows in CI.
- Multiple files under `examples/workflows/` reuse the same schema `id` (e.g. `small-test`); upserting them in order means the last file wins for that id.
- `GET /v1/workflows/{workflowID}/status` returns workflow instance state and audit data from the in-memory workflow repository; an optional `logs` field appears when the server log level is debug.
- `MemoryPackageRepository` uses a mutex because internal package registration saves packages from concurrent goroutines.
- `make dockerfile-lint` runs Hadolint in Docker on `Dockerfile` and `Dockerfile.dev` with `.hadolint.yaml`; it catches consecutive-`RUN` patterns that align with SonarCloud Docker rules (e.g. docker:S7031). `make sonar-local` uses `sonar-project.properties` and Dockerized SonarScanner; set `SONAR_TOKEN` from SonarCloud (analysis-capable token, not a GitHub token). It passes the current git branch and commit; if analysis fails with HTTP 404 on `analysis/analyses`, try `SONAR_REGION=us` for US-hosted SonarCloud orgs.
- `Dockerfile.dev` runs as non-root `fuse` (UID/GID 1000): module files stay root-owned and readable, `/go` is writable for caches and installs. On Linux, bind-mounted source must suit UID 1000 or use Compose `user` to match the host.
