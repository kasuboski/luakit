# Migration Guide

Converting Dockerfiles to Luakit scripts.

## Table of Contents

- [From Dockerfile](#from-dockerfile)
- [Feature Mapping](#feature-mapping)
- [Patterns](#patterns)
- [Examples](#examples)

## From Dockerfile

### Quick Reference

| Dockerfile | Luakit |
|------------|---------|
| `FROM alpine:3.19` | `local base = bk.image("alpine:3.19")` |
| `RUN apt-get update` | `base:run("apt-get update")` |
| `COPY . /app` | `base:copy(bk.local_("context"), ".", "/app")` |
| `ENV KEY=value` | `env = { KEY = "value" }` in export |
| `WORKDIR /app` | `cwd = "/app"` in run options |
| `USER app` | `user = "app"` in export |
| `EXPOSE 8080` | `expose = {"8080/tcp"} in export` |
| `ENTRYPOINT ["/app"]` | `entrypoint = {"/app"} in export` |
| `CMD ["--port", "8080"]` | `cmd = {"--port", "8080"} in export` |

---

## Feature Mapping

### FROM

**Dockerfile:**

```dockerfile
FROM alpine:3.19
FROM golang:1.21 AS builder
```

**Luakit:**

```lua
local base = bk.image("alpine:3.19")
local builder = bk.image("golang:1.21")
```

**Multi-stage Dockerfile:**

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o /out/app .

FROM alpine:3.19
COPY --from=builder /out/app /app
```

**Luakit:**

```lua
-- Builder stage
local builder = bk.image("golang:1.21")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/app .", { cwd = "/app" })

-- Runtime stage
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/app")
```

---

### RUN

**Dockerfile:**

```dockerfile
RUN apt-get update && apt-get install -y curl
RUN echo "hello" > /greeting.txt
```

**Luakit:**

```lua
-- String form (via shell)
local result1 = base:run("apt-get update && apt-get install -y curl")

-- Array form (direct exec)
local result2 = base:run({"sh", "-c", "echo hello > /greeting.txt"})
```

**With options:**

**Dockerfile:**

```dockerfile
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install -r requirements.txt
RUN --network=none \
    mise run test
```

**Luakit:**

```lua
local result1 = base:run("pip install -r requirements.txt", {
    mounts = { bk.cache("/root/.cache/pip") }
})

local result2 = base:run("mise run test", {
    network = "none"
})
```

---

### COPY

**Dockerfile:**

```dockerfile
COPY . /app
COPY --from=builder /out/app /app
COPY --chown=app:app src/ /app
```

**Luakit:**

```lua
-- From context
local result1 = base:copy(bk.local_("context"), ".", "/app")

-- From another state
local result2 = base:copy(builder, "/out/app", "/app")

-- With ownership
local result3 = base:copy(src, "src/", "/app/", {
    owner = { user = "app", group = "app" }
})
```

**With patterns:**

**Dockerfile:**

```dockerfile
COPY package*.json ./
```

**Luakit:**

```lua
local pkg = bk.local_("context", {
    include = { "package*.json" }
})
local result = base:copy(pkg, ".", "/app")
```

---

### ENV

**Dockerfile:**

```dockerfile
ENV NODE_ENV=production
ENV PORT=8080
ENV PATH=/usr/local/bin:$PATH
```

**Luakit:**

In `bk.export()`:

```lua
bk.export(final, {
    env = {
        NODE_ENV = "production",
        PORT = "8080",
        PATH = "/usr/local/bin:$PATH"
    }
})
```

In `run()` options:

```lua
local result = base:run("go build", {
    env = {
        CGO_ENABLED = "0",
        GOOS = "linux"
    }
})
```

---

### WORKDIR

**Dockerfile:**

```dockerfile
WORKDIR /app
```

**Luakit:**

Per command:

```lua
local result = base:run("make", { cwd = "/app" })
```

Global in export:

```lua
bk.export(final, {
    workdir = "/app"
})
```

---

### USER

**Dockerfile:**

```dockerfile
USER app
USER 1000:1000
```

**Luakit:**

Create user and set default:

```lua
local with_user = base:run({
    "sh", "-c",
    "groupadd -r app && " ..
    "useradd -r -g app -u 1000 app"
})

bk.export(with_user, {
    user = "app"
})
```

Or per command:

```lua
local result = base:run("npm install", { user = "app" })
```

---

### EXPOSE

**Dockerfile:**

```dockerfile
EXPOSE 8080
EXPOSE 443/tcp
```

**Luakit:**

```lua
bk.export(final, {
    expose = {"8080/tcp", "443/tcp"}
})
```

---

### ENTRYPOINT

**Dockerfile:**

```dockerfile
ENTRYPOINT ["/app/server"]
ENTRYPOINT ["/bin/sh", "-c"]
```

**Luakit:**

```lua
bk.export(final, {
    entrypoint = {"/app/server"}
})
```

---

### CMD

**Dockerfile:**

```dockerfile
CMD ["--port", "8080"]
CMD ["/bin/sh"]
```

**Luakit:**

```lua
bk.export(final, {
    cmd = {"--port", "8080"}
})
```

---

### ARG

**Dockerfile:**

```dockerfile
ARG VERSION=1.0.0
ARG BUILD_ENV=development

FROM alpine:${VERSION}
```

**Luakit:**

Use environment variables:

```lua
local version = os.getenv("VERSION") or "1.0.0"
local build_env = os.getenv("BUILD_ENV") or "development"

local base = bk.image("alpine:" .. version)
```

Pass via CLI:

```bash
VERSION=3.19 BUILD_ENV=production luakit build build.lua
```

---

## Patterns

### Pattern 1: Basic Alpine Image

**Dockerfile:**

```dockerfile
FROM alpine:3.19
RUN apk add --no-cache curl git
WORKDIR /app
COPY . .
RUN echo "built" > /timestamp
```

**Luakit:**

```lua
local base = bk.image("alpine:3.19")
local with_tools = base:run("apk add --no-cache curl git")
local with_dir = with_tools:mkdir("/app")
local with_src = with_dir:copy(bk.local_("context"), ".", "/app")
local final = with_src:run("echo 'built' > /timestamp", { cwd = "/app" })
bk.export(final)
```

---

### Pattern 2: Go Multi-Stage Build

**Dockerfile:**

```dockerfile
# Builder
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# Runtime
FROM alpine:3.19
COPY --from=builder /out/server /server
ENTRYPOINT ["/server"]
```

**Luakit:**

```lua
-- Builder
local builder = bk.image("golang:1.21")
local go_files = bk.local_("context", {
    include = { "go.mod", "go.sum" }
})
local with_go = builder:copy(go_files, ".", "/app")
local deps = with_go:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") }
})
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")
local built = with_src:run({
    "go", "build",
    "-o", "/out/server",
    "./cmd/server"
}, {
    cwd = "/app",
    env = { CGO_ENABLED = "0" },
    mounts = {
        bk.cache("/go/pkg/mod"),
        bk.cache("/root/.cache/go-build")
    }
})

-- Runtime
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, {
    entrypoint = {"/server"}
})
```

---

### Pattern 3: Node.js Multi-Stage Build

**Dockerfile:**

```dockerfile
# Builder
FROM node:20 AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

# Runtime
FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json ./package.json
ENV NODE_ENV=production
ENTRYPOINT ["node", "dist/index.js"]
```

**Luakit:**

```lua
-- Builder
local builder = bk.image("node:20")
local pkg_files = bk.local_("context", {
    include = { "package*.json" }
})
local with_pkg = builder:copy(pkg_files, ".", "/app")
local deps = with_pkg:run("npm ci --only=production", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")
local built = with_src:run("npm run build", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

-- Runtime
local runtime = bk.image("node:20-alpine")
local final = runtime:copy(built, "/app/node_modules", "/app/node_modules")
    :copy(built, "/app/dist", "/app/dist")
    :copy(built, "/app/package.json", "/app/package.json")

bk.export(final, {
    workdir = "/app",
    env = { NODE_ENV = "production" },
    entrypoint = {"node", "dist/index.js"}
})
```

---

### Pattern 4: Python Multi-Stage Build

**Dockerfile:**

```dockerfile
# Builder
FROM python:3.11-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN python -m compileall -q src/

# Runtime
FROM python:3.11-slim
WORKDIR /app
COPY --from=builder /app /app
ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "-m", "src.main"]
```

**Luakit:**

```lua
-- Builder
local builder = bk.image("python:3.11-slim")
local req = builder:copy(bk.local_("context", {
    include = { "requirements.txt" }
}), ".", "/app")
local deps = req:run("pip install --no-cache-dir -r requirements.txt", {
    cwd = "/app",
    mounts = { bk.cache("/root/.cache/pip") }
})
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")
local built = with_src:run("python -m compileall -q src/", { cwd = "/app" })

-- Runtime
local runtime = bk.image("python:3.11-slim")
local final = runtime:copy(built, "/app", "/app")

bk.export(final, {
    workdir = "/app",
    env = { PYTHONUNBUFFERED = "1" },
    entrypoint = {"python", "-m", "src.main"}
})
```

---

## Examples

### Example 1: Simple Web Server

**Dockerfile:**

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build
EXPOSE 3000
CMD ["node", "dist/index.js"]
```

**Luakit:**

```lua
local base = bk.image("node:20-alpine")
local pkg = base:copy(bk.local_("context", {
    include = { "package*.json" }
}), ".", "/app")
local deps = pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")
local built = with_src:run("npm run build", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") }
})

bk.export(built, {
    workdir = "/app",
    expose = {"3000/tcp"},
    cmd = {"node", "dist/index.js"}
})
```

---

### Example 2: Go Microservice

**Dockerfile:**

```dockerfile
# Build
FROM golang:1.21 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main .

# Run
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=build /src/main .
CMD ["./main"]
```

**Luakit:**

```lua
local builder = bk.image("golang:1.21")
local go_files = bk.local_("context", {
    include = { "go.mod", "go.sum" }
})
local with_go = builder:copy(go_files, ".", "/src")
local deps = with_go:run("go mod download", {
    cwd = "/src",
    mounts = { bk.cache("/go/pkg/mod") }
})
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/src")
local built = with_src:run({
    "go", "build",
    "-a", "-installsuffix", "cgo",
    "-ldflags=-w -s",
    "-o", "main", "."
}, {
    cwd = "/src",
    env = { CGO_ENABLED = "0", GOOS = "linux" },
    mounts = {
        bk.cache("/go/pkg/mod"),
        bk.cache("/root/.cache/go-build")
    }
})

local runtime = bk.image("alpine:3.19")
local with_certs = runtime:run("apk --no-cache add ca-certificates")
local final = with_certs:copy(built, "/src/main", "/root/main")

bk.export(final, {
    workdir = "/root/",
    cmd = {"./main"}
})
```

---

### Example 3: Django Application

**Dockerfile:**

```dockerfile
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY . .

# Collect static files
RUN python manage.py collectstatic --noinput

# Create non-root user
RUN groupadd -r app && useradd -r -g app app
RUN chown -R app:app /app

USER app

EXPOSE 8000
CMD ["gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:8000"]
```

**Luakit:**

```lua
local base = bk.image("python:3.11-slim")

-- Install system dependencies
local with_sys = base:run({
    "sh", "-c",
    "apt-get update && " ..
    "apt-get install -y --no-install-recommends postgresql-client && " ..
    "rm -rf /var/lib/apt/lists/*"
})

-- Install Python dependencies
local req = bk.local_("context", {
    include = { "requirements.txt" }
})
local with_req = with_sys:copy(req, ".", "/app")
local with_deps = with_req:run("pip install --no-cache-dir -r requirements.txt", {
    cwd = "/app",
    mounts = { bk.cache("/root/.cache/pip") }
})

-- Copy application
local src = bk.local_("context")
local with_app = with_deps:copy(src, ".", "/app")

-- Collect static files
local with_static = with_app:run("python manage.py collectstatic --noinput", {
    cwd = "/app"
})

-- Create non-root user
local with_user = with_static:run({
    "sh", "-c",
    "groupadd -r app && " ..
    "useradd -r -g app app && " ..
    "chown -R app:app /app"
})

bk.export(with_user, {
    workdir = "/app",
    user = "app",
    expose = {"8000/tcp"},
    cmd = {"gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:8000"}
})
```

---

### Example 4: Multi-Platform Dockerfile

**Dockerfile:**

```dockerfile
# This would require multiple Dockerfiles or buildx
FROM alpine:3.19
RUN echo "multi-platform not directly supported"
```

**Luakit:**

```lua
-- Build for multiple platforms
local platforms = {
    bk.platform("linux", "amd64"),
    bk.platform("linux", "arm64"),
    bk.platform("linux", "arm", "v7")
}

for _, p in ipairs(platforms) do
    local base = bk.image("alpine:3.19", { platform = p })
    local result = base:run("echo 'Hello from ' .. tostring(p) .. '!'")
    bk.export(result)
end
```

---

## Missing Features

| Dockerfile | Luakit Status | Alternative |
|------------|---------------|-------------|
| ARG | Not implemented | Use `os.getenv()` |
| HEALTHCHECK | Not implemented | Add external health checks |
| VOLUME | Not implemented | Add to export options (coming soon) |
| SHELL | Partial | Use array form |
| STOPSIGNAL | Not implemented | Add to export options (coming soon) |
| ONBUILD | Not implemented | Use helper functions |
| MAINTAINER | Deprecated | Use labels |

---

## Migration Checklist

When migrating from Dockerfile to Luakit:

- [ ] Replace `FROM` with `bk.image()`
- [ ] Convert `RUN` to `state:run()`
- [ ] Convert `COPY` to `state:copy()`
- [ ] Move `ENV` to `bk.export()` or `run()` options
- [ ] Move `WORKDIR` to `cwd` parameter or `bk.export()`
- [ ] Move `USER` to `bk.export()` or `run()` options
- [ ] Move `EXPOSE` to `bk.export()`
- [ ] Move `ENTRYPOINT` to `bk.export()`
- [ ] Move `CMD` to `bk.export()`
- [ ] Convert cache mounts to `bk.cache()`
- [ ] Convert multi-stage with separate variables
- [ ] Add cache mounts for dependencies
- [ ] Use array form for commands
- [ ] Call `bk.export()` exactly once
- [ ] Validate with `luakit validate`
- [ ] Test with `buildctl`

---

## Summary

Key migration points:

1. **Multi-stage**: Use separate variables for each stage
2. **Copy**: Use `state:copy()` with `bk.local_("context")`
3. **Environment**: Move to `bk.export()` or `run()` options
4. **Cache mounts**: Use `bk.cache()` for better performance
5. **Export**: Call `bk.export()` exactly once with all config

Luakit advantages over Dockerfile:

- **Better caching**: Explicit cache mounts
- **Programmatic**: Use Lua's full power
- **Advanced ops**: `bk.merge()`, `bk.diff()` not in Dockerfile
- **Testable**: Scripts can be validated and tested

Most Dockerfile patterns have direct Luakit equivalents. Use the migration examples as templates for your own Dockerfiles.
