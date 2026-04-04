#!/bin/bash
# Save media file to Obsidian vault attachments directory
# Usage: save-media.sh <source_path> <title>
# Output: Obsidian embed syntax on success

set -euo pipefail

SOURCE="$1"
TITLE="${2:-media}"
VAULT="/home/pi/Obsidian-Vault"
ATTACH_DIR="$VAULT/attachments"

mkdir -p "$ATTACH_DIR"

if [ ! -f "$SOURCE" ]; then
    echo "ERROR: source file not found: $SOURCE" >&2
    exit 1
fi

# Determine extension from source file
EXT="${SOURCE##*.}"
DATE=$(date +%Y-%m-%d)

# Build destination filename (sanitize title)
SAFE_TITLE=$(echo "$TITLE" | tr -d '/:*?"<>|' | tr ' ' '-')
DEST_NAME="${DATE}-${SAFE_TITLE}.${EXT}"
DEST_PATH="$ATTACH_DIR/$DEST_NAME"

# Avoid overwrite by appending number
COUNTER=1
while [ -f "$DEST_PATH" ]; do
    DEST_NAME="${DATE}-${SAFE_TITLE}-${COUNTER}.${EXT}"
    DEST_PATH="$ATTACH_DIR/$DEST_NAME"
    COUNTER=$((COUNTER + 1))
done

cp "$SOURCE" "$DEST_PATH"
echo "![[attachments/$DEST_NAME]]"
