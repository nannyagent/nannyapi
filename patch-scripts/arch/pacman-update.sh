#!/bin/bash
# Arch Linux Pacman Update Script
# Supports dry-run mode and package exceptions
# Usage: ./arch-pacman-update.sh [--dry-run] [--exclude package1,package2,...]

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

echo "=== Arch Linux Pacman Update Script ==="
echo "Dry Run: $DRY_RUN"
echo "Excluded Packages: ${EXCLUDE_PACKAGES:-none}"
echo ""

# Build ignore options
IGNORE_OPTS=""
if [ -n "$EXCLUDE_PACKAGES" ]; then
    echo "=== Excluding packages ==="
    IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
    for pkg in "${PACKAGES[@]}"; do
        pkg=$(echo "$pkg" | xargs)
        echo "Excluding: $pkg"
        IGNORE_OPTS="$IGNORE_OPTS --ignore $pkg"
    done
    echo ""
fi

# Sync package databases
echo "=== Syncing package databases ==="
if [ "$DRY_RUN" = false ]; then
    pacman -Sy --noconfirm $IGNORE_OPTS
else
    echo "[DRY RUN] Would run: pacman -Sy"
fi
echo ""

# Check for updates
echo "=== Available Updates ==="
UPDATES=$(pacman -Qu $IGNORE_OPTS 2>/dev/null || true)

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
        PACKAGE=$(echo "$line" | awk '{print $1}')
        CURRENT_VERSION=$(echo "$line" | awk '{print $2}')
        NEW_VERSION=$(echo "$line" | awk '{print $4}')
        
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
    pacman -Su --noconfirm $IGNORE_OPTS
    echo "Update completed successfully"
    echo ""
    
    # Show what was updated
    echo "=== Updated Packages ==="
    tail -50 /var/log/pacman.log | grep "upgraded"
else
    echo "=== Dry Run Complete ==="
    echo "No changes were made to the system"
fi

exit 0
