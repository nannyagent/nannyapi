# Deployment Guide

Production deployment guide for NannyAPI with systemd, Docker, and reverse proxy configurations.

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu/Debian recommended)
- **RAM**: Minimum 512MB, recommended 2GB+
- **Disk**: 4GB+ for application and database
- **Network**: Outbound HTTPS (443) for OAuth2 and AI services

### Required Software
- **NannyAPI**: Binary or Docker image
- **SQLite**: Embedded (included in PocketBase)

### Optional (Recommended for Production)
- **TensorZero**: AI gateway for investigations
- **ClickHouse**: Observability data storage
- **Nginx/Caddy**: Reverse proxy for SSL termination
- **Systemd**: Service management

---

## Environment Configuration

NannyAPI uses environment variables for configuration. Create `.env` file in working directory.

### Minimal Configuration

```bash
# Required for OAuth2 (optional but recommended)
GITHUB_CLIENT_ID="your-github-client-id"
GITHUB_CLIENT_SECRET="your-github-client-secret"
GOOGLE_CLIENT_ID="your-google-client-id"
GOOGLE_CLIENT_SECRET="your-google-client-secret"

# Frontend URL for device authorization flow
FRONTEND_URL="https://app.example.com"
```

### Full Configuration (with AI & Observability)

```bash
# ─── OAuth2 Providers ───────────────────────────────────────
GITHUB_CLIENT_ID="github-client-id"
GITHUB_CLIENT_SECRET="github-client-secret"
GOOGLE_CLIENT_ID="google-client-id"
GOOGLE_CLIENT_SECRET="google-client-secret"

# ─── Frontend Configuration ─────────────────────────────────
FRONTEND_URL="https://app.example.com"

# ─── TensorZero AI Gateway ──────────────────────────────────
TENSORZERO_API_URL="https://tensorzero.example.com"
TENSORZERO_API_KEY="tensorzero-api-key"

# ─── ClickHouse Observability ───────────────────────────────
CLICKHOUSE_URL="https://clickhouse.example.com:8123"
CLICKHOUSE_DATABASE="tensorzero"
CLICKHOUSE_USER="nannyapi"
CLICKHOUSE_PASSWORD="clickhouse-password"

# ─── PocketBase Settings ────────────────────────────────────
PB_AUTOMIGRATE="true"  # Auto-run migrations on start
```

### OAuth2 Setup

#### GitHub OAuth App
1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Set Authorization callback URL: `http://localhost:8090/api/oauth2-redirect`
4. Copy Client ID and Client Secret

#### Google OAuth2
1. Go to Google Cloud Console → APIs & Services → Credentials
2. Create OAuth 2.0 Client ID
3. Set Authorized redirect URIs: `http://localhost:8090/api/oauth2-redirect`
4. Copy Client ID and Client Secret

**Production:** Replace `localhost:8090` with your production domain.

---

## Installation Methods

### Method 1: Binary Installation (Recommended)

```bash
# Download latest release
curl -sL https://github.com/nannyagent/nannyapi/releases/latest/download/nannyapi-linux-amd64 \
  -o /usr/local/bin/nannyapi

# Make executable
chmod +x /usr/local/bin/nannyapi

# Create working directory
mkdir -p /var/lib/nannyapi
cd /var/lib/nannyapi

# Create .env file
nano .env
# Paste configuration from above

# Test run
./nannyapi serve --dir="./pb_data" --http="0.0.0.0:8090"
```

### Method 2: Build from Source

```bash
# Install Go 1.21+
curl -OL https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone repository
git clone https://github.com/nannyagent/nannyapi.git
cd nannyapi

# Build
go build -o nannyapi

# Install
sudo cp nannyapi /usr/local/bin/
```

### Method 3: Docker

> **⚠️ Note**: Docker deployment support is currently in development. The examples below are speculative and will be updated when Docker images are officially published. For production use, please use Method 1 (Binary) or Method 2 (Source) until Docker support is finalized.

