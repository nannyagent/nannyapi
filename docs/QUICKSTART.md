# Quick Start Guide

Get NannyAPI running in 5 minutes.

## Prerequisites

- **Go 1.24+** (for building from source)
- **Linux** (x86_64/amd64 or ARM64 architectures only)
- **Port 8090** available (default PocketBase port)

> **Note**: Currently, only Linux agents are supported. macOS, Windows, and other platforms may be added based on user requests.

## Installation

### Option 1: Binary Install (Recommended)

```bash
curl -sL https://raw.githubusercontent.com/nannyagent/nannyapi/main/install.sh | sudo bash
```

This installs the binary to `/usr/local/bin/nannyapi` and sets up a systemd service.

### Option 2: Docker

```bash
docker run -d \
  --name nannyapi \
  -p 8090:8090 \
  -v $(pwd)/pb_data:/pb_data \
  -e TENSORZERO_GATEWAY_URL=http://tensorzero:3000 \
  nannyagent/nannyapi:latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/nannyagent/nannyapi.git
cd nannyapi
make build
sudo mv nannyapi /usr/local/bin/
```

## Initial Configuration

1. **Start the service:**
   ```bash
   sudo systemctl start nannyapi
   # OR run directly:
   nannyapi serve --http="0.0.0.0:8090"
   ```

2. **Create an admin user:**
   ```bash
   nannyapi superuser upsert admin@example.com YourPassword123!
   ```

3. **Access the admin UI:**
   - Open http://localhost:8090/_/
   - Login with your admin credentials

## Register Your First Agent

### Step 1: Start Device Authorization

From your agent machine:

```bash
curl -X POST http://your-server:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action": "device_auth_start"}'
```

Response:
```json
{
  "device_code": "abcd-1234",
  "user_code": "WXYZ-5678",
  "verification_uri": "http://your-server:8090/_/#/auth/device",
  "expires_in": 600,
  "interval": 5
}
```

### Step 2: Authorize the Device

1. Open the `verification_uri` in a browser
2. Login as admin
3. Enter the `user_code`
4. Click "Authorize"

### Step 3: Register the Agent

```bash
curl -X POST http://your-server:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{
    "action": "register",
    "device_code": "abcd-1234",
    "hostname": "server-01",
    "os_type": "linux",
    "platform_family": "debian",
    "kernel_version": "5.15.0-91-generic"
  }'
```

Response:
```json
{
  "id": "abc123def456",
  "token": "eyJhbGc...",
  "refresh_token": "xyz789..."
}
```

**Save the tokens!** The agent will use these for all future API calls.

## Test the Installation

### Ingest Metrics

```bash
curl -X POST http://your-server:8090/api/agent \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_AGENT_TOKEN" \
  -d '{
    "action": "ingest_metrics",
    "metrics": {
      "cpu_percent": 45.2,
      "memory_used": 8589934592,
      "memory_total": 17179869184,
      "disk_used": 107374182400,
      "disk_total": 536870912000,
      "load_average": {"1m": 1.5, "5m": 1.2, "15m": 0.9},
      "uptime_seconds": 86400
    }
  }'
```

### View Agent in Admin UI

1. Go to http://localhost:8090/_/
2. Click "Collections" → "agents"
3. You should see your registered agent with live metrics

## Next Steps

- **[Architecture Guide](ARCHITECTURE.md)**: Understand the system design
- **[API Reference](API_REFERENCE.md)**: Full API documentation
- **[Deployment Guide](DEPLOYMENT.md)**: Production setup, monitoring, security
- **[Patch Management](PATCHING.md)**: Automated patching workflow

## Common Issues

### Port 8090 Already in Use

```bash
# Check what's using the port
sudo lsof -i :8090

# Use a different port
nannyapi serve --http="0.0.0.0:8091"
```

### Service Won't Start

```bash
# Check logs
sudo journalctl -u nannyapi -n 50 --no-pager

# Check binary permissions
ls -la /usr/local/bin/nannyapi

# Verify data directory
sudo mkdir -p /opt/nannyapi/pb_data
sudo chown nannyapi:nannyapi /opt/nannyapi
```

### Can't Access Admin UI

1. Check firewall: `sudo ufw status`
2. Verify service is running: `sudo systemctl status nannyapi`
3. Test locally first: `curl http://localhost:8090/api/health`

## Development Mode

For local development:

```bash
git clone https://github.com/nannyagent/nannyapi.git
cd nannyapi

# Build and run
make build
./nannyapi serve --http="0.0.0.0:8090"

# Run tests
make test

# Format code
make fmt
```

### Adding a Custom Handler

1. Create handler in `internal/<feature>/handlers.go`
2. Register route in `internal/hooks/setup.go`
3. Add types to `internal/types/<feature>_types.go`
4. Write tests in `tests/<feature>_test.go`

### Database Changes

Migrations are in `pb_migrations/`:

```go
// Example: pb_migrations/1735300000_add_my_field.go
package migrations

import (
    "github.com/pocketbase/dbx"
    "github.com/pocketbase/pocketbase/daos"
    m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    m.Register(func(db dbx.Builder) error {
        dao := daos.New(db)
        collection, _ := dao.FindCollectionByNameOrId("agents")
        
        // Add field logic here
        
        return dao.SaveCollection(collection)
    }, func(db dbx.Builder) error {
        // Rollback logic
        return nil
    })
}
```

## Testing

Run all tests:
```bash
make test
```

Run specific tests:
```bash
go test ./internal/agents/... -v
go test ./tests/ -v -run TestAgentRegistration
```

## Best Practices

1. **Authentication**: Always use bearer tokens for agent endpoints
2. **Metrics**: Ingest every 30 seconds to keep dashboard fresh
3. **Patches**: Always use `--dry-run` first to preview changes
4. **Scripts**: Verify SHA-256 hash before execution
5. **Tokens**: Store refresh tokens securely and rotate before 30-day expiry

## External Services (Optional)

### TensorZero (AI Investigations)

```bash
docker run -d \
  --name tensorzero \
  -p 3000:3000 \
  -e OPENAI_API_KEY=your-key \
  tensorzero/gateway:latest
```

Set `TENSORZERO_GATEWAY_URL=http://localhost:3000` in NannyAPI.

### ClickHouse (Observability)

```bash
docker run -d \
  --name clickhouse \
  -p 9000:9000 \
  -p 8123:8123 \
  clickhouse/clickhouse-server:latest
```

Set `CLICKHOUSE_URL=http://localhost:8123` in NannyAPI.

## Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for production setup, monitoring, and security hardening.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines.

