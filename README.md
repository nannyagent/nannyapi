# NannyAPI

<p align="center">
  <img src="https://avatars.githubusercontent.com/u/110624612" alt="NannyAgent Logo" width="150" />
</p>

<p align="center">
  <a href="https://github.com/nannyagent/nannyapi/actions/workflows/ci.yml">
    <img src="https://github.com/nannyagent/nannyapi/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://hub.docker.com/r/nannyagent/nannyapi">
    <img src="https://img.shields.io/docker/v/nannyagent/nannyapi?label=docker" alt="Docker">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
  </a>
</p>

NannyAPI is the central control plane for Nanny Agents. It handles agent registration, authentication, investigation orchestration, and patch management.

## Documentation

We have reorganized our documentation to help you get started quickly.

- **[Quick Start](docs/QUICKSTART.md)**: Get started in 5 minutes.
- **[Installation Guide](docs/INSTALLATION.md)**: Instructions for Docker, binary installation, systemd services, and building from source.
- **[Deployment Guide](docs/DEPLOYMENT.md)**: Production deployment, configuration, monitoring, security hardening, and troubleshooting.
- **[Architecture](docs/ARCHITECTURE.md)**: System components, authentication flows, database schema, AI integration (TensorZero), and observability (ClickHouse).
- **[API Reference](docs/API_REFERENCE.md)**: Complete API documentation with all endpoints, request/response examples, and authentication.
- **[Patch Management](docs/PATCHING.md)**: Automated patching workflow, script verification, exception handling, and dry-run mode.
- **[Proxmox Integration](docs/PROXMOX.md)**: Agentless LXC container management and Proxmox VE support.
- **[Security Policy](docs/SECURITY.md)**: Reporting vulnerabilities and AI safety.
- **[Contributing](docs/CONTRIBUTING.md)**: Development setup, guidelines, and contributor information.

## Quick Start

### Docker (Recommended)

```bash
# Create data directory for persistence
mkdir -p ./pb_data

# Run NannyAPI
docker run -d \
  --name nannyapi \
  -p 8090:8090 \
  -v $(pwd)/pb_data:/app/pb_data \
  docker.io/nannyagent/nannyapi:latest
```

> ⚠️ **Important**: NannyAPI uses SQLite. You must mount `/app/pb_data` to persist data. See [Installation Guide](docs/INSTALLATION.md) for details.

### Binary Installation (Linux)

```bash
curl -sL https://raw.githubusercontent.com/nannyagent/nannyapi/main/install.sh | sudo bash