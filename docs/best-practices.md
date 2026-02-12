# Best Practices

Production-ready patterns and recommendations for Luakit builds.

## Table of Contents

- [Performance](#performance)
- [Security](#security)
- [Maintainability](#maintainability)
- [Reliability](#reliability)
- [Testing](#testing)

## Performance

### 1. Optimize Layer Caching

Order operations for maximum cache reuse:

```lua
-- Good: Dependency files first
local pkg = base:copy(bk.local_("context", {
    include = { "package*.json" }
}), ".", "/app")
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})
local app = deps:copy(bk.local_("context"), ".", "/app")

-- Bad: Everything at once
local app = base:copy(bk.local_("context"), ".", "/app")
local deps = app:run("npm ci", { cwd = "/app" })
```

**Rule of thumb:** Copy files that change less frequently first.

### 2. Use Cache Mounts

Dramatically improve dependency installation:

```lua
-- Go
local result = base:run("go build ./...", {
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})

-- Node.js
local result = base:run("npm ci", {
    mounts = { bk.cache("/root/.npm", { id = "npm" })
})

-- Python
local result = base:run("pip install -r requirements.txt", {
    mounts = { bk.cache("/root/.cache/pip", { id = "pip" })}
})
```

**Common cache paths:**

| Language | Path | ID |
|----------|-------|-----|
| Go | `/go/pkg/mod` | `gomod` |
| Go | `/root/.cache/go-build` | `gobuild` |
| Node.js | `/root/.npm` | `npm` |
| Node.js | `/root/.cache/yarn` | `yarn` |
| Python | `/root/.cache/pip` | `pip` |
| Rust | `/usr/local/cargo/registry` | `cargo` |

### 3. Minimize Image Size

Use multi-stage builds:

```lua
-- Builder stage with all tools
local builder = bk.image("golang:1.21")
local built = builder:copy(src, ".", "/app")
    :run("go build -o /out/app .", { cwd = "/app" })

-- Runtime stage with only binary
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/app")
```

Additional tips:

- Use minimal base images (`alpine`, `scratch`, `distroless`)
- Remove package manager caches:
  ```lua
  local result = base:run({
      "apt-get", "update",
      "&&", "apt-get", "install", "-y", "curl",
      "&&", "rm", "-rf", "/var/lib/apt/lists/*"
  })
  ```
- Use `.dockerignore` equivalent patterns in `local_()`:
  ```lua
  bk.local_("context", {
      exclude = { ".git/", "node_modules/", "*.md" }
  })
  ```

### 4. Use Array Form for Commands

Avoid shell overhead:

```lua
-- Good: Direct execution
local result = base:run({"npm", "install"})

-- Avoid: Shell parsing
local result = base:run("npm install")
```

**When to use shell form:**

- Command chaining (`&&`, `||`, `;`)
- Shell builtins (`cd`, `export`)
- Pipes and redirections
- Glob patterns

### 5. Parallelize Independent Operations

BuildKit executes independent branches in parallel:

```lua
-- These run in parallel
local lint = base:run("npm run lint", { cwd = "/app" })
local test = base:run("npm run test", { cwd = "/app" })
local typecheck = base:run("npm run typecheck", { cwd = "/app" })

-- Merge results
local verified = bk.merge(lint, test, typecheck)
```

## Security

### 1. Use Non-Root Users

Don't run as root:

```lua
-- Create user
local with_user = base:run({
    "sh", "-c",
    "groupadd -r app && " ..
    "useradd -r -g app -u 1000 app && " ..
    "chown -R app:app /app"
})

-- Set default user
bk.export(final, { user = "app" })
```

### 2. Minimize Attack Surface

Use minimal base images:

```lua
-- Good: Minimal Alpine
local base = bk.image("alpine:3.19")

-- Good: Distroless (no shell)
local base = bk.image("gcr.io/distroless/static-debian12")

-- Better: Scratch (empty)
local base = bk.scratch()
```

### 3. Use Secrets for Sensitive Data

Never hardcode credentials:

```lua
-- Good: Use secret
local result = base:run({
    "sh", "-c",
    "cat /run/secrets/password | docker login -u user --password-stdin"
}, {
    mounts = { bk.secret("/run/secrets/password", { id = "password" }) }
})

-- Bad: Hardcoded
local result = base:run("docker login -u user -p secret123")
```

Provide secrets securely:

```bash
buildctl build \
  --secret id=password,src=$HOME/.docker/password
```

### 4. Use Read-Only Filesystems

Mount caches as read-only where possible:

```lua
local result = base:run("go build", {
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.bind(src_state, "/src", { readonly = true })
    }
})
```

### 5. Network Isolation

Use appropriate network modes:

```lua
-- Default: Limited network
local result = base:run("mise run build", { network = "sandbox" })

-- Only when needed: Full network
local result = base:run("curl https://api.example.com", { network = "host" })

-- Air-gapped: No network
local result = base:run("mise run test", { network = "none" })
```

### 6. Pin Versions

Use specific versions:

```lua
-- Good: Pinned
local base = bk.image("alpine:3.19.1")
local result = base:run("npm install express@4.18.2")

-- Bad: Latest
local base = bk.image("alpine:latest")
local result = base:run("npm install express")
```

## Maintainability

### 1. Use Descriptive Variable Names

Make scripts self-documenting:

```lua
-- Good
local base_image = bk.image("alpine:3.19")
local with_git = base_image:run("apk add --no-cache git")
local workspace = with_git:mkdir("/app")
local with_source = workspace:copy(src, ".", "/app")

-- Bad
local a = bk.image("alpine:3.19")
local b = a:run("apk add --no-cache git")
local c = b:mkdir("/app")
local d = c:copy(src, ".", "/app")
```

### 2. Break Down Complex Builds

Use helper functions or modules:

```lua
-- helpers.lua
local M = {}

function M.install_go_deps(base)
    return base:run("go mod download", {
        cwd = "/app",
        mounts = { bk.cache("/go/pkg/mod") }
    })
end

function M.build_go_app(base)
    return base:run("go build -o /out/app ./cmd/server", {
        cwd = "/app",
        mounts = {
            bk.cache("/go/pkg/mod"),
            bk.cache("/root/.cache/go-build")
        }
    })
end

return M

-- build.lua
local helpers = require("helpers")
local base = bk.image("golang:1.21")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")
local deps = helpers.install_go_deps(workspace)
local built = helpers.build_go_app(deps)
```

### 3. Group Related Operations

Use fluent interface for clarity:

```lua
-- Good: Grouped operations
local final = bk.image("alpine:3.19")
    :run("apk add --no-cache curl")
    :run("addgroup -g 1000 app && adduser -D -u 1000 -G app app")
    :mkdir("/app", { mode = "0755" })
    :copy(built, "/out/server", "/app/server")

-- Bad: Disconnected operations
local a1 = bk.image("alpine:3.19")
local a2 = a1:run("apk add --no-cache curl")
local a3 = a2:run("addgroup -g 1000 app")
local a4 = a3:run("adduser -D -u 1000 -G app app")
```

### 4. Add Comments

Document non-obvious choices:

```lua
-- Use alpine 3.19 for OpenSSL 3.0 compatibility
local base = bk.image("alpine:3.19")

-- Copy go.mod first for better cache hit rate
local go_files = base:copy(bk.local_("context", {
    include = { "go.mod", "go.sum" }
}), ".", "/app")

-- Shared cache for faster parallel builds
local deps = go_files:run("go mod download", {
    mounts = { bk.cache("/go/pkg/mod", { sharing = "shared" }) }
})
```

### 5. Use Standardized Structure

Follow consistent patterns:

```lua
-- 1. Define images
local builder = bk.image("golang:1.21")
local runtime = bk.image("alpine:3.19")

-- 2. Configure builder
local builder_configured = builder:run("apk add --no-cache git")

-- 3. Prepare source
local src = bk.local_("context")

-- 4. Build
local built = builder_configured:copy(src, ".", "/app")
    :run("go build -o /out/app .", { cwd = "/app" })

-- 5. Prepare runtime
local runtime_configured = runtime:run("apk add --no-cache ca-certificates")

-- 6. Final image
local final = runtime_configured:copy(built, "/out/app", "/app")

-- 7. Export
bk.export(final, { entrypoint = {"/app"} })
```

## Reliability

### 1. Use Specific Exit Codes

Define acceptable exit codes:

```lua
-- Accept 0-3 for certain linters
local result = base:run("make lint", {
    valid_exit_codes = {0, 1, 2, 3}
})

-- Or use range
local result = base:run("make check", {
    valid_exit_codes = "0..5"
})
```

### 2. Handle Missing Files

Use `allow_not_found`:

```lua
local result = base:rm("/opt/optional-file", {
    allow_not_found = true
})
```

### 3. Use Create Dest Path

Avoid errors copying to non-existent paths:

```lua
local result = base:copy(src, "/app/build/server", "/usr/local/bin/server", {
    create_dest_path = true
})
```

### 4. Pin Git Refs

Use specific commits/tags:

```lua
-- Good
local repo = bk.git("https://github.com/user/project.git", {
    ref = "v1.2.3"
})

-- Avoid: Floating ref
local repo = bk.git("https://github.com/user/project.git", {
    ref = "main"
})
```

### 5. Validate HTTP Downloads

Use checksums:

```lua
local file = bk.http("https://example.com/app.tar.gz", {
    checksum = "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
})
```

## Testing

### 1. Validate Scripts

Check syntax before building:

```bash
luakit validate build.lua
```

### 2. Visualize DAG

Inspect build graph:

```bash
luakit dag build.lua | dot -Tsvg > dag.svg
```

### 3. Use Dry Runs

Test without building:

```bash
luakit dag build.lua  # Only checks script validity
```

### 4. Add Health Checks

Configure health checks:

```lua
bk.export(final, {
    entrypoint = {"/app/server"},
    expose = {"8080/tcp"},
    -- Note: healthcheck not yet implemented, coming soon
    -- healthcheck = {
    --     test = {"CMD", "wget", "--spider", "http://localhost:8080/health"},
    --     interval = 30,
    --     timeout = 3,
    --     retries = 3
    -- }
})
```

### 5. Test Multi-Platform

Build for different platforms:

```bash
# Build for ARM64
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/arm64

# Build for multiple platforms
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/amd64,linux/arm64,linux/arm/v7
```

## Common Anti-Patterns

### ❌ Don't: Update `apt-get` and install in separate layers

```lua
-- Bad: Two layers
local layer1 = base:run("apt-get update")
local layer2 = layer1:run("apt-get install -y curl")

-- Good: One layer
local result = base:run("apt-get update && apt-get install -y curl")
```

### ❌ Don't: Copy everything at once

```lua
-- Bad: Invalidates cache on any change
local app = base:copy(bk.local_("context"), ".", "/app")

-- Good: Copy dependency files first
local pkg = base:copy(bk.local_("context", {
    include = { "package*.json" }
}), ".", "/app")
```

### ❌ Don't: Use `latest` tag

```lua
-- Bad: Unpredictable
local base = bk.image("alpine:latest")

-- Good: Predictable
local base = bk.image("alpine:3.19.1")
```

### ❌ Don't: Run as root

```lua
-- Bad: Security risk
bk.export(final)

-- Good: Non-root user
local with_user = final:run("useradd -u 1000 app")
bk.export(with_user, { user = "app" })
```

### ❌ Don't: Hardcode secrets

```lua
-- Bad: Secret in script
local result = base:run("echo 'password123' > /secret")

-- Good: Use secret mount
local result = base:run("cat /run/secrets/password > /secret", {
    mounts = { bk.secret("/run/secrets/password") }
})
```

## Checklist

Before committing a build script, verify:

- [ ] Uses pinned image versions (no `latest`)
- [ ] Runs as non-root user
- [ ] Uses cache mounts for dependencies
- [ ] Copies dependency files before source code
- [ ] Uses minimal base images for runtime
- [ ] No hardcoded secrets or credentials
- [ ] All sensitive files use proper permissions
- [ ] Script is validated with `luakit validate`
- [ ] DAG is reasonable size (visualize with `luakit dag`)
- [ ] Comments explain non-obvious decisions
- [ ] Variables have descriptive names
- [ ] Uses `allow_not_found` for optional files
- [ ] Uses checksums for HTTP downloads
- [ ] Git refs are pinned to tags/commits

## Summary

Follow these best practices to build:

- **Fast**: Optimized caching and parallel execution
- **Secure**: Minimal attack surface and proper isolation
- **Maintainable**: Clear structure and documentation
- **Reliable**: Pinned versions and proper error handling
- **Testable**: Validated scripts and multi-platform support
