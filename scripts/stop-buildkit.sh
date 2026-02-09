#!/bin/bash
set -e

echo "=== Stopping BuildKit daemon ==="

if docker ps --format '{{.Names}}' | grep -q '^luakit-buildkitd$'; then
    echo "Stopping BuildKit daemon container..."
    docker stop luakit-buildkitd
    docker rm luakit-buildkitd
    echo "BuildKit daemon stopped"
else
    echo "BuildKit daemon not running"
fi
