#!/bin/bash
# SUSE/openSUSE Zypper Update Script
# Supports dry-run mode and package exceptions
# Usage: ./suse-zypper-update.sh [--dry-run] [--exclude package1,package2,...]

set -e  # Exit on any error
set -o pipefail  # Exit on pipe failures

DRY_RUN=false
EXCLUDE_PACKAGES=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --exclude)
            EXCLUDE_PACKAGES="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== SUSE/openSUSE Zypper Update Script ==="
echo "Dry Run: $DRY_RUN"
echo "Excluded Packages: ${EXCLUDE_PACKAGES:-none}"
echo ""

# Function to lock packages
lock_packages() {
    if [ -n "$EXCLUDE_PACKAGES" ]; then
        echo "=== Locking excluded packages ==="
        IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
        for pkg in "${PACKAGES[@]}"; do
            pkg=$(echo "$pkg" | xargs)
            echo "Locking package: $pkg"
            if [ "$DRY_RUN" = false ]; then
                zypper al "$pkg" 2>/dev/null || true
            fi
        done
        echo ""
    fi
}

# Function to unlock packages (cleanup)
unlock_packages() {
    if [ -n "$EXCLUDE_PACKAGES" ]; then
        echo "=== Unlocking excluded packages (cleanup) ==="
        IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
        for pkg in "${PACKAGES[@]}"; do
            pkg=$(echo "$pkg" | xargs)
            echo "Unlocking package: $pkg"
            if [ "$DRY_RUN" = false ]; then
                zypper rl "$pkg" 2>/dev/null || true
            fi
        done
        echo ""
    fi
}

# Trap to ensure cleanup on exit
trap unlock_packages EXIT

# Lock excluded packages
lock_packages

# Refresh repositories
echo "=== Refreshing repositories ==="
if [ "$DRY_RUN" = false ]; then
    zypper refresh -q
else
    echo "[DRY RUN] Would run: zypper refresh"
fi
echo ""

# Check for updates
echo "=== Available Updates ==="
if [ "$DRY_RUN" = true ]; then
    UPDATES=$(zypper list-updates 2>/dev/null | grep "^v " || true)
else
    UPDATES=$(zypper list-updates 2>/dev/null | grep "^v " || true)
fi

if [ -z "$UPDATES" ]; then
    echo "No updates available"
    echo '{"updates_available": 0, "packages": []}'
    exit 0
fi

echo "$UPDATES"
echo ""

# Count updates
UPDATE_COUNT=$(echo "$UPDATES" | wc -l)
echo "=== Update Summary ==="
echo "Total packages to update: $UPDATE_COUNT"
echo ""

# Generate JSON output
echo "=== JSON Output (for UI parsing) ==="
echo "{"
echo "  \"updates_available\": $UPDATE_COUNT,"
echo "  \"packages\": ["

FIRST=true
while IFS= read -r line; do
    if [ -n "$line" ]; then
        PACKAGE=$(echo "$line" | awk '{print $3}')
        CURRENT_VERSION=$(echo "$line" | awk '{print $5}')
        NEW_VERSION=$(echo "$line" | awk '{print $7}')
        
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo ","
        fi
        
        echo -n "    {\"package\": \"$PACKAGE\", \"current_version\": \"$CURRENT_VERSION\", \"new_version\": \"$NEW_VERSION\"}"
    fi
done <<< "$UPDATES"

echo ""
echo "  ],"
echo "  \"dry_run\": $DRY_RUN"
echo "}"
echo ""

# Perform upgrade if not dry-run
if [ "$DRY_RUN" = false ]; then
    echo "=== Performing Update ==="
    zypper update -y --no-confirm
    echo "Update completed successfully"
else
    echo "=== Dry Run Complete ==="
    echo "No changes were made to the system"
fi

exit 0
