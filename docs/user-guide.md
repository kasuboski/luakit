# User Guide

This guide covers the core concepts and practical usage of Luakit for building container images.

## Table of Contents

- [Core Concepts](#core-concepts)
- [Building Scripts](#building-scripts)
- [Multi-Stage Builds](#multi-stage-builds)
- [Caching](#caching)
- [Networking](#networking)
- [Security](#security)
- [Platform Support](#platform-support)

## Core Concepts

### States and Immutability

In Luakit, a **State** represents a filesystem state at a point in the build graph. The fundamental principle is **immutability**: every operation returns a new state, leaving the original unchanged.

```lua
local base = bk.image("alpine:3.19")
local with_git = base:run("apk add git")
local with_vim = base:run("apk add vim")

-- base is still the original alpine image
-- with_git contains git
-- with_vim contains vim
```

This design enables:

- **Parallel execution**: BuildKit can execute independent branches simultaneously
- **Caching**: Identical states are deduplicated
- **Reusability**: States can be referenced multiple times

### Fluent Interface

Operations can be chained:

```lua
local result = bk.image("alpine:3.19")
    :run("apk add --no-cache git")
    :mkdir("/app")
    :copy(bk.local_("context"), ".", "/app")
```

Each method returns a new state, so you can continue chaining.

### Source Operations

Source operations create initial states with no dependencies:

```lua
-- Pull a container image
local alpine = bk.image("alpine:3.19")

-- Empty filesystem
local empty = bk.scratch()

-- Local build context
local src = bk.local_("context")

-- Git repository
local repo = bk.git("https://github.com/user/project.git", { ref = "v1.0.0" })

-- HTTP download
local file = bk.http("https://example.com/file.tar.gz")
```

### Exec Operations

Execute commands on states:

```lua
-- String form (via shell)
local result = base:run("apt-get update && apt-get install -y curl")

-- Array form (exec directly)
local result = base:run({"apt-get", "update"})

-- With options
local result = base:run("make -j$(nproc)", {
    cwd = "/app",
    env = { CC = "gcc" },
    user = "builder"
})
```

### File Operations

Manipulate files without running commands:

```lua
-- Copy files between states
local result = base:copy(source, "/src", "/dst")

-- Create directory
local result = base:mkdir("/app/data", { mode = "0755" })

-- Create file
local result = base:mkfile("/app/config.json", '{"key":"value"}')

-- Remove files
local result = base:rm("/tmp/*", { allow_wildcard = true })

-- Create symlink
local result = base:symlink("/usr/bin/node", "/usr/local/bin/node")
```

### Graph Operations

Combine states in advanced ways:

```lua
-- Merge multiple states (overlay)
local merged = bk.merge(state1, state2, state3)

-- Extract differences
local diff = bk.diff(before, after)
```

## Building Scripts

### Script Structure

A typical build script:

```lua
-- 1. Define base image
local base = bk.image("golang:1.21-alpine")

-- 2. Add dependencies
local builder = base:run("apk add --no-cache git ca-certificates")

-- 3. Copy source code
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")

-- 4. Build application
local built = workspace:run("go build -o /out/app ./cmd/server", {
    cwd = "/app"
})

-- 5. Prepare runtime
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/app/server")

-- 6. Export final image
bk.export(final, {
    entrypoint = {"/app/server"},
    env = {"PORT=8080"},
    user = "app",
    workdir = "/app"
})
```

### Working Directory

Set the working directory per command:

```lua
local result = base:run("make", { cwd = "/app/src" })
```

Or set globally in export:

```lua
bk.export(final, { workdir = "/app" })
```

### Environment Variables

Set per command:

```lua
local result = base:run("go build", {
    env = {
        CGO_ENABLED = "0",
        GOOS = "linux"
    }
})
```

Or in export:

```lua
bk.export(final, {
    env = {
        NODE_ENV = "production",
        PORT = "8080"
    }
})
```

### User Management

Run commands as specific user:

```lua
local result = base:run("npm install", { user = "node" })
```

Set default user:

```lua
bk.export(final, { user = "app" })
```

Create user:

```lua
local with_user = base:run({
    "sh", "-c",
    "addgroup -g 1000 app && " ..
    "adduser -D -u 1000 -G app app"
})
```

## Multi-Stage Builds

Multi-stage builds use separate images for build and runtime phases, reducing final image size.

### Basic Pattern

```lua
-- Builder stage
local builder = bk.image("golang:1.21")
local src = bk.local_("context")
local built = builder:copy(src, ".", "/app")
    :run("go build -o /out/server ./cmd/server", { cwd = "/app" })

-- Runtime stage
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, {
    entrypoint = {"/server"},
    user = "nobody"
})
```

### Three-Stage Pattern

```lua
-- Stage 1: Dependencies
local deps = bk.image("node:20")
    :copy(bk.local_("context", { include = { "package*.json" } }), ".", "/app")
    :run("npm ci", { cwd = "/app", mounts = { bk.cache("/root/.npm") } })

-- Stage 2: Build
local built = deps:copy(bk.local_("context"), ".", "/app")
    :run("npm run build", { cwd = "/app", mounts = { bk.cache("/root/.npm") } })

-- Stage 3: Runtime
local runtime = bk.image("node:20-alpine")
local final = runtime:copy(built, "/app/node_modules", "/app/node_modules")
    :copy(built, "/app/dist", "/app/dist")
    :copy(built, "/app/package.json", "/app/package.json")

bk.export(final, {
    entrypoint = {"node", "dist/index.js"},
    cwd = "/app"
})
```

### Advantages

- **Smaller final images**: Build tools not included
- **Better layer caching**: Dependencies cached separately from source
- **Different bases**: Use heavy tools in builder, minimal runtime
- **Security**: Reduce attack surface

## Caching

Caching dramatically improves build times by reusing previous results.

### Cache Mounts

Cache frequently-used directories:

```lua
local result = base:run("go build ./...", {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
        bk.cache("/root/.cache/go-build", { sharing = "shared", id = "gobuild" })
    }
})
```

Cache mount options:

- `id`: Namespace for cache (default: directory path)
- `sharing`: "shared" (default), "private", "locked"

**Sharing modes:**

- `shared`: Multiple concurrent builds can use the cache
- `private`: Cache is exclusive to this build
- `locked`: Sequential builds, prevents conflicts

### Common Cache Paths

| Language | Cache Path | Purpose |
|----------|------------|---------|
| Go | `/go/pkg/mod` | Go modules |
| Go | `/root/.cache/go-build` | Build cache |
| Node.js | `/root/.npm` | npm packages |
| Node.js | `/root/.cache/yarn` | Yarn cache |
| Python | `/root/.cache/pip` | pip packages |
| Python | `~/.cache/pip` | pip packages (some distros) |
| Rust | `/usr/local/cargo/registry` | Cargo registry |
| Rust | `/usr/local/cargo/git` | Cargo git deps |

### Layer Caching Strategies

Order operations for maximum cache hits:

```lua
-- Good: Copy package files first, install dependencies, then copy source
local base = bk.image("node:20")
local pkg = base:copy(bk.local_("context", {
    include = { "package*.json" }
}), ".", "/app")
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})
local app = deps:copy(bk.local_("context"), ".", "/app")
local built = app:run("npm run build", { cwd = "/app" })

-- Bad: Copy everything at once - any change invalidates dependency layer
local base = bk.image("node:20")
local app = base:copy(bk.local_("context"), ".", "/app")
local deps = app:run("npm ci", { cwd = "/app" })
local built = app:run("npm run build", { cwd = "/app" })
```

## Networking

Control network access during builds.

### Network Modes

```lua
-- Default: sandboxed network (limited DNS, no internet)
local result = base:run("make", { network = "sandbox" })

-- Host network (full network access)
local result = base:run("curl https://api.example.com", { network = "host" })

-- No network (isolated)
local result = base:run("mise run test", { network = "none" })
```

**When to use:**

- `sandbox` (default): Most builds, controlled internet access
- `host`: Need to access external APIs or services
- `none`: Fully isolated builds, air-gapped environments

### Security Modes

```lua
-- Default: sandboxed (restricted)
local result = base:run("make", { security = "sandbox" })

-- Insecure (full privileges, equivalent to --privileged)
local result = base:run("dmesg", { security = "insecure" })
```

**When to use:**

- `sandbox` (default): All normal builds
- `insecure`: Only when absolutely necessary (e.g., dmesg, some kernel operations)

## Security

### Secrets

Mount secrets during builds:

```lua
local result = base:run({
    "sh", "-c",
    "npm config set registry https://registry.npmjs.org/ && " ..
    "cat /run/secrets/npmrc > ~/.npmrc && " ..
    "npm ci"
}, {
    mounts = {
        bk.secret("/run/secrets/npmrc", { id = "npmrc" })
    }
})
```

Provide secret to buildctl:

```bash
buildctl build \
  --local context=. \
  --secret id=npmrc,src=$HOME/.npmrc
```

Secret options:

- `id`: Secret identifier (default: destination basename)
- `uid`, `gid`: Owner UID/GID
- `mode`: File permissions (default: 0400)
- `optional`: Allow build to continue if secret missing

### SSH

Use SSH for git operations:

```lua
local repo = bk.git("git@github.com:user/private-repo.git")
local result = base:run("go mod download", {
    cwd = "/app",
    mounts = { bk.ssh({ id = "default" }) }
})
```

Provide SSH key to buildctl:

```bash
buildctl build \
  --local context=. \
  --ssh id=default
```

SSH options:

- `id`: SSH agent ID (default: "default")
- `dest`: Mount destination (default: /run/ssh)
- `uid`, `gid`: Owner
- `mode`: Permissions (default: 0600)
- `optional`: Allow build to continue if SSH unavailable

### Sandbox Restrictions

Luakit scripts run in a sandboxed Lua VM:

- `os.execute`: Disabled
- `io.open`: Disabled
- `loadfile`: Disabled
- `dofile`: Disabled
- `package.loadlib`: Disabled

Only `bk.*` API functions can modify the build graph.

## Platform Support

### Single Platform

Build for specific platform:

```lua
local arm = bk.image("alpine:3.19", {
    platform = "linux/arm64"
})

-- Or using table
local arm = bk.image("alpine:3.19", {
    platform = { os = "linux", arch = "arm64", variant = "v8" }
})
```

### Cross-Platform Builds

Create platform-aware builds:

```lua
local platforms = {
    bk.platform("linux", "amd64"),
    bk.platform("linux", "arm64"),
    bk.platform("linux", "arm", "v7")
}

for _, p in ipairs(platforms) do
    local base = bk.image("alpine:3.19", { platform = p })
    -- Build for each platform
end
```

Build for all platforms with buildctl:

```bash
buildctl build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  --local context=.
```

### Platform Strings

Common platform strings:

- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`
- `linux/386`
- `linux/ppc64le`
- `linux/s390x`

## Next Steps

- [API Reference](api-reference.md) - Complete function documentation
- [Tutorials](tutorials.md) - Hands-on examples
- [Best Practices](best-practices.md) - Production-ready patterns
- [Migration Guide](migration.md) - Converting from Dockerfile
