# Patch Management

NannyAPI provides a comprehensive patch management system that supports various operating systems and virtualization environments, including Proxmox.

## Workflow

The patching process is designed to be safe and controllable.

### 1. Dry-Run
Before applying any updates, the agent performs a dry-run.
- **Purpose**: To identify which packages *would* be updated without making any changes.
- **Output**: A list of upgradable packages, their current versions, and the new versions.
- **API**: The agent reports this data to the API, allowing admins to review pending updates.

### 2. Apply
When a patch job is executed (either manually or scheduled):
- The agent **does not** run arbitrary package manager commands.
- Instead, it downloads a verified **Patch Script** from the API (e.g., `apt-update.sh`, `dnf-update.sh`).
- It executes this script with specific arguments (e.g., `--dry-run` or `--apply`).
- **Package Outputs**: The full stdout/stderr of the script execution is captured and sent to the API.

### 3. Package Exceptions
If a specific package causes issues or needs to remain at a fixed version, you can add it to the **Package Exceptions** list for that agent.
- **Mechanism**: When the API triggers a patch job, it passes the list of excluded packages to the script (e.g., `--exclude pkg1,pkg2`).
- **Behavior**: The patch script is responsible for ensuring these packages are ignored during the update process (e.g., using `apt-mark hold` or excluding them from the transaction).

### 4. Schedules
Patching can be automated via schedules defined in the NannyAPI.
- Admins can define maintenance windows.
- Agents check for scheduled jobs and execute them within the window.

## Proxmox Support

For detailed information on Proxmox patching, including agentless LXC updates, please refer to the **[Proxmox Integration Guide](PROXMOX.md)**.

## Security & Integrity

### SHA256 Verification
To prevent Man-in-the-Middle (MITM) attacks and ensure script integrity:
- All patch scripts stored in the NannyAPI backend are hashed (SHA256).
- When an agent requests a script, it also receives the expected hash.
- The agent verifies the hash of the downloaded script before execution. If the hash does not match, execution is aborted immediately.

## Supported Distributions

NannyAPI supports the following Linux distributions and package managers:

| Distribution Family | Package Manager | Supported Versions |
|---------------------|-----------------|--------------------|
| **Debian/Ubuntu**   | `apt`           | Debian 10+, Ubuntu 20.04+ |
| **RHEL/CentOS**     | `dnf` / `yum`   | RHEL 8+, CentOS Stream 8+ |
| **Fedora**          | `dnf`           | Fedora 38+ |
| **SUSE/OpenSUSE**   | `zypper`        | SLES 15+, Leap 15+ |
| **Arch Linux**      | `pacman`        | Rolling |

## Patch Scripts

The system uses standardized shell scripts for each package manager to ensure consistent behavior. These scripts handle:
- Refreshing package lists.
- Listing upgradable packages (dry-run).
- Applying updates non-interactively.
- capturing exit codes and errors.
