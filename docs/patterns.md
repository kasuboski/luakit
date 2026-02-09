# Common Patterns

Reusable patterns for common build scenarios in Luakit.

## Table of Contents

- [Multi-Stage Builds](#multi-stage-builds)
- [Parallel Builds](#parallel-builds)
- [Layer Optimization](#layer-optimization)
- [Reproducible Builds](#reproducible-builds)
- [Dependency Management](#dependency-management)
- [Configuration](#configuration)
- [Testing](#testing)

## Multi-Stage Builds

### Pattern 1: Simple Two-Stage

Build and runtime separation:

```lua
-- Builder stage
local builder = bk.image("golang:1.21")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/app .", { cwd = "/app" })

-- Runtime stage
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/app/server")

bk.export(final, {
    entrypoint = {"/app/server"}
})
```

### Pattern 2: Three-Stage with Dependencies

Separate dependencies, build, and runtime:

```lua
-- Stage 1: Install dependencies
local base = bk.image("node:20")
local pkg = base:copy(bk.local_("context", {
    include_patterns = { "package*.json" }
}), ".", "/app")
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Stage 2: Build application
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")
local built = with_src:run("npm run build", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Stage 3: Minimal runtime
local runtime = bk.image("node:20-alpine")
local final = runtime:copy(built, "/app/node_modules", "/app/node_modules")
    :copy(built, "/app/dist", "/app/dist")

bk.export(final, {
    entrypoint = {"node", "dist/index.js"}
})
```

### Pattern 3: Cross-Compilation

Build for different platforms:

```lua
-- Builder for multiple platforms
local platforms = {
    "linux/amd64",
    "linux/arm64",
    "linux/arm/v7"
}

for _, platform in ipairs(platforms) do
    local builder = bk.image("golang:1.21", { platform = platform })
    local built = builder:copy(src, ".", "/app")
        :run("CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app .", {
            cwd = "/app"
        })

    -- Platform-specific runtime
    local runtime = bk.image("alpine:3.19", { platform = platform })
    local final = runtime:copy(built, "/out/app", "/app/server")

    -- Export with platform tag
    bk.export(final, {
        entrypoint = {"/app/server"}
    })
end
```

### Pattern 4: Distroless Runtime

Minimal, secure runtime:

```lua
-- Builder with tools
local builder = bk.image("golang:1.21")
local src = bk.local_("context")
local built = builder:copy(src, ".", "/app")
    :run("go build -o /out/app .", { cwd = "/app" })

-- Distroless runtime
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/app", "/server")

bk.export(final, {
    entrypoint = {"/server"},
    user = "nobody"
})
```

---

## Parallel Builds

### Pattern 1: Independent Operations

Run independent operations in parallel:

```lua
local base = bk.image("node:20")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")

-- These run in parallel
local linted = workspace:run("npm run lint", { cwd = "/app" })
local tested = workspace:run("npm run test", { cwd = "/app" })
local built = workspace:run("npm run build", { cwd = "/app" })

-- Merge results
local verified = bk.merge(linted, tested, built)

bk.export(verified)
```

### Pattern 2: Parallel Multi-Platform

Build for multiple platforms simultaneously:

```bash
#!/bin/bash
for platform in linux/amd64 linux/arm64; do
    luakit build build.lua | buildctl build \
        --no-frontend \
        --local context=. \
        --platform $platform \
        -t myapp:$platform &
done
wait
```

### Pattern 3: Parallel Dependency Installation

Install multiple dependencies in parallel:

```lua
local base = bk.image("alpine:3.19")

-- Parallel installation
local with_git = base:run("apk add --no-cache git")
local with_vim = base:run("apk add --no-cache vim")
local with_curl = base:run("apk add --no-cache curl")

-- Merge all
local tools = bk.merge(with_git, with_vim, with_curl)
```

---

## Layer Optimization

### Pattern 1: Dependency Layer Separation

Copy dependency files first:

```lua
-- Copy go.mod and go.sum
local go_files = bk.local_("context", {
    include_patterns = { "go.mod", "go.sum" }
})
local with_go = base:copy(go_files, ".", "/app")

-- Download dependencies (cached)
local deps = with_go:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") }
})

-- Copy full source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("go build -o /out/app .", {
    cwd = "/app"
})
```

### Pattern 2: Single Layer for Related Operations

Combine related operations:

```lua
-- Good: Single layer
local result = base:run({
    "sh", "-c",
    "apt-get update && " ..
    "apt-get install -y --no-install-recommends curl git && " ..
    "rm -rf /var/lib/apt/lists/*"
})

-- Bad: Multiple layers
local layer1 = base:run("apt-get update")
local layer2 = layer1:run("apt-get install -y curl")
local layer3 = layer2:run("apt-get install -y git")
```

### Pattern 3: Chained Copy

Copy multiple files in sequence:

```lua
-- Copy package files first
local pkg = base:copy(bk.local_("context", {
    include_patterns = { "package*.json" }
}), ".", "/app")

-- Install dependencies
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("npm run build", { cwd = "/app" })
```

---

## Reproducible Builds

### Pattern 1: Pinned Versions

Use specific versions:

```lua
-- Good: Pinned
local base = bk.image("alpine:3.19.1")
local result = base:run("npm install express@4.18.2")

-- Bad: Floating
local base = bk.image("alpine:latest")
local result = base:run("npm install express")
```

### Pattern 2: Source Control Pins

Pin git refs:

```lua
-- Good: Specific commit
local repo = bk.git("https://github.com/user/project.git", {
    ref = "a1b2c3d4e5f6..."
})

-- Good: Specific tag
local repo = bk.git("https://github.com/user/project.git", {
    ref = "v1.2.3"
})

-- Bad: Branch
local repo = bk.git("https://github.com/user/project.git", {
    ref = "main"
})
```

### Pattern 3: Deterministic Build

Use deterministic build flags:

```lua
-- Go
local built = base:run({
    "go", "build",
    "-ldflags=-s -w -X main.Version=1.0.0",
    "-o", "/out/app", "."
}, {
    cwd = "/app"
})

-- Node.js
local built = base:run("npm run build", {
    cwd = "/app",
    env = { NODE_ENV = "production" }
})

-- Python
local built = base:run({
    "python", "-m", "PyInstaller",
    "--onefile",
    "--name", "app",
    "src/main.py"
}, {
    cwd = "/app"
})
```

### Pattern 4: Fixed Timestamps

Set fixed timestamps:

```lua
local result = base:run({
    "sh", "-c",
    "find /app -exec touch -t 202401010000 {} +"
})
```

---

## Dependency Management

### Pattern 1: Go Modules

Optimal Go caching:

```lua
local builder = bk.image("golang:1.21")

-- Copy go.mod and go.sum
local go_files = bk.local_("context", {
    include_patterns = { "go.mod", "go.sum" }
})
local with_go = builder:copy(go_files, ".", "/app")

-- Download dependencies
local deps = with_go:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod", { id = "gomod" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("go build -o /out/app .", {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})
```

### Pattern 2: Node.js npm

Optimal npm caching:

```lua
local base = bk.image("node:20")

-- Copy package files
local pkg = base:copy(bk.local_("context", {
    include_patterns = { "package*.json" }
}), ".", "/app")

-- Install dependencies
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("npm run build", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})
```

### Pattern 3: Python pip

Optimal pip caching:

```lua
local base = bk.image("python:3.11-slim")

-- Copy requirements
local req = base:copy(bk.local_("context", {
    include_patterns = { "requirements*.txt" }
}), ".", "/app")

-- Install dependencies
local deps = req:run("pip install --no-cache-dir -r requirements.txt", {
    cwd = "/app",
    mounts = { bk.cache("/root/.cache/pip", { id = "pip" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("python -m compileall src/", {
    cwd = "/app"
})
```

### Pattern 4: Rust Cargo

Optimal Cargo caching:

```lua
local builder = bk.image("rust:1.75-alpine")

-- Copy Cargo.toml and Cargo.lock
local cargo_files = bk.local_("context", {
    include_patterns = { "Cargo.toml", "Cargo.lock" }
})
local with_cargo = builder:copy(cargo_files, ".", "/app")

-- Download dependencies
local deps = with_cargo:run("cargo fetch", {
    cwd = "/app",
    mounts = {
        bk.cache("/usr/local/cargo/registry", { id = "cargo" }),
        bk.cache("/usr/local/cargo/git", { id = "cargo" })
    }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("cargo build --release", {
    cwd = "/app",
    mounts = {
        bk.cache("/usr/local/cargo/registry", { id = "cargo" }),
        bk.cache("/usr/local/cargo/git", { id = "cargo" }),
        bk.cache("/usr/local/cargo/target", { id = "build" })
    }
})
```

---

## Configuration

### Pattern 1: Environment Variables

Set environment variables:

```lua
-- In export
bk.export(final, {
    env = {
        NODE_ENV = "production",
        PORT = "8080",
        LOG_LEVEL = "info"
    }
})

-- In command
local result = base:run("go build", {
    env = {
        CGO_ENABLED = "0",
        GOOS = "linux",
        GOARCH = "amd64"
    }
})
```

### Pattern 2: Configuration Files

Create configuration files:

```lua
local base = bk.image("alpine:3.19")

-- Create config file
local with_config = base:mkfile("/etc/myapp/config.json", [[
{
    "server": {
        "port": 8080,
        "host": "0.0.0.0"
    },
    "database": {
        "host": "localhost",
        "port": 5432,
        "name": "mydb"
    }
}
]], {
    mode = "0644",
    owner = { user = "app", group = "app" }
})

bk.export(with_config)
```

### Pattern 3: Runtime vs Build Configuration

Separate build and runtime configs:

```lua
-- Build configuration
local build_env = {
    DEBUG = "false",
    OPTIMIZATION = "true"
}

-- Runtime configuration
local runtime_env = {
    NODE_ENV = "production",
    PORT = "8080"
}

local builder = bk.image("node:20")
local built = builder:copy(src, ".", "/app")
    :run("npm run build", {
        cwd = "/app",
        env = build_env
    })

local runtime = bk.image("node:20-alpine")
local final = runtime:copy(built, "/app/dist", "/app/dist")

bk.export(final, {
    env = runtime_env,
    entrypoint = {"node", "dist/index.js"}
})
```

---

## Testing

### Pattern 1: Test During Build

Run tests as part of build:

```lua
local base = bk.image("golang:1.21")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")

-- Download dependencies
local deps = workspace:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") }
})

-- Run tests
local tested = deps:run("go test ./...", {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod"),
        bk.cache("/root/.cache/go-build")
    }
})

-- Build
local built = tested:run("go build -o /out/app .", {
    cwd = "/app"
})

bk.export(built, {
    entrypoint = {"/app"}
})
```

### Pattern 2: Parallel Test Suites

Run multiple test suites in parallel:

```lua
local base = bk.image("node:20")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")

-- Install dependencies
local deps = workspace:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Parallel tests
local unit_tests = deps:run("npm run test:unit", {
    cwd = "/app",
    valid_exit_codes = {0}
})

local integration_tests = deps:run("npm run test:integration", {
    cwd = "/app",
    valid_exit_codes = {0}
})

local e2e_tests = deps:run("npm run test:e2e", {
    cwd = "/app",
    valid_exit_codes = {0}
})

-- Merge results
local all_passed = bk.merge(unit_tests, integration_tests, e2e_tests)

-- Build
local built = all_passed:run("npm run build", { cwd = "/app" })

bk.export(built)
```

### Pattern 3: Lint and Type Check

Include linting and type checking:

```lua
local base = bk.image("node:20")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")

-- Install dependencies
local deps = workspace:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Lint (allow 0 or 1 for certain linters)
local linted = deps:run("npm run lint", {
    cwd = "/app",
    valid_exit_codes = {0}
})

-- Type check
local typed = deps:run("npm run type-check", {
    cwd = "/app",
    valid_exit_codes = {0}
})

-- Merge lint and type-check results
local checked = bk.merge(linted, typed)

-- Build
local built = checked:run("npm run build", {
    cwd = "/app"
})

bk.export(built)
```

---

## Complete Examples

### Example 1: Production Go Microservice

```lua
-- Multi-stage Go build with all optimizations
local builder = bk.image("golang:1.21-alpine")
local builder_deps = builder:run({
    "apk", "add", "--no-cache", "git", "ca-certificates"
})

-- Copy dependency manifests
local go_files = bk.local_("context", {
    include_patterns = { "go.mod", "go.sum" }
})
local with_go = builder_deps:copy(go_files, ".", "/app")

-- Download dependencies
local deps = with_go:run({
    "go", "mod", "download"
}, {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod", { id = "gomod" }) }
})

-- Copy full source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build with optimizations
local built = with_src:run({
    "go", "build",
    "-ldflags=-s -w -X main.Version=1.0.0",
    "-trimpath",
    "-o", "/out/server",
    "./cmd/server"
}, {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})

-- Minimal runtime
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, {
    entrypoint = {"/server"},
    user = "nobody",
    expose = {"8080/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Go Microservice",
        ["org.opencontainers.image.version"] = "1.0.0"
    }
})
```

### Example 2: Production Node.js App

```lua
-- Multi-stage Node.js build with all optimizations
local builder = bk.image("node:20-slim")

-- Copy package files
local pkg = builder:copy(bk.local_("context", {
    include_patterns = { "package*.json" }
}), ".", "/app")

-- Install production dependencies
local deps = pkg:run({
    "npm", "ci", "--only=production"
}, {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build
local built = with_src:run("npm run build", {
    cwd = "/app",
    env = { NODE_ENV = "production" },
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

-- Minimal runtime
local runtime = bk.image("node:20-alpine")

-- Copy node_modules, dist, and package.json
local final = runtime:copy(built, "/app/node_modules", "/app/node_modules")
    :copy(built, "/app/dist", "/app/dist")
    :copy(built, "/app/package.json", "/app/package.json")

bk.export(final, {
    entrypoint = {"node", "dist/index.js"},
    cwd = "/app",
    user = "node",
    expose = {"3000/tcp"},
    env = {
        NODE_ENV = "production",
        PORT = "3000"
    }
})
```

### Example 3: Production Python Service

```lua
-- Multi-stage Python build with all optimizations
local builder = bk.image("python:3.11-slim")

-- Copy requirements
local req = builder:copy(bk.local_("context", {
    include_patterns = { "requirements*.txt" }
}), ".", "/app")

-- Install dependencies
local deps = req:run({
    "pip", "install", "--no-cache-dir", "-r", "requirements.txt"
}, {
    cwd = "/app",
    mounts = { bk.cache("/root/.cache/pip", { id = "pip" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Compile Python files
local compiled = with_src:run("python -m compileall -q src/", {
    cwd = "/app"
})

-- Minimal runtime
local runtime = bk.image("python:3.11-slim")
local final = runtime:copy(compiled, "/app", "/app")

bk.export(final, {
    entrypoint = {"python", "-m", "src.main"},
    cwd = "/app",
    user = "appuser",
    expose = {"8000/tcp"}
})
```

---

## Summary

These patterns provide:

- **Performance**: Optimized caching and parallel execution
- **Security**: Minimal images and proper user isolation
- **Reproducibility**: Pinned versions and deterministic builds
- **Maintainability**: Clear structure and reusable code

Apply these patterns to your builds to achieve production-ready results.
