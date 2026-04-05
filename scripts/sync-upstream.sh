#!/bin/bash
# Auto-sync upstream (sipeed/picoclaw) into fork (TK-CamBaz/picoclaw)
# Aborts merge on conflict to avoid breaking local changes

set -e

cd /home/pi/picoclaw

git fetch upstream

# Only merge if there are new commits
LOCAL=$(git rev-parse main)
REMOTE=$(git rev-parse upstream/main)

if [ "$LOCAL" = "$REMOTE" ]; then
    echo "$(date): already up to date"
    exit 0
fi

if git merge upstream/main --no-edit; then
    git push origin main
    echo "$(date): synced to $(git rev-parse --short HEAD)"
else
    git merge --abort
    echo "$(date): merge conflict, aborted"
    exit 1
fi
