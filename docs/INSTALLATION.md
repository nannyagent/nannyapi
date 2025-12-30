# Installation Guide

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

> **Note**: Docker installation is currently provided as a template and will be fully supported in future releases.

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

### Packaging
To package the binary for distribution, you can simply compress the `bin/nannyapi` file.
```bash
tar -czvf nannyapi-linux-amd64.tar.gz -C bin nannyapi
```

## Upgrade and Migration

When upgrading to a new version, it is critical to follow these steps to ensure data integrity.

### 1. Backup Data
**CRITICAL**: Before running any migration scripts, you **MUST** backup your `pb_data` directory.

```bash
# Example backup command
cp -r /var/lib/nannyapi/pb_data /var/lib/nannyapi/pb_data.bkp.$(date +%d%m%y)
```

### 2. Run Migrations
The `nannyapi` binary includes migration commands. You should execute these pointing to the `pb_migrations` directory if you have custom migrations or rely on the embedded ones.

```bash
# Run migrations
/usr/local/bin/nannyapi migrate up --dir="/var/lib/nannyapi/pb_data"
```

### 3. Verify Migrations
Ensure the migration command finished without errors and returned a 0 exit code. Check the output for any failure messages.

### 4. Start the Service
Once migrations are successful, start the systemd service.

**Important**: Ensure you have created a `.env` file in `/var/lib/nannyapi/.env` with the necessary configuration variables before starting the service. See [Deployment Guide](DEPLOYMENT.md) for details.

```bash
sudo systemctl start nannyapi
```

### 5. Verify Service Status
Check the service status and logs to ensure no errors are observed.

```bash
sudo systemctl status nannyapi
# Check logs
sudo journalctl -u nannyapi -f
```
