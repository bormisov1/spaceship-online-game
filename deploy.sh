#!/usr/bin/env bash
set -euo pipefail

REMOTE="root@151.247.209.112"
REMOTE_DIR="/home/claude/spaceship-online-game"
SERVICE="spaceship-server"
BUILD_RUST=false

for arg in "$@"; do
    case "$arg" in
        --rust) BUILD_RUST=true ;;
    esac
done

if $BUILD_RUST; then
    echo ">> Building Rust/WASM client..."
    (cd client-rust && PATH="$HOME/.cargo/bin:$PATH" trunk build --release)
fi

echo ">> Building server binary..."
(cd server && CGO_ENABLED=0 go build -buildvcs=false -o ../spaceship-server .)

echo ">> Stopping service..."
ssh "$REMOTE" "systemctl stop $SERVICE"

echo ">> Deploying binary..."
scp -q spaceship-server "$REMOTE:$REMOTE_DIR/$SERVICE"
rm -f spaceship-server

if $BUILD_RUST; then
    echo ">> Copying Rust client dist..."
    ssh "$REMOTE" "mkdir -p $REMOTE_DIR/client-rust"
    scp -rq client-rust/dist "$REMOTE:$REMOTE_DIR/client-rust/"
fi

echo ">> Starting service..."
ssh "$REMOTE" "systemctl start $SERVICE"

echo ">> Done. https://spaceships.x.bormisov.com/"
