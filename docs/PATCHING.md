# Patch Management

<p align="center">
  <img src="https://avatars.githubusercontent.com/u/110624612" alt="NannyAgent Logo" width="120" />
</p>

Comprehensive patch management system for **Linux systems only**, supporting multiple distributions with SHA-256 integrity verification and persistent package exceptions.

## Overview

NannyAPI's patch system provides:
- **Secure Script Delivery**: SHA-256 verification prevents MITM attacks
- **Dry-Run Mode**: Preview updates before applying
- **Package Exceptions**: Persistent exclusions per agent
- **Multi-Distribution Support**: Debian, RHEL, Arch, SUSE families (only)
- **Automated Script Selection**: Auto-matches agent platform
- **Real-time Execution**: Agents receive operations via subscriptions

**Supported Linux Distributions:**
- **Debian/Ubuntu** (apt) - `platform_family: debian`
- **RHEL/CentOS/Fedora/Rocky/AlmaLinux** (dnf/yum) - `platform_family: rhel`
- **Arch Linux/Manjaro** (pacman) - `platform_family: arch`
- **SUSE/openSUSE** (zypper) - `platform_family: suse`

> **Note**: Additional Linux distributions may be added based on user requests.

## Complete Patch Workflow

```flowchart
┌──────────────────────────────────────────────────────────────┐
│                 END-TO-END PATCH WORKFLOW                    │
└──────────────────────────────────────────────────────────────┘

1. USER INITIATES PATCH
   └─> POST /api/patches
       {
         "agent_id": "abc123",
         "mode": "dry-run"
       }

2. API PROCESSES REQUEST
   ├─> Verify user owns agent
   ├─> Get agent platform_family (e.g., "debian")
   ├─> Query scripts collection for matching script
   ├─> Auto-populate script_id, script_url
   ├─> Query package_exceptions for active exclusions
   ├─> Inject exclusions into patch operation
   └─> Create patch_operation record (status: pending)

3. AGENT RECEIVES NOTIFICATION
   └─> Via realtime subscription to patch_operations

4. AGENT VALIDATES SCRIPT
   ├─> GET /api/scripts/{script_id}/validate
   │   Returns: {"sha256": "abc123..."}
   └─> Store expected SHA256

5. AGENT DOWNLOADS SCRIPT
   ├─> GET /api/files/scripts/{id}/{filename}
   └─> Calculate SHA256 of downloaded file

6. INTEGRITY CHECK
   ├─> Compare calculated vs expected SHA256
   ├─> If mismatch → ABORT, report error
   └─> If match → Proceed

7. EXECUTE SCRIPT
   ├─> Make script executable: chmod +x script.sh
   ├─> Run with arguments:
   │   ./apt-update.sh --dry-run --exclude kernel,nginx
   ├─> Capture: stdout, stderr, exit_code
   └─> Monitor execution

8. UPLOAD RESULTS
   └─> POST /api/patches/{id}/result
       - stdout_file: execution output
       - stderr_file: error output
       - exit_code: 0 (success) or non-zero
       - status: completed or failed

9. USER REVIEWS RESULTS
   ├─> View patch operation in dashboard
   ├─> Download stdout/stderr logs
   └─> Decide: apply updates or cancel
```

## Patch Modes

### Dry-Run Mode
Simulates updates without making changes:
- Lists all available updates
- Shows current and target versions
- Displays package sizes and dependencies
- Returns JSON output for UI parsing
- No system modifications

**Example Output:**
```json
{
  "updates_available": 15,
  "packages": [
    {
      "package": "nginx",
      "current_version": "1.18.0-0ubuntu1.4",
      "new_version": "1.18.0-0ubuntu1.5",
      "repository": "Ubuntu:22.04/jammy-updates"
    },
    {
      "package": "openssl",
      "current_version": "3.0.2-0ubuntu1.10",
      "new_version": "3.0.2-0ubuntu1.12",
      "repository": "Ubuntu:22.04/jammy-security"
    }
  ],
  "dry_run": true
}
```

