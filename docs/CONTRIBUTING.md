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

To run the test suite:

```bash
make test
```

For coverage reports:

```bash
make coverage
```

### Code Style

Please ensure your code is formatted and linted before submitting a PR.

```bash
make fmt
make lint
```
