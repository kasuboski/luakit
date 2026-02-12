# Integration Guide

How to integrate Luakit with BuildKit, Docker, CI/CD systems, and other tools.

## Table of Contents

- [With buildctl](#with-buildctl)
- [Gateway Mode](#gateway-mode)
- [With Docker](#with-docker)
- [CI/CD](#cicd)
- [Other Tools](#other-tools)

## With buildctl

Luakit outputs LLB protobuf that can be piped directly to `buildctl`.

### Basic Usage

```bash
luakit build build.lua | buildctl build --no-frontend --local context=.
```

### Detailed Command

```bash
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --local source=. \
    -t myapp:latest
```

### buildctl Options

#### --no-frontend

Disable default frontend (Dockerfile) and use provided LLB directly.

**Required when piping from luakit.**

#### --local name=<path>

Mount local directory into build context.

**Example:**

```bash
buildctl build \
  --no-frontend \
  --local context=/path/to/project \
  <(luakit build build.lua)
```

#### --output

Specify output format and destination.

**Formats:**

- `type=local,dest=<path>`: Export files to directory
- `type=image,name=<tag>,push=true`: Build and push image
- `type=docker`: Write Docker image to stdout
- `type=oci`: Write OCI image to stdout

**Examples:**

```bash
# Export to directory
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --output type=local,dest=./output

# Push to registry
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --output type=image,name=registry.example.com/myapp:latest,push=true

# Save as Docker image
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --output type=docker,name=myapp:latest | docker load
```

#### --secret

Provide secrets to the build.

**Syntax:** `--secret id=<id>,src=<path>`

**Example:**

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --secret id=npmrc,src=$HOME/.npmrc \
  --secret id=apikey,src=$HOME/.apikey
```

In script:

```lua
local result = base:run("npm install", {
    mounts = {
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.secret("/run/secrets/apikey", { id = "apikey" })
    }
})
```

#### --ssh

Provide SSH agent for git operations.

**Syntax:** `--ssh id=<id>`

**Example:**

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --ssh id=default
```

In script:

```lua
local repo = bk.git("git@github.com:user/private-repo.git")
local result = base:run("git submodule update", {
    mounts = { bk.ssh({ id = "default" }) }
})
```

#### --platform

Build for specific platform.

**Example:**

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --platform linux/arm64
```

#### --progress

Control progress output.

**Values:** `auto`, `plain`, `tty`, `quiet`

**Example:**

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --progress=tty
```

---

## Gateway Mode

Luakit can run as a BuildKit gateway frontend, allowing it to be invoked like any other frontend.

### Build the Gateway Image

```bash
# From Luakit repository
docker build -t luakit:gateway -f Dockerfile.gateway .
```

### Using Gateway Frontend

#### Via buildctl

```bash
buildctl build \
  --frontend gateway.v0 \
  --opt source=luakit:gateway \
  --local context=. \
  --local build.lua=build.lua
```

#### Via Docker

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=luakit:gateway \
  -f build.lua \
  -t myapp:latest
```

#### Via Dockerfile Syntax Directive

```dockerfile
#syntax=luakit:gateway
FROM alpine:3.19
RUN echo "Hello"
```

### Gateway Configuration

The gateway mode reads the build script from the build context:

```bash
# Mount build.lua as context
buildctl build \
  --frontend gateway.v0 \
  --opt source=luakit:gateway \
  --local context=. \
  --local build.lua=build.lua
```

### Frontend Options

Pass options to the frontend via `--opt`:

```bash
buildctl build \
  --frontend gateway.v0 \
  --opt source=luakit:gateway \
  --opt VERSION=1.0.0 \
  --local context=. \
  <(echo 'local version = os.getenv("VERSION")')
```

---

## With Docker

Use Luakit with Docker BuildKit.

### Docker Buildx

#### Basic Build

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  -t myapp:latest
```

#### Save to Local Docker Daemon

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  --load \
  -t myapp:latest
```

#### Push to Registry

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  --push \
  -t registry.example.com/myapp:latest
```

#### Multi-Platform Build

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  --platform linux/amd64,linux/arm64 \
  --push \
  -t registry.example.com/myapp:latest
```

### Using buildctl with Docker

```bash
# Build and load into Docker
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --output type=docker,name=myapp:latest | docker load

# Run
docker run --rm -p 8080:8080 myapp:latest
```

---

## CI/CD

### GitHub Actions

#### Basic Workflow

```yaml
name: Build and Push

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Set up BuildKit
        uses: docker/setup-buildx-action@v2

      - name: Cache BuildKit
        uses: actions/cache@v3
        with:
          path: /tmp/.buildx-cache
          key: ${{ runner.os }}-buildx-${{ github.sha }}
          restore-keys: ${{ runner.os }}-buildx-

      - name: Validate script
        run: |
          # Install luakit
          go install github.com/kasuboski/luakit/cmd/luakit@latest
          luakit validate build.lua

      - name: Build image
        run: |
          luakit build build.lua | \
          buildctl build \
            --no-frontend \
            --local context=. \
            --output type=image,name=docker.io/${{ github.repository }}:latest,push=true

      - name: Build and push multi-platform
        run: |
          luakit build build.lua | \
          buildctl build \
            --no-frontend \
            --local context=. \
            --platform linux/amd64,linux/arm64 \
            --output type=image,name=docker.io/${{ github.repository }}:latest,push=true
```

#### With Secrets

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Build with secrets
        env:
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
        run: |
          echo "$NPM_TOKEN" > .npmrc
          luakit build build.lua | \
          buildctl build \
            --no-frontend \
            --local context=. \
            --secret id=npmrc,src=$(pwd)/.npmrc \
            --output type=image,name=myapp:latest,push=true
```

### GitLab CI

#### Basic Pipeline

```yaml
image: docker:latest

services:
  - docker:dind

variables:
  DOCKER_DRIVER: overlay2
  DOCKER_TLS_CERTDIR: ""

stages:
  - build

build:
  stage: build
  before_script:
    - apk add --no-cache go
    - go install github.com/kasuboski/luakit/cmd/luakit@latest
  script:
    - luakit validate build.lua
    - luakit build build.lua | buildctl build \
        --no-frontend \
        --local context=. \
        --output type=image,name=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA,push=true
    - luakit build build.lua | buildctl build \
        --no-frontend \
        --local context=. \
        --output type=image,name=$CI_REGISTRY_IMAGE:latest,push=true
```

### Jenkins

#### Pipeline

```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                sh '''
                    # Install luakit
                    go install github.com/kasuboski/luakit/cmd/luakit@latest

                    # Validate
                    luakit validate build.lua

                    # Build
                    luakit build build.lua | buildctl build \
                        --no-frontend \
                        --local context=. \
                        -t myapp:${BUILD_NUMBER}
                '''
            }
        }
        stage('Test') {
            steps {
                sh '''
                    # Run tests in built image
                    docker run --rm myapp:${BUILD_NUMBER} npm test
                '''
            }
        }
        stage('Push') {
            when {
                branch 'main'
            }
            steps {
                sh '''
                    docker tag myapp:${BUILD_NUMBER} myapp:latest
                    docker push myapp:latest
                '''
            }
        }
    }
}
```

### CircleCI

```yaml
version: 2.1

jobs:
  build:
    docker:
      - image: docker:latest
    steps:
      - checkout
      - setup_remote_docker
      - run:
          name: Install luakit
          command: |
            apk add --no-cache go
            go install github.com/kasuboski/luakit/cmd/luakit@latest
      - run:
          name: Validate
          command: luakit validate build.lua
      - run:
          name: Build
          command: |
            luakit build build.lua | buildctl build \
              --no-frontend \
              --local context=. \
              -t myapp:${CIRCLE_SHA1}
      - run:
          name: Push
          command: |
            if [ "$CIRCLE_BRANCH" = "main" ]; then
              docker tag myapp:${CIRCLE_SHA1} myapp:latest
              docker push myapp:latest
            fi
```

---

## Other Tools

### With Skaffold

```yaml
# skaffold.yaml
apiVersion: skaffold/v2beta28
kind: Config
build:
  local:
    push: false
  artifacts:
    - image: myapp
      build:
        type: custom
        buildArgs:
          - buildctl build
          - --no-frontend
          - --local context=.
          - --output type=docker,name=myapp
          - <(luakit build build.lua)
deploy:
  kubectl:
    manifests:
      - k8s/*.yaml
```

### With Tilt

```python
# Tiltfile
local('go install github.com/kasuboski/luakit/cmd/luakit@latest')

def build_lua(name, script):
    local('luakit build {} | buildctl build \
        --no-frontend \
        --local context=. \
        --output type=docker,name={}'.format(script, name))
    return local_image(name)

docker_build('myapp', build_lua('myapp', 'build.lua'))
k8s_yaml('k8s/*.yaml')
```

### With Packer

```hcl
# build.pkr.hcl
source "docker" "example" {
  image = "alpine:3.19"

  # Use luakit to build
  changes = [
    "COPY (from luakit) /app /app"
  ]

  # Pre-build with luakit
  provisioner "shell-local" {
    inline = [
      "go install github.com/kasuboski/luakit/cmd/luakit@latest",
      "luakit build build.lua | buildctl build --no-frontend --local context=. -o temp.tar",
      "docker load < temp.tar"
    ]
  }
}
```

### With Nix

```nix
# default.nix
{ pkgs ? import <nixpkgs> {} }:
pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    buildkit
  ];

  shellHook = ''
    # Install luakit
    if ! command -v luakit &> /dev/null; then
      go install github.com/kasuboski/luakit/cmd/luakit@latest
    fi

    export PATH=$GOPATH/bin:$PATH
  '';
}
```

---

## Best Practices for Integration

### 1. Use Fixed Versions

Pin luakit version:

```bash
go install github.com/kasuboski/luakit/cmd/luakit@v0.1.0
```

### 2. Cache Dependencies

Cache BuildKit layers:

```bash
buildctl build \
  --no-frontend \
  --local context=. \
  --export-cache type=registry,ref=registry.example.com/myapp:cache \
  --import-cache type=registry,ref=registry.example.com/myapp:cache
```

### 3. Validate Before Build

Always validate scripts:

```bash
luakit validate build.lua && \
  luakit build build.lua | buildctl build ...
```

### 4. Use Multi-Platform Builds

Build for multiple architectures:

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --platform linux/amd64,linux/arm64,linux/arm/v7
```

### 5. Separate Contexts

Keep build context separate from source:

```bash
buildctl build \
  --no-frontend \
  --local context=/app/src \
  --local cache=/app/cache \
  <(luakit build build.lua)
```

---

## Troubleshooting Integration

### BuildKit Connection Issues

```bash
# Check BuildKit status
buildctl debug workers

# Start BuildKit daemon
buildkitd

# Use specific socket
export BUILDKIT_HOST=unix:///run/buildkit/buildkitd.sock
```

### Permission Issues

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# New login required
newgrp docker
```

### Cache Not Working

```bash
# Clear cache
buildctl prune

# Verify cache mount paths
luakit dag build.lua | grep cache
```

### Gateway Frontend Not Found

```bash
# Check frontend image
docker images | grep luakit

# Pull frontend
docker pull luakit:gateway

# Build gateway
docker build -t luakit:gateway -f Dockerfile.gateway .
```

---

## Advanced Patterns

### Pattern 1: Build Matrix

```bash
#!/bin/bash
for platform in linux/amd64 linux/arm64 linux/arm/v7; do
    for version in 1.0 1.1 1.2; do
        luakit build build.lua | buildctl build \
            --no-frontend \
            --local context=. \
            --platform $platform \
            -t myapp:$version-$platform
    done
done
```

### Pattern 2: Conditional Secrets

```bash
#!/bin/bash
SECRETS=""
if [ -f .env ]; then
    SECRETS="--secret id=env,src=.env"
fi

luakit build build.lua | buildctl build \
    --no-frontend \
    --local context=. \
    $SECRETS
```

### Pattern 3: Incremental Builds

```bash
# Export cache
luakit build build.lua | buildctl build \
    --no-frontend \
    --local context=. \
    --export-cache type=local,dest=/tmp/buildkit-cache

# Import cache
luakit build build.lua | buildctl build \
    --no-frontend \
    --local context=. \
    --import-cache type=local,src=/tmp/buildkit-cache
```

---

## Summary

Luakit integrates seamlessly with:

- **buildctl**: Native BuildKit CLI
- **Docker BuildKit**: Via gateway frontend
- **CI/CD**: GitHub Actions, GitLab CI, Jenkins, CircleCI
- **Dev Tools**: Skaffold, Tilt, Packer, Nix

Key integration points:

1. Pipe LLB to `buildctl build --no-frontend`
2. Use gateway mode for Docker integration
3. Validate scripts before building
4. Use cache for faster CI/CD
5. Support multi-platform builds
