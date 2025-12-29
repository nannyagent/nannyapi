#!/bin/sh

# Alpine Linux Patch Script (apk)
# Supports dry-run and apply modes

MODE=$1

if [ -z "$MODE" ]; then
    echo "Usage: $0 <dry-run|apply>"
    exit 1
fi

# Update package index
apk update > /dev/null 2>&1

if [ "$MODE" = "dry-run" ]; then
    echo "Checking for available updates..."
    # apk upgrade --simulate returns exit code 0 if successful
    # We want to list upgradable packages.
    # apk version -l '<' shows packages that have updates
    UPDATES=$(apk version -l '<')
    
    if [ -z "$UPDATES" ]; then
        echo "No updates available."
        # Output JSON for parsing
        echo '{"updates": []}'
    else
        # Format output as JSON
        # apk version output format: "package-1.2.3-r0 < 1.2.3-r1"
        # We need to parse this.
        echo "Found updates:"
        echo "$UPDATES"
        
        # Construct JSON manually
        JSON="["
        FIRST=1
        # Process line by line
        echo "$UPDATES" | while read -r line; do
            if [ -n "$line" ]; then
                # Extract package name, current version, new version
                # Example: "musl-1.2.3-r0 < 1.2.3-r1"
                # This is tricky with shell.
                # Let's try a simpler approach with awk if available, or just basic string manipulation
                PKG_NAME=$(echo "$line" | awk '{print $1}' | sed 's/-[0-9].*//')
                CURRENT_VER=$(echo "$line" | awk '{print $1}' | sed 's/.*-\([0-9].*\)/\1/')
                NEW_VER=$(echo "$line" | awk '{print $3}')
                
                if [ "$FIRST" -eq 0 ]; then
                    JSON="$JSON,"
                fi
                JSON="$JSON{\"name\": \"$PKG_NAME\", \"current_version\": \"$CURRENT_VER\", \"version\": \"$NEW_VER\"}"
                FIRST=0
            fi
        done
        JSON="$JSON]"
        echo "$JSON" > updates.json
        cat updates.json
    fi
    
elif [ "$MODE" = "apply" ]; then
    echo "Applying updates..."
    # apk upgrade --interactive is not good for automation.
    # apk upgrade --available might be too aggressive?
    # Just apk upgrade
    apk upgrade --no-cache
    
    if [ $? -eq 0 ]; then
        echo "Updates applied successfully."
        echo '{"status": "success"}'
    else
        echo "Failed to apply updates."
        echo '{"status": "failure"}'
        exit 1
    fi
else
    echo "Invalid mode: $MODE"
    exit 1
fi
