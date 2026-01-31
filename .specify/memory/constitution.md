# FUSE Constitution

## Core Principles

### I. Test-First Development (NON-NEGOTIABLE)

Every feature MUST have unit tests written before implementation. The TDD cycle is mandatory: Write tests → Tests fail → Implement → Tests pass → Refactor. This ensures code quality, prevents regressions, and documents expected behavior.

**Coverage Requirements:**
- Critical business logic: 90%+ coverage
- Repository implementations: 80%+ coverage
- HTTP handlers: 70%+ coverage
- Actor message handling: 80%+ coverage

**Enforcement:**
- All tests MUST pass before merge (`make test`)
- No code merges without passing test suite
- Tests are written alongside code, not after
- Table-driven tests for multiple scenarios
- Mock external dependencies, test real logic

### II. Code Quality Gates (NON-NEGOTIABLE)

All code MUST pass quality gates before commit and merge. The mandatory workflow is: Lint → Build → Test. No exceptions.

**Quality Gates:**
- **Linting**: All code MUST pass `make lint` before commit
- **Build**: Build MUST succeed (`make build`) before merge
- **Testing**: All tests MUST pass (`make test`) before merge
- **Coverage**: Coverage thresholds MUST be met per layer

**Enforcement:**
- Pre-commit hooks enforce quality gates
- CI/CD pipeline validates all gates
- No merge without passing all gates
- Clear error messages guide fixes

### III. Go Best Practices

Follow Effective Go guidelines and project conventions. Use interfaces for modularity and testability. Implement proper error handling with wrapped errors. Ensure thread-safe implementations where needed.

**Standards:**
- Follow Effective Go idioms and patterns
- Use meaningful variable and function names
- Write comprehensive comments for public APIs
- Keep functions focused and small
- Use interfaces to enable modularity
- Proper error handling with `fmt.Errorf("context: %w", err)`
- Thread-safe implementations with sync primitives

### IV. Actor Model Architecture

All workflow nodes operate as independent actors. Communication occurs through message passing. Supervisor patterns provide fault tolerance. Actor isolation ensures state management.

**Principles:**
- Each workflow node operates as an independent ergo actor
- Message passing for all inter-actor communication
- Supervisor patterns for fault tolerance and recovery
- Actor isolation and state management
- No shared mutable state between actors
- Let supervisors handle failures

### V. Domain-Driven Design

Clear domain boundaries and bounded contexts. Repository pattern for data access. Service layer for business logic orchestration. Value objects and entities properly modeled.

**Patterns:**
- Clear domain boundaries and bounded contexts
- Repository pattern for data access (interface-first design)
- Service layer for business logic orchestration
- Value objects and entities properly modeled
- Ubiquitous language in domain code
- Domain events for cross-boundary communication

### VI. Clean Architecture & Hexagonal Architecture

Dependency inversion: inner layers don't depend on outer layers. Ports and adapters pattern. Business logic isolated from infrastructure. Testable through interfaces.

**Structure:**
- Dependency inversion principle strictly followed
- Ports and adapters pattern for external dependencies
- Business logic isolated from infrastructure
- Testable through interfaces (dependency injection with fx)
- Layer separation: domain, application, infrastructure
- Interface-based design for all external dependencies

### VII. Microservices Principles

Service independence and autonomy. API-first design. Event-driven communication where appropriate. CQRS patterns for read/write separation.

**Guidelines:**
- Service independence and autonomy
- API-first design (REST, gRPC)
- Event-driven communication where appropriate
- CQRS patterns for read/write separation
- Service boundaries clearly defined
- Fault tolerance and resilience built-in

### VIII. Concurrency & Parallelism

Proper use of Go channels for communication. Actor model for concurrent workflows. Avoid shared mutable state. Use sync primitives correctly (RWMutex for read-heavy workloads).

**Best Practices:**
- Understand concurrency vs parallelism
- Proper use of Go channels for communication
- Actor model for concurrent workflows
- Avoid shared mutable state
- Use sync primitives correctly (RWMutex for read-heavy)
- Context for cancellation and timeouts
- Worker pools for parallel processing
- Race condition detection and prevention

## Development Workflow

### Test-First Workflow

1. **Write Tests First**: Define test cases before implementation
2. **Tests Fail**: Verify tests fail (red phase)
3. **Implement**: Write minimal code to make tests pass
4. **Tests Pass**: Verify all tests pass (green phase)
5. **Refactor**: Improve code while keeping tests green
6. **Quality Gates**: Run lint → build → test before commit

### Quality Gate Enforcement

Before every commit:
1. Run `make lint` - Fix all linting issues
2. Run `make build` - Ensure build succeeds
3. Run `make test` - Ensure all tests pass
4. Check coverage thresholds

Before every merge:
- All quality gates must pass
- All tests must pass
- Coverage thresholds must be met
- Code review must verify compliance

## Architecture Guidelines

### Actor Model Implementation

- Use ergo.services actor framework
- Follow actor patterns from `.cursor/rules/03-actor-patterns.mdc`
- Implement proper supervisor strategies
- Use message passing for all communication
- Maintain actor isolation

### Domain-Driven Design

- Model domain entities and value objects
- Define bounded contexts clearly
- Use repository pattern for data access
- Implement domain services for complex logic
- Use domain events for cross-boundary communication

### Clean Architecture Layers

- **Domain Layer**: Core business logic, entities, value objects
- **Application Layer**: Use cases, services, orchestration
- **Infrastructure Layer**: Repositories, external services, adapters
- **Presentation Layer**: HTTP handlers, CLI, actors

### Microservices Communication

- REST APIs for synchronous communication
- Event-driven patterns for asynchronous communication
- CQRS for read/write separation
- Service mesh for inter-service communication (future)

## Quality Standards

### Code Quality

- Follow `.golangci.yml` linting rules
- Cyclomatic complexity under 15
- No hardcoded values (use config)
- Proper error handling
- Thread-safe where needed

### Testing Standards

- Unit tests for all business logic
- Integration tests for repositories
- Actor-specific tests for message handling
- HTTP handler tests for endpoints
- Table-driven tests for multiple scenarios
- Test helpers to reduce boilerplate

### Documentation Standards

- Doc comments on all exported types and functions
- Detailed documentation for complex logic
- Package-level documentation
- Examples in documentation for non-trivial APIs
- Error documentation with return conditions

## Governance

### Constitution Supremacy

This constitution supersedes all other practices and guidelines. All development must comply with these principles. When conflicts arise, the constitution takes precedence.

### Amendment Process

1. **Proposal**: Document proposed changes with rationale
2. **Review**: Review impact on existing code and practices
3. **Approval**: Require approval from project maintainers
4. **Migration**: Create migration plan for existing code
5. **Version**: Increment constitution version
6. **Communication**: Update all relevant documentation

### Compliance Review

- All PRs must verify compliance with constitution
- Code reviews must check constitution adherence
- CI/CD must enforce quality gates
- Complexity must be justified
- Use `.cursor/rules/` for runtime development guidance

### Version History

**Version**: 1.0.0 | **Ratified**: 2026-01-30 | **Last Amended**: 2026-01-30