### Apply Mode
Executes actual updates:
- Refreshes package indexes
- Downloads and installs updates
- Respects package exclusions
- Captures full output logs
- Reports success/failure status

**Safety Features:**
- Non-interactive mode (no user prompts)
- Automatic yes to all prompts
- Dependency resolution
- Atomic updates where supported
- Exit code reporting

## Package Exceptions

Persistent package exclusions prevent specific packages from being updated.

### Creating Exceptions

**Via API:**
```json
POST /api/collections/package_exceptions/records
Authorization: Bearer <user_token>

{
  "agent_id": "abc123xyz456",
  "package_name": "kernel",
  "reason": "Custom kernel build with hardware driver patches",
  "is_active": true,
  "expires_at": "2024-12-31T23:59:59Z"
}
```

### Exception Behavior

**Automatic Injection:**
When creating a patch operation, the API automatically:
1. Queries `package_exceptions` for the agent
2. Filters for active, non-expired exceptions
3. Extracts package names
4. Injects as `exclusions` array in patch operation
5. Passes to script via `--exclude` argument

**Script Implementation:**
- **Debian/Ubuntu**: `apt-mark hold` before update
- **RHEL/Fedora**: `--exclude=package` flag
- **Arch Linux**: `--ignore` flag
- **SUSE**: `--exclude` flag

### Managing Exceptions

**List Exceptions:**
```bash
GET /api/collections/package_exceptions/records?filter=agent_id='abc123'
```

**Deactivate Exception:**
```json
PATCH /api/collections/package_exceptions/records/{id}

{
  "is_active": false
}
```

**Temporary Exception:**
Set `expires_at` for automatic expiration:
```json
{
  "expires_at": "2024-02-01T00:00:00Z"
}
```

## Supported Distributions

### Debian/Ubuntu Family
**Package Manager:** apt
**Script:** `debian/apt-update.sh`
**Supported Versions:** Debian 10+, Ubuntu 20.04+

**Features:**
- `apt-mark hold` for exclusions
- Automatic unhold after update
- JSON output with package details
- Security update prioritization
- Repository information

**Platform Detection:**
- `platform_family`: `debian`
- Keywords: "ubuntu", "debian", "mint", "pop", "elementary"

### RHEL/CentOS/Fedora Family
**Package Manager:** dnf/yum
**Scripts:**
- `rhel/dnf-update.sh` (RHEL 8+, Fedora)
- `rhel/yum-update.sh` (Legacy RHEL/CentOS 7)

**Features:**
- Native `--exclude` flag support
- Transaction-based updates
- Group updates
- Obsolete package handling
- Repository filtering

**Platform Detection:**
- `platform_family`: `rhel`
- Keywords: "red hat", "rhel", "centos", "fedora", "alma", "rocky", "oracle"

### Arch Linux Family
**Package Manager:** pacman
**Script:** `arch/pacman-update.sh`
**Supported:** Arch Linux, Manjaro

**Features:**
- `--ignore` flag for exclusions
- System upgrade: `pacman -Syu`
- AUR package support (optional)
- Rolling release handling

**Platform Detection:**
- `platform_family`: `arch`
- Keywords: "arch", "manjaro", "endeavouros"

### SUSE/openSUSE Family
**Package Manager:** zypper
**Script:** `suse/zypper-update.sh`
**Supported:** SLES 15+, Leap 15+, Tumbleweed

**Features:**
- Patch and package updates
- `--exclude` flag support
- Repository management
- Pattern updates

**Platform Detection:**
- `platform_family`: `suse`
- Keywords: "suse", "sles", "opensuse", "leap", "tumbleweed"

## Patch Scripts

All scripts follow a standardized interface:

### Script Arguments
```bash
./patch-script.sh [OPTIONS]

OPTIONS:
  --dry-run List available updates without applying
  --exclude PKG1,PKG2 Exclude specific packages (comma-separated)
```

### Exit Codes
- `0`: Success (updates applied or dry-run completed)
- `1`: General error (invalid arguments, permission denied)
- `2`: No updates available
- `100+`: Package manager specific errors

### Output Format

