# Contributing to FUSE

Thank you for your interest in contributing to FUSE (Utility for Stateful Events)! This document provides guidelines and instructions for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make
- Git

### Setting Up Development Environment

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/fuse.git`
3. Set up upstream remote: `git remote add upstream https://github.com/open-source-cloud/fuse.git`
4. Create a new branch for your changes: `git checkout -b feature/your-feature-name`

## Development Workflow

### Building the Project

```bash
make build
```

### Running Tests

```bash
make test
```

### Linting

The project uses golangci-lint for code quality checks:

```bash
make lint
```

To automatically fix linting issues where possible:

```bash
make lint-fix
```

## Code Conventions

### Project Structure

- `cmd/`: Application entry points
- `pkg/`: Public libraries that can be imported by external projects
- `internal/`: Private application code
- `tests/`: Integration tests
- `bin/`: Build artifacts (not committed to git)
- `docs/`: Documentation
- `examples/`: Example usage and configurations

### Coding Style

- Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go)
- Use meaningful variable and function names
- Write comprehensive comments for public APIs
- Keep functions focused and small
- Use interfaces to enable modularity

### Commit Messages

- Use the imperative mood ("Add feature" not "Added feature")
- Keep the first line under 50 characters
- Reference issues and pull requests where appropriate
- Consider using conventional commits format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `style:` for formatting changes
  - `refactor:` for code refactoring
  - `test:` for adding or modifying tests
  - `chore:` for maintenance tasks

## Submitting Changes

1. Make sure your code passes all tests and linting
2. Update documentation if necessary
3. Push your changes to your fork
4. Submit a pull request to the main repository
5. Ensure the CI/CD pipeline passes

## Pull Request Process

1. Update the README.md or relevant documentation with details of your changes
2. Add or update tests for any new functionality
3. Ensure your PR passes all CI checks
4. Wait for review and address any feedback

## Definitions

- **Node**: A single step in a workflow that can execute logic and validate its configuration
- **NodeProvider**: Component responsible for creating and managing nodes
- **Workflow**: Complete automation definition containing nodes and edges
- **Edge**: Connection between nodes, optionally with conditional logic
- **WorkflowEngine**: Component that orchestrates workflow execution
- **Schema**: Definition of structure for workflows, enabling validation

## License

By contributing to FUSE, you agree that your contributions will be licensed under the project's license.

## Questions and Support

If you have questions or need support, please open an issue on GitHub or reach out to the maintainers.

Thank you for contributing to FUSE!
