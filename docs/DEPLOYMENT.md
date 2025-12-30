# Deployment Guide

This guide covers how to deploy NannyAPI in a production environment.

## Prerequisites

- **Server**: A Linux server (Ubuntu/Debian recommended).
- **Database**: 
  - **SQLite**: Embedded (default, handled by PocketBase).
  - **ClickHouse**: Required for advanced AI observability (optional but recommended).
- **AI Gateway**: **TensorZero** instance running and accessible.

## Environment Configuration

NannyAPI uses environment variables for configuration. You should create a `.env` file in your working directory (e.g., `/var/lib/nannyapi/.env`).

### Required Variables

```bash
# GitHub credentials
GITHUB_CLIENT_ID="github-client-id"
GITHUB_CLIENT_SECRET="github-client-secret"

# Google credentials
GOOGLE_CLIENT_ID="google-client-id"
GOOGLE_CLIENT_SECRET="google-client-secret"

# TensorZero Core API Configuration
TENSORZERO_API_URL="tensorzero-core-api-endpoint"
TENSORZERO_API_KEY="tensorzero-core-api-key"

# ClickHouse Configuration (TensorZero Data Storage & Observability)
CLICKHOUSE_URL="clickhouse-endpoint"
CLICKHOUSE_DATABASE="tensorzero"
CLICKHOUSE_USER="clickhouseuser"
CLICKHOUSE_PASSWORD="clickhousepwd"
```

## Systemd Deployment

If you used the `install.sh` script, a systemd service file was created for you. You need to ensure it loads the environment variables.

1. **Edit the Service File**:
   ```bash
   sudo nano /etc/systemd/system/nannyapi.service
   ```

2. **Add EnvironmentFile Directive**:
   Ensure the `[Service]` section includes `EnvironmentFile`.

   ```ini
   [Unit]
   Description=NannyAPI Service
   After=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/var/lib/nannyapi
   # Load env vars from file
   EnvironmentFile=/var/lib/nannyapi/.env
   ExecStart=/usr/local/bin/nannyapi serve --dir="/var/lib/nannyapi/pb_data" --publicDir="/var/lib/nannyapi/pb_public" --http="0.0.0.0:8090"
   Restart=on-failure
   LimitNOFILE=4096

   [Install]
   WantedBy=multi-user.target
   ```

3. **Reload and Restart**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart nannyapi
   ```

## Docker Deployment (Template)

A `Dockerfile` is provided in the repository. You can use it to build a container image.

```bash
docker build -t nannyapi:latest .
```

Run with Docker Compose (example):

```yaml
version: '3.8'
services:
  nannyapi:
    image: nannyapi:latest
    ports:
      - "8090:8090"
    volumes:
      - ./pb_data:/pb_data
    env_file:
      - .env
    depends_on:
      - clickhouse
      - tensorzero
```

## Reverse Proxy (Nginx)

It is highly recommended to run NannyAPI behind a reverse proxy like Nginx or Caddy to handle SSL termination.

**Nginx Example:**

```nginx
server {
    listen 80;
    server_name api.nanny.dev;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```