**Human-Readable:**
```text
=== Debian/Ubuntu APT Update Script ===
Dry Run: true
Excluded Packages: kernel,nginx

=== Holding excluded packages ===
Holding package: kernel
Holding package: nginx

=== Updating package lists ===
[DRY RUN] Would run: apt-get update

=== Available Updates ===
15 packages can be upgraded:
  nginx: 1.18.0-0ubuntu1.4 → 1.18.0-0ubuntu1.5
  openssl: 3.0.2-0ubuntu1.10 → 3.0.2-0ubuntu1.12
  ...
```

**JSON Output (for parsing):**
```json
{
  "updates_available": 15,
  "packages": [...],
  "dry_run": true,
  "exclusions": ["kernel", "nginx"],
  "execution_time": 2.5
}
```

### Script Security

**SHA-256 Verification:**
1. Scripts stored in `scripts` collection
2. SHA-256 calculated automatically on upload
3. Agent validates before execution
4. Prevents tampering during transit

**Script Upload Workflow:**
```bash
# Upload new script (API admin only)
POST /api/collections/scripts/records

{
  "name": "apt-update.sh",
  "platform_family": "debian",
  "file": <binary_upload>
}

# SHA-256 auto-calculated by hook
# Returns: {"sha256": "abc123..."}
```

## Advanced Scenarios

### Version-Specific Scripts

For OS version-specific scripts:

```json
{
  "name": "apt-update-focal.sh",
  "platform_family": "debian",
  "os_version": "20.04",
  "file": <upload>
}
```

**Selection Priority:**
1. Exact match: `platform_family` + `os_version`
2. Family match: `platform_family` only
3. Error if no match found

### Emergency Rollback

If patch causes issues:

1. **Review Patch Output:**
   - Download stdout/stderr files
   - Check exit code
   - Identify problematic package

2. **Add Exception:**
   ```json
   POST /api/collections/package_exceptions/records
   {
     "agent_id": "abc123",
     "package_name": "problematic-pkg",
     "reason": "Causes service crash in production",
     "is_active": true
   }
   ```

3. **Manual Rollback (if needed):**
   - SSH into agent machine
   - Downgrade package: `apt install pkg=old-version`

### Scheduled Patching

NannyAPI provides built-in cron-based scheduling for patch operations via the `patch_schedules` collection.

**Schedule Features:**
- Define schedules using standard cron expressions (e.g., `0 2 * * *`)
- **Uniqueness Constraint**: Only **one** active schedule is allowed per Agent (host) or per LXC container.
  - You cannot create multiple schedules for the same host.
  - You cannot create multiple schedules for the same LXC container.
  - To change the frequency, update the existing schedule instead of creating a new one.

**Creating a Schedule:**
```bash
POST /api/collections/patch_schedules/records
{
  "agent_id": "abc123host",
  "lxc_id": "lxc100",  // Optional: Omit for host schedule
  "cron_expression": "0 3 * * 0",  // Weekly on Sunday at 3 AM
  "is_active": true
}
```

**Deprecated Manual Cron Method:**
(Legacy) Formerly required external scheduler (cron, systemd timers).

---

## Remote Reboot Management

NannyAPI provides a complete remote reboot system that allows users to reboot agents (hosts or LXC containers) directly from the API.

### Reboot Architecture

```flowchart
┌──────────────────────────────────────────────────────────────┐
│                    REBOOT WORKFLOW                           │
└──────────────────────────────────────────────────────────────┘

1. USER INITIATES REBOOT
   └─> POST /api/reboot
       {
         "agent_id": "abc123",
         "lxc_id": "lxc100",   // Optional: for LXC container
         "reason": "Monthly maintenance",
         "timeout_seconds": 300
       }

2. API CREATES REBOOT OPERATION
   ├─> Verify user owns agent
   ├─> Check no pending reboot exists
   ├─> Create reboot_operations record (status: pending)
   ├─> Set agent.pending_reboot_id
   └─> Status changes to "sent"

3. AGENT RECEIVES VIA REALTIME
   └─> Agent subscribes to reboot_operations collection
   └─> Receives standard PocketBase record event

4. AGENT ACKNOWLEDGES
   └─> POST /api/reboot/{id}/acknowledge
   └─> Status: "sent" → "rebooting"

5. AGENT EXECUTES REBOOT
   ├─> For host: systemctl reboot (or equivalent)
   └─> For LXC: pct reboot <vmid>

6. AGENT RECONNECTS
   └─> Agent metrics update last_seen

7. API MONITORS COMPLETION
   ├─> Background job checks every 30s
   ├─> If last_seen > acknowledged_at → completed
   ├─> If timeout exceeded → timeout
   └─> Clears agent.pending_reboot_id
```

