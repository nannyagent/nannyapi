# Installation Guide

NannyAPI can be installed using several methods. Choose the one that best fits your environment.

## Installation Methods Overview

| Method | Best For | Persistence | Complexity |
|--------|----------|-------------|------------|
| [Docker](#docker-installation) | Quick setup, containerized environments | Volume mount required | Low |
| [Systemd Binary](#automated-installation-linux-systemd) | Production Linux servers | Native filesystem | Medium |
| [Build from Source](#building-from-source) | Development, customization | Native filesystem | High |

---

## Docker Installation

Docker is the recommended method for quick deployments and containerized environments.

### Prerequisites
- Docker Engine 20.10+ or Docker Desktop
- Docker Compose v2+ (optional, but recommended)

### Quick Start with Docker Run

```bash
# Create data directory (CRITICAL for persistence)
mkdir -p ./pb_data

# Run NannyAPI
docker run -d \
  --name nannyapi \
  -p 8090:8090 \
  -v $(pwd)/pb_data:/app/pb_data \
  -e PB_AUTOMIGRATE=true \
  docker.io/nannyagent/nannyapi:latest
```

### Docker Compose (Recommended)

1. Download the docker-compose.yml:
   ```bash
   curl -sL https://raw.githubusercontent.com/nannyagent/nannyapi/main/docker-compose.yml -o docker-compose.yml
   ```

2. Create a `.env` file for configuration:
   ```bash
   cat > .env << 'EOF'
   # OAuth2 (optional)
   GITHUB_CLIENT_ID=your-client-id
   GITHUB_CLIENT_SECRET=your-client-secret
   
   # Frontend URL
   FRONTEND_URL=https://your-frontend.example.com
   EOF
   ```

3. Start the service:
   ```bash
   docker compose up -d
   ```

4. Check logs:
   ```bash
   docker compose logs -f nannyapi
   ```

### ⚠️ Critical: Data Persistence

**NannyAPI uses SQLite (via PocketBase) for data storage. You MUST mount the `/app/pb_data` volume to preserve data across container restarts.**

Without proper volume mounting:
- All agents, users, and configurations will be lost on container restart
- Migration history will be reset
- You will start with a fresh database each time

```bash
# Required volume mount
-v /path/to/persistent/storage:/app/pb_data
```

### Available Docker Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release from main branch |
| `x.y.z` | Specific version (e.g., `1.0.0`) |
| `sha-xxxxxx` | Specific commit SHA |

### Docker Image Registry

Images are published to Docker Hub:
```
docker.io/nannyagent/nannyapi
```

---

## Automated Installation (Linux Systemd)

We provide an `install.sh` script to automate the installation of the `nannyapi` binary and setup a systemd service.

### Prerequisites
- Linux environment with `systemd`
- `curl` installed
- Root privileges

### Steps
1. Download the `install.sh` script.
2. Run the script as root:
   ```bash
   sudo ./install.sh
   ```

The script performs the following actions:
- Downloads the latest binary.
- Archives the current binary (if present) to a `.bkp.ddmmyy` format.
- Replaces the old binary with the new one.
- Sets up a systemd service at `/etc/systemd/system/nannyapi.service`.
- Prints the installed version.
- **Does NOT start the service** (to allow for migrations).

---

## Building from Source

If you prefer to build the binary yourself, follow these steps.

### Prerequisites
- Go 1.21+ installed
- Make (optional, but recommended)

### Build Steps
1. Clone the repository:
   ```bash
   git clone https://github.com/nannyagent/nannyapi.git
   cd nannyapi
   ```

2. Build the binary using `make`:
   ```bash
   make build
   ```
   This will create the binary at `bin/nannyapi`.

   Alternatively, using `go build`:
   ```bash
   go build -o bin/nannyapi ./main.go
   ```

### Building Docker Image Locally

```bash
# Clone repository
git clone https://github.com/nannyagent/nannyapi.git
cd nannyapi

# Build Docker image
docker build -t nannyapi:local .

# Run locally built image
docker run -d \
  --name nannyapi \
  -p 8090:8090 \
  -v $(pwd)/pb_data:/app/pb_data \
  nannyapi:local
```

### Packaging
To package the binary for distribution, you can simply compress the `bin/nannyapi` file.
```bash
tar -czvf nannyapi-linux-amd64.tar.gz -C bin nannyapi
```

---

## Upgrade and Migration

When upgrading to a new version, it is critical to follow these steps to ensure data integrity.

### Understanding Migrations

NannyAPI uses PocketBase's built-in migration system. Migrations are Go files located in `pb_migrations/` that define schema changes. When the application starts with `PB_AUTOMIGRATE=true`, migrations run automatically.

**How migrations work:**
1. On startup, NannyAPI checks the `_migrations` table in SQLite
2. Compares against embedded migration files
3. Runs any pending migrations in order
4. Records completed migrations in `_migrations` table

### 1. Backup Data

**CRITICAL**: Before running any migration scripts, you **MUST** backup your `pb_data` directory.

#### For Docker:
```bash
# Stop container first
docker compose stop nannyapi

# Backup the volume
cp -r ./pb_data ./pb_data.bkp.$(date +%Y%m%d)

# Or use docker cp
docker cp nannyapi:/app/pb_data ./pb_data.bkp.$(date +%Y%m%d)
```

#### For Systemd/Binary:
```bash
# Example backup command
cp -r /var/lib/nannyapi/pb_data /var/lib/nannyapi/pb_data.bkp.$(date +%Y%m%d)
```

### 2. Run Migrations

#### For Docker (Automatic):
Migrations run automatically when `PB_AUTOMIGRATE=true` (default). Simply pull the new image and restart:

```bash
docker compose pull
docker compose up -d
```

#### For Docker (Manual):
```bash
docker run --rm \
  -v $(pwd)/pb_data:/app/pb_data \
  docker.io/nannyagent/nannyapi:latest \
  ./nannyapi migrate up
```

#### For Systemd/Binary:
```bash
# Run migrations
/usr/local/bin/nannyapi migrate up --dir="/var/lib/nannyapi/pb_data"
```

### 3. Verify Migrations
Ensure the migration command finished without errors and returned a 0 exit code. Check the output for any failure messages.

### 4. Start the Service

#### For Docker:
```bash
docker compose up -d
```

#### For Systemd:
**Important**: Ensure you have created a `.env` file in `/var/lib/nannyapi/.env` with the necessary configuration variables before starting the service. See [Deployment Guide](DEPLOYMENT.md) for details.

```bash
sudo systemctl start nannyapi
```

### 5. Verify Service Status

#### For Docker:
```bash
docker compose ps
docker compose logs -f nannyapi
```

#### For Systemd:
```bash
sudo systemctl status nannyapi
# Check logs
sudo journalctl -u nannyapi -f
```

---

## Important Considerations

### SQLite and High Availability

⚠️ **NannyAPI uses SQLite as its database backend. This has important implications:**

1. **Single Instance Only**: Run only ONE instance of NannyAPI at a time. Multiple instances will cause database locking issues.

2. **No Horizontal Scaling**: You cannot run multiple replicas for load balancing. SQLite does not support concurrent writes from multiple processes.

3. **No Kubernetes Replicas**: If deploying to Kubernetes, set `replicas: 1` and use `RollingUpdate` strategy with `maxUnavailable: 0`.

4. **Backup Strategy**: Since SQLite is a single file, backups are straightforward but must be done when the application is stopped or using SQLite's backup API.

### Why SQLite?

SQLite was chosen for NannyAPI because:
- Zero external dependencies (no separate database server)
- Simple deployment and operations
- Excellent performance for single-node workloads
- Built-in with PocketBase

For high-availability requirements, consider placing NannyAPI behind a load balancer with health checks and implementing proper backup/restore procedures.