```bash
# Using docker run (Coming Soon)
docker run -d \
  --name nannyapi \
  -p 8090:8090 \
  -v $(pwd)/pb_data:/pb_data \
  -v $(pwd)/.env:/.env \
  --env-file .env \
  ghcr.io/nannyagent/nannyapi:latest \
  serve --dir="/pb_data" --http="0.0.0.0:8090"
```

**Docker Compose (Coming Soon):**
```yaml
version: '3.8'

services:
  nannyapi:
    image: ghcr.io/nannyagent/nannyapi:latest
    container_name: nannyapi
    restart: unless-stopped
    ports:
      - "8090:8090"
    volumes:
      - ./pb_data:/pb_data
      - ./.env:/.env
    env_file:
      - .env
    command: serve --dir="/pb_data" --http="0.0.0.0:8090"
    depends_on:
      - clickhouse  # Optional: if using ClickHouse

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    container_name: clickhouse
    restart: unless-stopped
    ports:
      - "8123:8123"
      - "9000:9000"
    environment:
      CLICKHOUSE_DB: tensorzero
      CLICKHOUSE_USER: nannyapi
      CLICKHOUSE_PASSWORD: clickhouse-password
    volumes:
      - clickhouse_data:/var/lib/clickhouse

volumes:
  clickhouse_data:
```

---

## Systemd Service Configuration

Recommended for production deployments.

### Create Service File

```bash
sudo nano /etc/systemd/system/nannyapi.service
```

**Service Configuration:**
```ini
[Unit]
Description=NannyAPI Service - Agent Management & Patch Orchestration
Documentation=https://github.com/nannyagent/nannyapi
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/var/lib/nannyapi
EnvironmentFile=/var/lib/nannyapi/.env

# Security Hardening (optional but recommended)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/var/lib/nannyapi

# Resource Limits
LimitNOFILE=65536
LimitNPROC=4096

# Main service command
ExecStart=/usr/local/bin/nannyapi serve \
  --dir="/var/lib/nannyapi/pb_data" \
  --http="0.0.0.0:8090"

# Auto-restart on failure
Restart=on-failure
RestartSec=10s
StartLimitBurst=5
StartLimitInterval=60s

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nannyapi

[Install]
WantedBy=multi-user.target
```

### Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable nannyapi

# Start service
sudo systemctl start nannyapi

# Check status
sudo systemctl status nannyapi

# View logs
sudo journalctl -u nannyapi -f
```

### Service Management Commands

```bash
# Start service
sudo systemctl start nannyapi

# Stop service
sudo systemctl stop nannyapi

# Restart service
sudo systemctl restart nannyapi

# Reload configuration (without restart)
sudo systemctl reload nannyapi

# View logs (last 50 lines)
sudo journalctl -u nannyapi -n 50

# Follow logs in real-time
sudo journalctl -u nannyapi -f

