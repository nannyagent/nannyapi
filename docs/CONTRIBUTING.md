# Contributing to NannyAPI

We welcome contributions! This guide will help you set up your development environment.

## Development Environment Setup

### Prerequisites
- Go 1.24+
- Docker & Docker Compose (for ClickHouse/TensorZero)
- Make

### Quick Start Script

We provide a script to reset the database and start the server with default configurations.

```bash
./scripts/reset-and-start.sh
```

This script will:
1. Clean up the `pb_data` directory.
2. Build the binary.
3. Load environment variables from `.env`.
4. Run migrations.
5. Create a default admin user (`admin@nannyapi.local` / `AdminPass-123`).
6. Start the server on port 8090.

### Running Tests

**Unit testing is critical to the stability and reliability of NannyAPI.** All contributions must include comprehensive unit tests.

#### Testing Guidelines

1. **Write Tests First**: Follow TDD (Test-Driven Development) when possible
2. **Test Coverage**: Aim for >80% coverage on new code
3. **Test All Paths**: Cover success cases, error cases, and edge cases
4. **Use Table-Driven Tests**: For multiple similar test cases
5. **Mock External Dependencies**: Use mocks for TensorZero, ClickHouse, etc.
6. **Test Files Location**: Place tests in `tests/` directory or alongside code as `*_test.go`

#### Running Tests

All tests are executed via the Makefile:

```bash
# Run all tests with race detection
make test

# Run tests with coverage report
make coverage

# Run specific package tests
go test ./internal/agents/... -v
go test ./tests/ -v -run TestAgentRegistration
```

#### Required Before PR Submission

```bash
# 1. Run tests
make test

# 2. Check coverage
make coverage
# Open coverage.html in browser to review

# 3. Run linter
make lint

# 4. Format code
make fmt
```

**Pull requests without adequate test coverage will not be merged.**

### Code Style

Please ensure your code is formatted and linted before submitting a PR.

```bash
make fmt
make lint
```
