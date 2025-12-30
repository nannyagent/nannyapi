# NannyAPI

[![CI](https://github.com/nannyagent/nannyapi/actions/workflows/ci.yml/badge.svg)](https://github.com/nannyagent/nannyapi/actions/workflows/ci.yml)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

NannyAPI is the central control plane for Nanny Agents. It handles agent registration, authentication, investigation orchestration, and patch management.

## Documentation

We have reorganized our documentation to help you get started quickly.

- **[Installation Guide](docs/INSTALLATION.md)**: Instructions for installing the binary, setting up systemd services, and building from source.
- **[Architecture](docs/ARCHITECTURE.md)**: Overview of the system components, AI integration (TensorZero), and observability (ClickHouse).
- **[Patch Management](docs/PATCHING.md)**: Details on how the patching system works, including dry-runs and Proxmox support.
- **[API Reference](docs/API_REFERENCE.md)**: Comprehensive guide to the API endpoints, payloads, and responses.
- **[Security Policy](docs/SECURITY.md)**: Reporting vulnerabilities and AI safety.
- **[Contributing](docs/CONTRIBUTING.md)**: Development setup and guidelines.
- **[Contributors](docs/CONTRIBUTORS.md)**: List of contributors.

## Quick Start

To install the latest version on Linux:

```bash
curl -sL https://raw.githubusercontent.com/nannyagent/nannyapi/main/install.sh | sudo bash
```

For detailed installation and upgrade instructions, please refer to the [Installation Guide](docs/INSTALLATION.md).

## Features

- **Agent Management**: Secure registration and management of Nanny Agents.
- **AI-Powered Investigations**: Integrates with TensorZero to analyze system issues and generate resolution plans.
- **Patch Management**: Automated and manual patching for Linux systems (Debian, RHEL, Arch, SUSE) and Proxmox.
- **Observability**: Built-in support for ClickHouse to store high-fidelity telemetry and AI episodes.
- **Security**: OAuth2 authentication (Google/GitHub) and secure agent communication.

## License

This project is licensed under the GPLv3 License - see the [LICENSE](LICENSE) file for details.
