#!/bin/bash
# Debian/Ubuntu APT Update Script
# Supports dry-run mode and package exceptions
# Usage: ./debian-apt-update.sh [--dry-run] [--exclude package1,package2,...]

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

echo "=== Debian/Ubuntu APT Update Script ==="
echo "Dry Run: $DRY_RUN"
echo "Excluded Packages: ${EXCLUDE_PACKAGES:-none}"
echo ""

# Function to hold/mark packages
hold_packages() {
    if [ -n "$EXCLUDE_PACKAGES" ]; then
        echo "=== Holding excluded packages ==="
        IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
        for pkg in "${PACKAGES[@]}"; do
            pkg=$(echo "$pkg" | xargs)  # Trim whitespace
            if dpkg -l | grep -q "^ii  $pkg "; then
                echo "Holding package: $pkg"
                if [ "$DRY_RUN" = false ]; then
                    echo "$pkg hold" | dpkg --set-selections
                fi
            else
                echo "Package not installed (skipping hold): $pkg"
            fi
        done
        echo ""
    fi
}

# Function to unhold packages (cleanup)
unhold_packages() {
    if [ -n "$EXCLUDE_PACKAGES" ]; then
        echo "=== Unholding excluded packages (cleanup) ==="
        IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
        for pkg in "${PACKAGES[@]}"; do
            pkg=$(echo "$pkg" | xargs)
            if dpkg -l | grep -q "^hi  $pkg "; then
                echo "Unholding package: $pkg"
                if [ "$DRY_RUN" = false ]; then
                    echo "$pkg install" | dpkg --set-selections
                fi
            fi
        done
        echo ""
    fi
}

# Trap to ensure cleanup on exit
trap unhold_packages EXIT

# Hold excluded packages
hold_packages

# Update package lists
echo "=== Updating package lists ==="
if [ "$DRY_RUN" = false ]; then
    apt-get update -qq
else
    echo "[DRY RUN] Would run: apt-get update"
fi
echo ""

# Show available updates
echo "=== Available Updates ==="
# Use a temporary variable to capture output and exit code
TEMP_UPDATES=$(apt list --upgradable 2>/dev/null || true)

# Check if updates are available
if echo "$TEMP_UPDATES" | grep -q upgradable; then
    UPDATES=$(echo "$TEMP_UPDATES" | grep -v "^Listing")
else
    UPDATES=""
fi

if [ -z "$UPDATES" ]; then
    echo "No updates available"
    echo '{"updates_available": 0, "packages": []}'
    exit 0
fi

echo "$UPDATES"
echo ""

# Count and parse updates
UPDATE_COUNT=$(echo "$UPDATES" | wc -l)
echo "=== Update Summary ==="
echo "Total packages to update: $UPDATE_COUNT"
echo ""

# Generate JSON output for UI
echo "=== JSON Output (for UI parsing) ==="
echo "{"
echo "  \"updates_available\": $UPDATE_COUNT,"
echo "  \"packages\": ["

FIRST=true
while IFS= read -r line; do
    if [ -n "$line" ]; then
        PACKAGE=$(echo "$line" | awk '{print $1}' | cut -d'/' -f1)
        NEW_VERSION=$(echo "$line" | awk '{print $2}')
        CURRENT_VERSION=$(dpkg-query -W -f='${Version}' "$PACKAGE" 2>/dev/null || echo "unknown")
        
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
    echo "=== Performing Upgrade ==="
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -qq
    echo "Upgrade completed successfully"
    echo ""
    
    # Show what was upgraded
    echo "=== Upgraded Packages ==="
    if [ -f /var/log/dpkg.log ]; then
        grep " upgrade " /var/log/dpkg.log | tail -20
    else
        echo "No dpkg.log found"
    fi
else
    echo "=== Dry Run Complete ==="
    echo "No changes were made to the system"
fi

exit 0