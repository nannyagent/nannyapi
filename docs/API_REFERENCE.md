# API Reference

This document provides a comprehensive guide to the NannyAPI endpoints.

## Base URL
`http://<your-server-ip>:8090/api`

## Authentication
Most endpoints require authentication.
- **User Auth**: Bearer Token (obtained via OAuth2 or Email/Password login).
- **Agent Auth**: Bearer Token (obtained via Device Flow).

---

## 1. Agent Management

### Device Authentication Start
Initiates the device authentication flow for an agent. This is the first step for an agent to register itself.

**Endpoint**: `POST /api/collections/device_codes/records` (Internal) or via custom handler `POST /api/agent/device-auth-start`

*Note: The actual endpoint depends on implementation. Based on handlers, it seems to be a custom route.*

**Endpoint**: `POST /api/agent` (Action: `device-auth-start`)

**Request Payload**:
```json
{
  "action": "device-auth-start"
}
```

**Response**:
```json
{
  "device_code": "xxxx-xxxx-xxxx",
  "user_code": "ABCD-1234",
  "verification_uri": "http://nanny.local/verify",
  "expires_in": 600
}
```

### Authorize Device
Used by the frontend/user to authorize a pending device code.

**Endpoint**: `POST /api/agent` (Action: `authorize`)
**Headers**: `Authorization: Bearer <USER_TOKEN>`

**Request Payload**:
```json
{
  "action": "authorize",
  "user_code": "ABCD-1234"
}
```

### Register Agent
Completes the registration process using the device code after user authorization.

**Endpoint**: `POST /api/agent` (Action: `register`)

**Request Payload**:
```json
{
  "action": "register",
  "device_code": "xxxx-xxxx-xxxx",
  "hostname": "prod-server-01",
  "platform": "linux",
  "platform_family": "debian",
  "os_version": "12",
  "arch": "amd64",
  "version": "1.0.0",
  "public_key": "ssh-rsa AAAAB3..."
}
```

**Response**:
```json
{
  "token": "eyJhbGciOiJIUzI1Ni...",
  "record": {
    "id": "agent_123",
    "hostname": "prod-server-01",
    "status": "active",
    "platform": "linux",
    "platform_family": "debian",
    "os_version": "12"
  }
}
```

---

## 2. Investigations

### Create Investigation
Initiates a new investigation for an agent. This triggers the agent to collect data.

**Endpoint**: `POST /api/collections/investigations/records`
**Headers**: `Authorization: Bearer <USER_TOKEN>`

**Request Payload**:
```json
{
  "agent_id": "agent_123",
  "issue": "High CPU usage observed on process 'java'",
  "priority": "high"
}
```

**Response**:
```json
{
  "id": "inv_456",
  "agent_id": "agent_123",
  "user_prompt": "High CPU usage observed on process 'java'",
  "status": "pending",
  "priority": "high",
  "created_at": "2023-10-27T10:00:00Z"
}
```

### Get Investigation Details
Retrieves details of a specific investigation, including the AI resolution plan.

**Endpoint**: `GET /api/collections/investigations/records/:id`
**Headers**: `Authorization: Bearer <USER_TOKEN>`

**Response**:
```json
{
  "id": "inv_456",
  "status": "completed",
  "resolution_plan": "The 'java' process was consuming 90% CPU due to a garbage collection loop. Recommended action: Restart the service.",
  "episode_id": "ep_789",
  "metadata": {
    "cpu_load": 95,
    "memory_usage": "8GB"
  }
}
```

---

## 3. Patch Management

### Create Patch Operation
Triggers a patch operation (dry-run or apply).

**Endpoint**: `POST /api/collections/patch_operations/records`
**Headers**: `Authorization: Bearer <USER_TOKEN>`

**Request Payload**:
```json
{
  "agent_id": "agent_123",
  "mode": "dry-run",
  "script_args": "--exclude pkg1,pkg2"
}
```
*Note: `mode` can be `dry-run` or `apply`. `script_args` can be used to pass exceptions.*

**Response**:
```json
{
  "id": "patch_789",
  "agent_id": "agent_123",
  "mode": "dry-run",
  "status": "pending",
  "script_url": "/api/files/scripts/script_123/apt-update.sh",
  "created_at": "2023-10-27T10:05:00Z"
}
```

### Get Patch Operation Status
Checks the status of a patch job.

**Endpoint**: `GET /api/collections/patch_operations/records/:id`
**Headers**: `Authorization: Bearer <USER_TOKEN>`

**Response**:
```json
{
  "id": "patch_789",
  "status": "completed",
  "output_path": "/storage/patches/patch_789.log",
  "error_msg": ""
}
```

---

## 4. Proxmox (Future/Planned)

*Note: Proxmox support is currently in development. The following endpoints are planned.*

### List Nodes
**Endpoint**: `GET /api/proxmox/nodes`

### List LXC Containers
**Endpoint**: `GET /api/proxmox/nodes/:node/lxc`

### Patch LXC Container (Host-Initiated)
Triggers a patch operation inside an LXC container from the host agent.

**Endpoint**: `POST /api/proxmox/patch/lxc`

**Request Payload**:
```json
{
  "node": "pve-01",
  "vmid": "100",
  "mode": "dry-run"
}
```

---

## 5. Agent Metrics

### Report Metrics
Agents report system metrics periodically.

**Endpoint**: `POST /api/collections/agent_metrics/records`
**Headers**: `Authorization: Bearer <AGENT_TOKEN>`

**Request Payload**:
```json
{
  "agent_id": "agent_123",
  "cpu_percent": 15.5,
  "memory_used": 4096,
  "memory_total": 16384,
  "disk_usage": 45.2
}
```