### Reboot Statuses

| Status | Description |
|--------|-------------|
| `pending` | Reboot operation created, waiting to send |
| `sent` | Command sent via realtime, awaiting agent acknowledgment |
| `rebooting` | Agent acknowledged, reboot in progress |
| `completed` | Agent reconnected after reboot |
| `failed` | Reboot failed (agent reported error) |
| `timeout` | Agent did not reconnect within timeout period |

### API Endpoints

#### Create Reboot (User Only)

```bash
POST /api/reboot
Authorization: Bearer <user_token>

{
  "agent_id": "abc123xyz",
  "lxc_id": "lxc456",           # Optional
  "reason": "Security patches",  # Optional
  "timeout_seconds": 300         # Optional, default 300 (5 min)
}
```

**Response:**
```json
{
  "success": true,
  "reboot_id": "reb123",
  "status": "sent",
  "message": "Reboot command sent. Agent will receive via realtime subscription."
}
```

> **Security Note:** Only authenticated **users** can initiate reboots. Agents cannot reboot themselves via API.

#### List Reboots

```bash
GET /api/reboot?agent_id=abc123&status=completed
Authorization: Bearer <user_token>
```

#### Acknowledge Reboot (Agent Only)

```bash
POST /api/reboot/{id}/acknowledge
Authorization: Bearer <agent_token>
```

### Realtime Subscription for Agents

Agents must subscribe to `reboot_operations` collection changes for their `agent_id`. When a reboot is created, agents receive the standard PocketBase realtime event containing the record:

```json
{
  "action": "create",
  "record": {
    "id": "reb123",
    "agent_id": "abc123xyz",
    "lxc_id": "lxc456",
    "vmid": 100,
    "status": "sent",
    "reason": "Monthly maintenance",
    "timeout_seconds": 300,
    "requested_at": "2026-01-24T10:30:00Z"
  }
}
```

**Agent Implementation:**
1. Subscribe to realtime changes on `reboot_operations`
2. Filter for records where `agent_id` matches
3. On receiving a record with `status: "sent"`:
   - Call `POST /api/reboot/{id}/acknowledge`
   - Execute reboot command:
     - **Host:** `systemctl reboot` or `shutdown -r now`
     - **LXC:** `pct reboot <vmid>` (via Proxmox API or CLI)
4. After reboot, normal metrics ingestion updates `last_seen`
5. API auto-detects reconnection and marks complete

### Scheduled Reboots

Similar to patch scheduling, you can schedule recurring reboots via `reboot_schedules` collection.

**Features:**
- Standard cron expressions
- **Uniqueness Constraint**: One schedule per agent/LXC
- Automatic `reboot_operations` creation at scheduled time

**Creating a Reboot Schedule:**

```bash
POST /api/collections/reboot_schedules/records
Authorization: Bearer <user_token>

{
  "user_id": "<your_user_id>",
  "agent_id": "abc123host",
  "lxc_id": "lxc100",           // Optional: Omit for host
  "cron_expression": "0 4 * * 0", // Weekly Sunday 4 AM
  "reason": "Weekly maintenance window",
  "is_active": true
}
```

**Response includes:**
- `next_run_at`: Calculated next execution time
- `last_run_at`: Last execution (if any)

### Reboot Verification

The API automatically verifies successful reboots by:

1. **Setting `pending_reboot_id`** on agent when reboot created
2. **Monitoring `last_seen`** timestamp after acknowledgment
3. **Comparing timestamps**: If `last_seen > acknowledged_at`, reboot succeeded
4. **Clearing `pending_reboot_id`** and setting status to `completed`