# View logs since boot
sudo journalctl -u nannyapi -b
```

---

## Reverse Proxy Setup

### Nginx Configuration

**Install Nginx:**
```bash
sudo apt update
sudo apt install nginx certbot python3-certbot-nginx
```

**Create Site Configuration:**
```bash
sudo nano /etc/nginx/sites-available/nannyapi
```

**HTTP Configuration (for testing):**
```nginx
server {
    listen 80;
    server_name api.example.com;

    # Request size limits
    client_max_body_size 100M;

    # Proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Main API
    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # Timeouts for long-running operations
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # Real-time subscriptions (SSE)
    location /api/realtime {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # SSE specific settings
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # File downloads
    location /api/files {
        proxy_pass http://127.0.0.1:8090;
        proxy_buffering off;
        proxy_request_buffering off;
    }
}
```

**HTTPS Configuration (production):**
```nginx
server {
    listen 80;
    server_name api.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    # SSL Certificates (Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Request size limits
    client_max_body_size 100M;

    # Proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # Main API
    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # Real-time subscriptions
    location /api/realtime {
        proxy_pass http://127.0.0.1:8090;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }

    # File downloads
    location /api/files {
        proxy_pass http://127.0.0.1:8090;
        proxy_buffering off;
        proxy_request_buffering off;
    }
}
```

**Enable Site:**
```bash
# Create symbolic link
sudo ln -s /etc/nginx/sites-available/nannyapi /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx

# Obtain SSL certificate
sudo certbot --nginx -d api.example.com
```

### Caddy Configuration

**Install Caddy:**
```bash
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install caddy
```

**Caddyfile:**
```
api.example.com {
    reverse_proxy localhost:8090 {
        # Real-time subscriptions
        @realtime path /api/realtime*
        handle @realtime {
            header_up Connection "Upgrade"
            header_up Upgrade "websocket"
        }
    }
    
    # Request limits
    request_body {
        max_size 100MB
    }
}

# Caddy Configuration (continued)

```
    
    # Auto-reload on configuration change
    auto_https off
}
```

**Reload Caddy:**
```bash
sudo systemctl reload caddy
```

---

## Creating Admin User

After first deployment, create an admin superuser:

```bash
# Navigate to working directory
cd /var/lib/nannyapi

# Create admin user
./nannyapi superuser upsert admin@example.com SecurePassword123!

# Or if installed globally
nannyapi superuser upsert admin@example.com SecurePassword123!
```

**Admin Login:**
- Navigate to `https://api.example.com/_/`
- Login with admin credentials
- Access PocketBase admin dashboard

---

## Database Management

### Backup Strategy

**Automated Backup Script:**
```bash
#!/bin/bash
# /usr/local/bin/backup-nannyapi.sh

BACKUP_DIR="/var/backups/nannyapi"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_PATH="/var/lib/nannyapi/pb_data/data.db"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Stop service (optional, for consistency)
# systemctl stop nannyapi

# Backup SQLite database
sqlite3 "$DB_PATH" ".backup '$BACKUP_DIR/data_$TIMESTAMP.db'"

# Backup storage files
tar -czf "$BACKUP_DIR/storage_$TIMESTAMP.tar.gz" \
  /var/lib/nannyapi/pb_data/storage/

# Start service if stopped
# systemctl start nannyapi

# Keep only last 7 days
find "$BACKUP_DIR" -type f -mtime +7 -delete

echo "Backup completed: $TIMESTAMP"
```

**Schedule with Cron:**
```bash
# Edit crontab
sudo crontab -e

# Add daily backup at 2 AM
0 2 * * * /usr/local/bin/backup-nannyapi.sh >> /var/log/nannyapi-backup.log 2>&1
```

### Database Migration

Migrations run automatically when `PB_AUTOMIGRATE=true` is set. For manual migration:

```bash
# Dry-run (show pending migrations)
./nannyapi migrate collections

# Apply migrations
./nannyapi migrate up
```

### Database Optimization

**Vacuum Database (shrink file size):**
```bash
sqlite3 /var/lib/nannyapi/pb_data/data.db "VACUUM;"
```

**Analyze Database (update statistics):**
```bash
sqlite3 /var/lib/nannyapi/pb_data/data.db "ANALYZE;"
```

**Scheduled Optimization:**
```bash
# Monthly on 1st at 3 AM
0 3 1 * * sqlite3 /var/lib/nannyapi/pb_data/data.db "VACUUM; ANALYZE;"
```

---

## Monitoring & Health Checks

### Health Endpoint

```bash
# PocketBase built-in health check
curl http://localhost:8090/api/health

# Expected response: 200 OK
```

### Service Monitoring with Systemd

**Check Service Status:**
```bash
systemctl status nannyapi
```

**Enable Email Alerts on Failure:**
```bash
# Install mailutils
sudo apt install mailutils

# Edit service file
sudo nano /etc/systemd/system/nannyapi.service

# Add under [Service]
OnFailure=status-email@%n.service
```

### External Monitoring

**Uptime Monitoring:**
- UptimeRobot: https://uptimerobot.com
- Pingdom: https://www.pingdom.com
- StatusCake: https://www.statuscake.com

**APM Integration:**
- Datadog
- New Relic
- Prometheus + Grafana

---

## Scaling Considerations

### Single Server (Default)
- Supports 100-1000 agents
- 10k concurrent real-time connections
- Suitable for most deployments

### Multi-Server Deployment

**Requirements:**
- Shared storage for `pb_data/`
- Sticky sessions for real-time subscriptions
- Load balancer with health checks

**Example (Nginx Load Balancer):**
```nginx
upstream nannyapi_backend {
    ip_hash;  # Sticky sessions for real-time
    server 192.168.1.10:8090 max_fails=3 fail_timeout=30s;
    server 192.168.1.11:8090 max_fails=3 fail_timeout=30s;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;
    
    location / {
        proxy_pass http://nannyapi_backend;
        # ... proxy headers ...
    }
}
```

**Shared Storage Options:**
- NFS mount for `pb_data/`
- GlusterFS for distributed storage
- Ceph for object storage

---

## Security Hardening

### Firewall Configuration

**UFW (Ubuntu):**
```bash
# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP/HTTPS (if not behind proxy)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Or allow only from load balancer
sudo ufw allow from 192.168.1.100 to any port 8090

# Enable firewall
sudo ufw enable
```

### SSL/TLS Best Practices

**Let's Encrypt with Certbot:**
```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d api.example.com

# Auto-renewal (already configured by certbot)
sudo certbot renew --dry-run
```

**Strong SSL Configuration:**
```nginx
# TLS 1.2+ only
ssl_protocols TLSv1.2 TLSv1.3;

# Strong ciphers
ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';

# HSTS
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

# OCSP Stapling
ssl_stapling on;
ssl_stapling_verify on;
ssl_trusted_certificate /etc/letsencrypt/live/api.example.com/chain.pem;
```

### File Permissions

```bash
# Set ownership
sudo chown -R root:root /var/lib/nannyapi

# Restrict permissions
sudo chmod 700 /var/lib/nannyapi
sudo chmod 600 /var/lib/nannyapi/.env
sudo chmod 644 /var/lib/nannyapi/pb_data/data.db
```

### Fail2Ban Integration

**Create Fail2Ban Filter:**
```bash
sudo nano /etc/fail2ban/filter.d/nannyapi.conf
```

```ini
[Definition]
failregex = ^.*"error":"authentication required".*"ip":"<HOST>".*$
            ^.*"error":"invalid credentials".*"ip":"<HOST>".*$
ignoreregex =
```

**Create Jail:**
```bash
sudo nano /etc/fail2ban/jail.d/nannyapi.conf
```

```ini
[nannyapi]
enabled = true
port = 8090,http,https
filter = nannyapi
logpath = /var/log/nannyapi/*.log
maxretry = 5
bantime = 3600
findtime = 600
```

---

## External Services Setup

### TensorZero Deployment

See TensorZero documentation: https://www.tensorzero.com/docs

**Configuration in NannyAPI:**
```bash
TENSORZERO_API_URL="https://tensorzero.example.com"
TENSORZERO_API_KEY="your-api-key"
```

### ClickHouse Deployment

> **\u26a0\ufe0f Note**: Docker Compose examples below are speculative. Use official ClickHouse installation methods until Docker support is finalized.

**Docker Compose (Speculative):**
```yaml
clickhouse:
  image: clickhouse/clickhouse-server:latest
  restart: unless-stopped
  ports:
    - "8123:8123"  # HTTP
    - "9000:9000"  # Native
  environment:
    CLICKHOUSE_DB: tensorzero
    CLICKHOUSE_USER: nannyapi
    CLICKHOUSE_PASSWORD: secure-password
  volumes:
    - clickhouse_data:/var/lib/clickhouse
```

**Cloud Options:**
- ClickHouse Cloud: https://clickhouse.com/cloud
- AWS: ClickHouse on EC2
- GCP: ClickHouse on GCE

**Configuration in NannyAPI:**
```bash
CLICKHOUSE_URL="https://clickhouse.example.com:8123"
CLICKHOUSE_DATABASE="tensorzero"
CLICKHOUSE_USER="nannyapi"
CLICKHOUSE_PASSWORD="secure-password"
```

---

## Troubleshooting

### Service Won't Start

**Check logs:**
```bash
sudo journalctl -u nannyapi -n 50
```

**Common issues:**
- Port 8090 already in use: Change port or kill process
- Missing .env file: Ensure file exists and is readable
- Permission denied: Check file ownership and permissions

### Database Locked

**Symptoms:** `database is locked` errors

**Solutions:**
- Stop all instances: `sudo systemctl stop nannyapi`
- Check for stale locks: `fuser /var/lib/nannyapi/pb_data/data.db`
- Kill processes: `sudo killall nannyapi`
- Restart: `sudo systemctl start nannyapi`

### High Memory Usage

**Check memory:**
```bash
ps aux | grep nannyapi
free -h
```

**Solutions:**
- Increase server RAM
- Optimize queries (add indexes)
- Clean up old metrics/investigations
- Implement data retention policies

### Slow API Responses

**Analyze:**
- Check database size: `du -h /var/lib/nannyapi/pb_data/data.db`
- Review slow queries in logs
- Monitor system resources: `htop`

**Optimize:**
- VACUUM database
- Add database indexes
- Enable query caching
- Upgrade server resources

---

## Upgrade Procedure

### Backup First
```bash
# Run backup script
/usr/local/bin/backup-nannyapi.sh
```

### Upgrade Binary
```bash
# Stop service
sudo systemctl stop nannyapi

# Backup current binary
sudo cp /usr/local/bin/nannyapi /usr/local/bin/nannyapi.backup

# Download new version
curl -sL https://github.com/nannyagent/nannyapi/releases/latest/download/nannyapi-linux-amd64 \
  -o /tmp/nannyapi

# Install
sudo mv /tmp/nannyapi /usr/local/bin/nannyapi
sudo chmod +x /usr/local/bin/nannyapi

# Start service (migrations run automatically)
sudo systemctl start nannyapi

# Check status
sudo systemctl status nannyapi
```

### Rollback
```bash
# Stop service
sudo systemctl stop nannyapi

# Restore backup binary
sudo cp /usr/local/bin/nannyapi.backup /usr/local/bin/nannyapi

# Restore database (if needed)
sudo cp /var/backups/nannyapi/data_TIMESTAMP.db /var/lib/nannyapi/pb_data/data.db

# Start service
sudo systemctl start nannyapi
```

---

## Production Checklist

Before going to production, verify:

- [ ] SSL/TLS certificate installed and valid
- [ ] Reverse proxy configured with security headers
- [ ] Firewall rules configured
- [ ] Admin user created with strong password
- [ ] OAuth2 providers configured (if using)
- [ ] Environment variables set in .env
- [ ] Systemd service enabled and running
- [ ] Automated backups configured
- [ ] Monitoring and alerts set up
- [ ] Log rotation configured
- [ ] Database optimized and indexed
- [ ] TensorZero and ClickHouse configured (if using)
- [ ] Test agent registration flow
- [ ] Test patch operation workflow
- [ ] Test investigation workflow
- [ ] Documentation updated for team

---

## Related Documentation

- [Architecture Guide](ARCHITECTURE.md): System design and components
- [API Reference](API_REFERENCE.md): Complete API documentation
- [Patch Management](PATCHING.md): Patch workflow details
- [Security Policy](SECURITY.md): Security best practices
