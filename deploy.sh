#!/usr/bin/env bash
set -euo pipefail

REMOTE="root@151.247.209.112"
REMOTE_DIR="/home/claude/spaceship-online-game"
SERVICE="spaceship-server"
DEPLOY_REMOTE=false

for arg in "$@"; do
    case "$arg" in
        --remote) DEPLOY_REMOTE=true ;;
    esac
done

echo ">> Building Rust/WASM client..."
(cd client-rust && PATH="$HOME/.cargo/bin:$PATH" trunk build --release)

echo ">> Building server binary..."
(cd server && CGO_ENABLED=0 go build -buildvcs=false -o ../spaceship-server .)

if $DEPLOY_REMOTE; then
    echo ">> Stopping remote service..."
    ssh "$REMOTE" "systemctl stop $SERVICE"

    echo ">> Deploying binary..."
    scp -q spaceship-server "$REMOTE:$REMOTE_DIR/$SERVICE"
    rm -f spaceship-server

    echo ">> Copying Rust client dist..."
    ssh "$REMOTE" "mkdir -p $REMOTE_DIR/client-rust"
    scp -rq client-rust/dist "$REMOTE:$REMOTE_DIR/client-rust/"

    echo ">> Starting remote service..."
    ssh "$REMOTE" "systemctl start $SERVICE"
else
    CURRENT_USER="$(whoami)"
    if [ "$CURRENT_USER" != "claude" ]; then
        echo "Error: local deploy only works as user 'claude' (current: $CURRENT_USER). Use --remote to deploy remotely."
        exit 1
    fi

    echo ">> Stopping local service..."
    sudo systemctl stop "$SERVICE"

    echo ">> Installing binary..."
    if [ "$(pwd)" != "$REMOTE_DIR" ]; then
        cp spaceship-server "$REMOTE_DIR/$SERVICE"
        rm -f spaceship-server
    fi

    echo ">> Installing client dist..."
    mkdir -p "$REMOTE_DIR/client-rust"
    cp -r client-rust/dist "$REMOTE_DIR/client-rust/"

    echo ">> Starting local service..."
    sudo systemctl start "$SERVICE"
fi

echo ">> Done. https://spaceships.x.bormisov.com/"