**Timeout Handling:**
- Default timeout: 300 seconds (5 minutes)
- If agent doesn't reconnect within timeout:
  - Status set to `timeout`
  - `error_message`: "Agent did not reconnect within timeout period"
  - `pending_reboot_id` cleared

### Collections Reference

#### reboot_operations

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | relation | User who initiated |
| `agent_id` | relation | Target agent |
| `lxc_id` | relation | Optional LXC target |
| `status` | select | pending/sent/rebooting/completed/failed/timeout |
| `reason` | text | User-provided reason |
| `requested_at` | date | When reboot was requested |
| `acknowledged_at` | date | When agent acknowledged |
| `completed_at` | date | When agent reconnected |
| `error_message` | text | Error details if failed |
| `timeout_seconds` | number | Timeout in seconds |

#### reboot_schedules

| Field | Type | Description |
|-------|------|-------------|
| `user_id` | relation | Schedule owner |
| `agent_id` | relation | Target agent |
| `lxc_id` | relation | Optional LXC target |
| `cron_expression` | text | Standard cron format |
| `reason` | text | Reason for scheduled reboot |
| `next_run_at` | date | Next scheduled execution |
| `last_run_at` | date | Last execution time |
| `is_active` | bool | Enable/disable schedule |

---

## Best Practices

### 1. Always Dry-Run First
```text
# Test what will be updated
POST /api/patches {"mode": "dry-run"}

# Review output, then apply
POST /api/patches {"mode": "apply"}
```

### 2. Exception Management
- Document why packages are excluded
- Set expiration dates for temporary holds
- Review exceptions quarterly
- Remove obsolete exceptions

### 3. Monitoring
- Check patch operation status regularly
- Review stdout/stderr logs for errors
- Monitor agent health after patches
- Track patch success/failure rates

### 4. Staging Environment
- Test patches in staging first
- Use identical OS versions
- Verify application compatibility
- Roll out to production gradually

### 5. Backup Strategy
- Snapshot VMs before major updates
- Backup critical config files
- Document rollback procedures
- Test restore procedures

## Troubleshooting

### Script Validation Fails

**Error:** SHA-256 mismatch

**Causes:**
- Network interference/MITM
- Corrupted download
- Database corruption

**Solution:**
```text
# Re-download script
curl -H "Auth: Bearer $TOKEN" \
  http://api:8090/api/files/...

# Verify manually
sha256sum script.sh

# Report if persistent
```

### Package Manager Locked

**Error:** `Unable to acquire dpkg lock`

**Solution:**
- Wait for concurrent updates to finish
- Check for unattended-upgrades
- Kill stale apt processes

### Permission Denied

**Error:** `Permission denied`

**Cause:**
- Agent not running as root
- Sudo not configured

**Solution:**
- Run agent as root, or
- Configure passwordless sudo:
  ```bash
  nanny-agent ALL=(ALL) NOPASSWD: /usr/bin/apt-get, /usr/bin/dnf
  ```

### No Updates Available

**Exit Code:** 2

**Meaning:** System is up-to-date

**Action:** Normal, no action needed

## Security Considerations

### Script Integrity
- **SHA-256 verification mandatory**
- Agent aborts on hash mismatch
- Scripts signed by API
- Prevents code injection

### Privilege Escalation
- Scripts require root/sudo
- Read-only commands in dry-run
- Write operations only in apply mode
- No arbitrary command execution

### Audit Trail
- All operations logged
- User/agent attribution
- Timestamps recorded
- Stdout/stderr preserved

### Network Security
- HTTPS recommended for production
- Bearer token authentication
- Rate limiting enabled
- Input validation

## Proxmox Integration

For Proxmox host agents, see [PROXMOX.md](PROXMOX.md) for:
- Agentless LXC container patching
- Host-initiated updates
- Container snapshot integration

## Related Documentation

- [Architecture Guide](ARCHITECTURE.md): System design
- [API Reference](API_REFERENCE.md): Endpoint details
- [Security Policy](SECURITY.md): Security practices
