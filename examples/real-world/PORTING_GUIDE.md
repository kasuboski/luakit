# Real-World Dockerfile Porting to Luakit

This document describes the process and lessons learned from porting real-world Dockerfiles to Luakit scripts.

## Overview

Three production-style Dockerfiles were ported to Luakit to validate practical viability:

1. **Node.js Web Application** - Multi-stage build with production optimizations
2. **Go Microservice** - Optimized binary with minimal runtime
3. **Python Data Science Container** - Complete ML environment

## Porting Process

### 1. Node.js Web Application

**Dockerfile**: `examples/real-world/nodejs/Dockerfile`
**Luakit**: `examples/real-world/nodejs/build.lua`

**Dockerfile Structure:**
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM node:20-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json ./package.json
RUN addgroup && adduser && chown
USER nodejs
EXPOSE 3000
HEALTHCHECK ...
CMD ["node", "dist/index.js"]
```

**Luakit Port:**
```lua
local builder = bk.image("node:20-alpine")

local deps = builder:run({ "npm", "ci", "--only=production" }, {
    cwd = "/app",
    mounts = {
        bk.local_("context", { include = { "package*.json" } }),
    },
})

local built = deps:run({ "npm", "run", "build" }, {
    cwd = "/app",
    mounts = {
        bk.local_("context"),
        bk.cache("/root/.npm", { sharing = "locked" }),
    },
})

local runtime = bk.image("node:20-alpine")

local runtime_deps = runtime:copy(built, "/app/node_modules", "/app/node_modules")
local runtime_dist = runtime_deps:copy(built, "/app/dist", "/app/dist")
local runtime_pkg = runtime_dist:copy(built, "/app/package.json", "/app/package.json")

local with_user = runtime_pkg:run({
    "sh", "-c",
    "addgroup -g 1001 -S nodejs && " ..
    "adduser -S nodejs -u 1001 && " ..
    "chown -R nodejs:nodejs /app"
})

bk.export(with_user, {
    env = { NODE_ENV = "production", PORT = "3000" },
    user = "nodejs",
    workdir = "/app",
    expose = {"3000/tcp"},
    labels = { ... },
})
```

**Key Changes:**
- `WORKDIR` becomes `cwd` parameter in `run()` or `workdir` in `bk.export()`
- Multi-stage builds use separate variables and `copy()` instead of `--from`
- Array form commands: `{"npm", "run", "build"}` instead of shell string
- `ENV` in Dockerfile → `env` table in `bk.export()` or `run()` options
- `USER` → `user` in `bk.export()`
- `EXPOSE` → `expose` in `bk.export()`
- No direct `HEALTHCHECK` equivalent yet (see challenges)

### 2. Go Microservice

**Dockerfile**: `examples/real-world/go/Dockerfile`
**Luakit**: `examples/real-world/go/build.lua`

**Key Porting Decisions:**
- Used cache mounts for Go modules: `bk.cache("/go/pkg/mod", ...)`
- Multi-stage pattern: builder → runtime with `copy()`
- Build flags passed as shell command: `CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' ...`
- `apk add` commands with proper cleanup in one step

**Luakit Improvements:**
- Explicit cache mount with `sharing = "shared"` for faster rebuilds
- Separate cache IDs for Go modules and build cache
- Clean separation of build and runtime phases

### 3. Python Data Science Container

**Dockerfile**: `examples/real-world/python/Dockerfile`
**Luakit**: `examples/real-world/python/build.lua`

**Key Porting Decisions:**
- Long `apt-get` command with cleanup in single `run()` step
- `pip install` with cache mount for faster rebuilds
- Multi-step dependency installation: system → pip → application
- User creation and permission setting in final step

**Luakit Improvements:**
- Cache mount for pip packages: `bk.cache("/root/.cache/pip", ...)`
- Cleaner separation of concerns with named variables
- More explicit about which files come from where

## Challenges Identified

### 1. No Direct HEALTHCHECK Equivalent
**Issue**: Dockerfiles support `HEALTHCHECK` directive for container health monitoring.

**Current Workaround**: Not implemented in Luakit. Would need to be added to `bk.export()` options.

**Proposed Solution**: Add `healthcheck` field to export options:
```lua
bk.export(final, {
    healthcheck = {
        test = {"CMD", "wget", "--spider", "http://localhost:8080/health"},
        interval = 30,
        timeout = 3,
        start_period = 5,
        retries = 3,
    }
})
```

### 2. Limited Pattern Matching in COPY
**Issue**: Dockerfiles support patterns in `COPY package*.json ./` but Luakit's `local_()` mount needs explicit patterns.

**Current Workaround**: Use `include` option with explicit list:
```lua
bk.local_("context", { include = { "package*.json" } })
```

**Limitation**: Can't use complex patterns or exclusions easily.

