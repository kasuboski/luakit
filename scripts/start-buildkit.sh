#!/bin/bash
set -e

echo "=== Starting BuildKit daemon ==="

# Check if buildkitd container is already running
if docker ps --format '{{.Names}}' | grep -q '^luakit-buildkitd$'; then
    echo "BuildKit daemon already running"
    export BUILDKIT_HOST=tcp://127.0.0.1:1234
    exit 0
fi

# Check if container exists but stopped
if docker ps -a --format '{{.Names}}' | grep -q '^luakit-buildkitd$'; then
    echo "Starting existing BuildKit daemon container"
    docker start luakit-buildkitd
    export BUILDKIT_HOST=tcp://127.0.0.1:1234
    sleep 2
    exit 0
fi

# Start new BuildKit daemon container
echo "Starting new BuildKit daemon container..."
docker run -d \
    --name luakit-buildkitd \
    --privileged \
    -p 127.0.0.1:1234:1234 \
    moby/buildkit:latest \
    --addr tcp://0.0.0.0:1234

export BUILDKIT_HOST=tcp://127.0.0.1:1234

# Wait for BuildKit to be ready
echo "Waiting for BuildKit to be ready..."
for i in {1..30}; do
    if buildctl --addr="$BUILDKIT_HOST" du &>/dev/null; then
        echo "BuildKit daemon is ready"
        exit 0
    fi
    echo "Waiting... ($i/30)"
    sleep 1
done

echo "Error: BuildKit daemon failed to start"
exit 1
