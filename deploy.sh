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

echo ">> Copying files..."
scp -rq server "$REMOTE:$REMOTE_DIR/"

if $BUILD_RUST; then
    echo ">> Copying Rust client dist..."
    ssh "$REMOTE" "mkdir -p $REMOTE_DIR/client-rust"
    scp -rq client-rust/dist "$REMOTE:$REMOTE_DIR/client-rust/"
fi

echo ">> Building and restarting..."
ssh "$REMOTE" "cd $REMOTE_DIR/server && go build -buildvcs=false -o $REMOTE_DIR/$SERVICE . && systemctl restart $SERVICE"

echo ">> Done. https://spaceships.x.bormisov.com/"
if $BUILD_RUST; then
    echo ">>       https://spaceships.x.bormisov.com/rust/"
fi
