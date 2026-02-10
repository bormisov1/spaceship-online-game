#!/usr/bin/env bash
set -euo pipefail

REMOTE="root@151.247.209.112"
REMOTE_DIR="/home/claude/spaceship-online-game"
SERVICE="spaceship-server"

echo ">> Copying files..."
scp -rq client server "$REMOTE:$REMOTE_DIR/"

echo ">> Building and restarting..."
ssh "$REMOTE" "cd $REMOTE_DIR/server && go build -o /home/claude/$SERVICE . && systemctl restart $SERVICE"

echo ">> Done. https://spaceships.x.bormisov.com/"
