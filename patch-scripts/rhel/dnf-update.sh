#!/bin/bash
# RHEL/CentOS/Fedora DNF Update Script
# Supports dry-run mode and package exceptions
# Usage: ./rhel-dnf-update.sh [--dry-run] [--exclude package1,package2,...]

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

echo "=== RHEL/CentOS/Fedora DNF Update Script ==="
echo "Dry Run: $DRY_RUN"
echo "Excluded Packages: ${EXCLUDE_PACKAGES:-none}"
echo ""

# Build exclude options for dnf
EXCLUDE_OPTS=""
if [ -n "$EXCLUDE_PACKAGES" ]; then
    echo "=== Excluding packages ==="
    IFS=',' read -ra PACKAGES <<< "$EXCLUDE_PACKAGES"
    for pkg in "${PACKAGES[@]}"; do
        pkg=$(echo "$pkg" | xargs)
        echo "Excluding: $pkg"
        EXCLUDE_OPTS="$EXCLUDE_OPTS --exclude=$pkg"
    done
    echo ""
fi

# Check for available updates
echo "=== Checking for Updates ==="
if [ "$DRY_RUN" = true ]; then
    UPDATES=$(dnf check-update $EXCLUDE_OPTS -q 2>/dev/null | grep -v "^Last metadata" | grep -v "^$" || true)
else
    UPDATES=$(dnf check-update $EXCLUDE_OPTS -q 2>/dev/null | grep -v "^Last metadata" | grep -v "^$" || true)
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
    if [ -n "$line" ] && [[ ! "$line" =~ ^Obsoleting ]]; then
        PACKAGE=$(echo "$line" | awk '{print $1}')
        NEW_VERSION=$(echo "$line" | awk '{print $2}')
        REPO=$(echo "$line" | awk '{print $3}')
        CURRENT_VERSION=$(rpm -q --queryformat '%{VERSION}-%{RELEASE}' "$PACKAGE" 2>/dev/null || echo "unknown")
        
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo ","
        fi
        
        echo -n "    {\"package\": \"$PACKAGE\", \"current_version\": \"$CURRENT_VERSION\", \"new_version\": \"$NEW_VERSION\", \"repository\": \"$REPO\"}"
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
    dnf update -y $EXCLUDE_OPTS
    echo "Update completed successfully"
    echo ""
    
    # Show what was updated
    echo "=== Updated Packages ==="
    dnf history info last
else
    echo "=== Dry Run Complete ==="
    echo "No changes were made to the system"
fi

exit 0