### 3. Multi-line Shell Scripts
**Issue**: Complex multi-line shell scripts in Dockerfiles need to be converted to Lua string concatenation.

**Example:**
```dockerfile
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001 && \
    chown -R nodejs:nodejs /app
```

**Luakit version:**
```lua
local with_user = runtime:run({
    "sh", "-c",
    "addgroup -g 1001 -S nodejs && " ..
    "adduser -S nodejs -u 1001 && " ..
    "chown -R nodejs:nodejs /app"
})
```

**Workaround**: Use Lua string concatenation or array form for complex commands.

### 4. ARG Directive Missing
**Issue**: Dockerfiles support `ARG` for build-time arguments. Luakit doesn't have a direct equivalent.

**Proposed Solution**: Could be added as:
```lua
local args = bk.get_args({ "VERSION", "DEBUG" })
local image = bk.image("myapp:" .. args.VERSION)
```

### 5. SHELL Directive Missing
**Issue**: Dockerfiles support `SHELL` to change default shell. Luakit always uses `/bin/sh -c` for string commands.

**Workaround**: Use array form to bypass shell:
```lua
local result = base:run({ "/bin/bash", "-c", "my command" })
```

### 6. STOPSIGNAL Not Implemented
**Issue**: Dockerfiles support `STOPSIGNAL` to set signal for stopping container.

**Proposed Solution**: Add to export options:
```lua
bk.export(final, {
    stop_signal = "SIGTERM"
})
```

### 7. VOLUME Declaration Missing
**Issue**: Dockerfiles support `VOLUME` to declare anonymous volumes. No equivalent in Luakit.

**Proposed Solution**: Add to export options:
```lua
bk.export(final, {
    volumes = { "/data", "/logs" }
})
```

## Advantages of Luakit Over Dockerfiles

### 1. Explicit Caching
```lua
bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" })
bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" })
```
Clear, named cache mounts with explicit sharing modes.

### 2. Programmatic Control
```lua
if args.DEBUG then
    result = base:run({ "make", "debug" })
else
    result = base:run({ "make", "release" })
end
```
Conditional logic not possible in Dockerfiles.

### 3. Better Multi-Stage Clarity
```lua
local builder = bk.image("golang:1.21")
local runtime = bk.image("alpine:3.19")

local built = builder:run(...)
local final = runtime:copy(built, "/app/main", "/app/main")
```
Explicit variable names make stage relationships clear.

### 4. Composable Operations
```lua
local merged = bk.merge(state1, state2, state3)
local diffed = bk.diff(base, modified)
```
Merge and diff operations not available in Dockerfiles.

### 5. Testable Scripts
```lua
L := NewVM()
L.DoFile("build.lua")
state := GetExportedState()
```
Can programmatically test and validate build scripts.

## Best Practices Identified

### 1. Use Array Form for Commands
```lua
base:run({"npm", "install"})  -- Good, no shell involved
base:run("npm install")       -- Uses shell, may fail
```

### 2. Name Intermediate States
```lua
local base = bk.image("alpine:3.19")
local with_deps = base:run("apk add git")
local with_app = with_deps:copy(...)
```
Makes debugging and understanding build flow easier.

### 3. Use Cache Mounts for Dependencies
```lua
bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" })
```
Dramatically improves rebuild times.

### 4. Group Related Operations
```lua
local final = bk.image("alpine:3.19")
    :run("apk add ca-certificates")
    :run("adduser app")
    :copy(built, "/app", "/app")
```
Fluent interface chains operations clearly.

### 5. Export with Complete Metadata
```lua
bk.export(final, {
    env = { ... },
    user = "...",
    workdir = "...",
    expose = { ... },
    labels = { ... },
})
```
Don't rely on default values; be explicit.

## Missing Features Summary

| Dockerfile Feature | Luakit Status | Priority |
|-------------------|---------------|----------|
| ARG (build args) | Not implemented | High |
| HEALTHCHECK | Not implemented | High |
| VOLUME | Not implemented | Medium |
| SHELL | Partial (array form workaround) | Low |
| STOPSIGNAL | Not implemented | Low |
| ONBUILD | Not implemented | Low |
| MAINTAINER | Use labels instead | N/A |

## Conclusion

Luakit successfully handles all common Dockerfile patterns for real-world applications:

✅ Multi-stage builds
✅ Environment variables
✅ User management
✅ Working directories
✅ File copying
✅ Cache mounts
✅ Command execution
✅ Image configuration

The main gaps are:
❌ Health checks
❌ Build arguments (ARG)
❌ Volume declarations
❌ Stop signal

These can be added to `bk.export()` options without major architectural changes.

Overall, Luakit provides a powerful, programmatic alternative to Dockerfiles with better caching support, explicit state management, and testable scripts.
