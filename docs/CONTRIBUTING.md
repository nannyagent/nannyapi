# Contributing to NannyAPI

<p align="center">
  <img src="https://avatars.githubusercontent.com/u/110624612" alt="NannyAgent Logo" width="120" />
</p>

We welcome contributions! This guide will help you set up your development environment and understand our contribution process.

## Development Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to your branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Code Style Guidelines

- Follow standard Go coding conventions
- Use `gofmt` to format your code
- Run `go vet` and `staticcheck` before committing
- Maintain test coverage above 80%
- Document all exported functions and types
- Keep functions small and focused
- Use meaningful variable names

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

## Commit Messages

- Use clear and meaningful commit messages
- Follow conventional commits format:
  - `feat:` New feature
  - `fix:` Bug fix
  - `docs:` Documentation changes
  - `test:` Adding tests
  - `refactor:` Code refactoring
  - `style:` Formatting changes
  - `chore:` Maintenance tasks

## Pull Request Process

1. Follow the PR template
2. Ensure all tests pass
3. Update relevant documentation
4. Add test cases for new functionality
5. Request review from maintainers
6. Address review comments promptly

## Code Review

- Be respectful and constructive in reviews
- Focus on code quality and correctness
- Consider performance implications
- Check for security issues
- Verify documentation updates

## Security Issues

- Report security vulnerabilities privately to: **support@nannyai.dev**
- Do not create public issues for security bugs
- Follow responsible disclosure practices

## Contributors

Thank you to all the contributors who have helped make NannyAPI better! 🎉

<!-- This section is automatically updated by GitHub's contribution tracking -->

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

### Code Style

Please ensure your code is formatted and linted before submitting a PR.

```bash
make fmt
make lint
```
