# Tutorials

Hands-on tutorials to learn Luakit through practical examples.

## Table of Contents

- [Tutorial 1: Simple Image](#tutorial-1-simple-image)
- [Tutorial 2: Multi-Stage Build](#tutorial-2-multi-stage-build)
- [Tutorial 3: Caching Strategy](#tutorial-3-caching-strategy)
- [Tutorial 4: Advanced Patterns](#tutorial-4-advanced-patterns)

## Tutorial 1: Simple Image

**Goal:** Build a simple Alpine image with a greeting file.

### Step 1: Create Build Script

Create `simple.lua`:

```lua
local base = bk.image("alpine:3.19")
local result = base:run("echo 'Hello from Luakit!' > /greeting.txt")
bk.export(result)
```

**Explanation:**

1. `bk.image("alpine:3.19")` - Pull Alpine Linux 3.19
2. `base:run(...)` - Execute command to create greeting file
3. `bk.export(result)` - Mark as final output

### Step 2: Build Image

**Using buildctl:**

```bash
luakit build simple.lua | buildctl build --no-frontend --local context=.
```

**Using Docker:**

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/simple.lua \
  --local context=. \
  -t hello:latest
```

### Step 3: Verify

```bash
docker run --rm hello:latest cat /greeting.txt
# Output: Hello from Luakit!
```

### Exercise

Modify the script to:

1. Add a second file at `/version.txt` containing "1.0.0"
2. Create a directory `/app` with mode 0755

**Solution:**

```lua
local base = bk.image("alpine:3.19")
local with_greeting = base:run("echo 'Hello from Luakit!' > /greeting.txt")
local with_version = with_greeting:run("echo '1.0.0' > /version.txt")
local with_dir = with_version:mkdir("/app", { mode = "0755" })
bk.export(with_dir)
```

---

## Tutorial 2: Multi-Stage Build

**Goal:** Build a Go application with separate build and runtime stages.

### Scenario

Build a simple Go HTTP server:

**main.go:**

```go
package main

import (
    "fmt"
    "net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello from Luakit + Go!")
}

func main() {
    http.HandleFunc("/", handler)
    http.ListenAndServe(":8080", nil)
}
```

### Step 1: Create Build Script

Create `go-app.lua`:

```lua
-- Builder stage
local builder = bk.image("golang:1.21-alpine")

-- Copy go.mod and go.sum first (better caching)
local go_files = bk.local_("context", {
    include_patterns = { "go.mod", "go.sum" }
})
local with_go_files = builder:copy(go_files, ".", "/app")

-- Download dependencies
local deps = with_go_files:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod", { id = "gomod" }) }
})

-- Copy full source
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build application
local built = with_src:run({
    "go", "build",
    "-ldflags=-s -w",
    "-o", "/out/server",
    "."
}, {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})

-- Runtime stage
local runtime = bk.image("alpine:3.19")

-- Copy only the binary
local final = runtime:copy(built, "/out/server", "/app/server")

-- Configure runtime
bk.export(final, {
    entrypoint = {"/app/server"},
    expose = {"8080/tcp"},
    user = "nobody"
})
```

### Step 2: Build Image

```bash
luakit build go-app.lua | buildctl build --no-frontend --local context=. -t go-hello:latest
```

### Step 3: Run and Test

```bash
docker run --rm -p 8080:8080 go-hello:latest

# In another terminal:
curl http://localhost:8080
# Output: Hello from Luakit + Go!
```

### Key Concepts

1. **Separate stages**: Builder uses Go image, runtime uses minimal Alpine
2. **Cache optimization**: Copy `go.mod`/`go.sum` first for better caching
3. **Cache mounts**: Use `bk.cache()` for Go modules and build cache
4. **Minimal runtime**: Only copy the compiled binary, not source

### Exercise

Add a build flag to control debug vs release builds:

**Solution:**

```lua
-- Add at top of script
local release = true

-- Adjust build command
local build_cmd = release
    and {"go", "build", "-ldflags=-s -w", "-o", "/out/server", "."}
    or  {"go", "build", "-o", "/out/server", "."}

local built = with_src:run(build_cmd, {
    cwd = "/app",
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})
```

---

## Tutorial 3: Caching Strategy

**Goal:** Optimize a Node.js application build with effective caching.

### Scenario

Build a Node.js Express application:

**package.json:**

```json
{
  "name": "express-app",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0"
  },
  "scripts": {
    "start": "node index.js",
    "build": "echo 'Building...' > dist/build.txt"
  }
}
```

**index.js:**

```javascript
const express = require('express');
const app = express();

app.get('/', (req, res) => {
  res.send('Hello from Luakit + Node!');
});

app.listen(3000, () => {
  console.log('Server running on port 3000');
});
```

### Step 1: Inefficient Build (Bad Caching)

Create `node-bad.lua`:

```lua
local base = bk.image("node:20-alpine")
local src = bk.local_("context")

-- Copy everything at once
local app = base:copy(src, ".", "/app")

-- Install dependencies
local deps = app:run("npm ci", { cwd = "/app" })

-- Build
local built = deps:run("npm run build", { cwd = "/app" })

bk.export(built, {
    entrypoint = {"node", "index.js"},
    workdir = "/app"
})
```

**Problem:** Any file change invalidates the dependency layer!

### Step 2: Efficient Build (Good Caching)

Create `node-good.lua`:

```lua
local base = bk.image("node:20-alpine")

-- Copy package files first
local pkg_files = bk.local_("context", {
    include_patterns = { "package*.json" }
})
local with_pkg = base:copy(pkg_files, ".", "/app")

-- Install dependencies (cached unless package.json changes)
local deps = with_pkg:run("npm ci", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

-- Copy source code
local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

-- Build application
local built = with_src:run("npm run build", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

bk.export(built, {
    entrypoint = {"node", "index.js"},
    workdir = "/app"
})
```

### Step 3: Compare Performance

**Test bad version:**

```bash
time luakit build node-bad.lua | buildctl build --no-frontend --local context=.
# First build: 30s
# After code change: 30s (reinstalls dependencies!)
```

**Test good version:**

```bash
time luakit build node-good.lua | buildctl build --no-frontend --local context=.
# First build: 30s
# After code change: 5s (uses cached dependencies!)
```

### Step 4: Verify Layers

```bash
buildctl build \
  --local context=. \
  --output type=image,oci-mediatypes=true,name=node:good \
  <(luakit build node-good.lua)

docker history node:good
# See separate dependency and source layers
```

### Key Concepts

1. **Layer ordering**: Copy dependency files before source code
2. **Cache mounts**: Use `bk.cache()` for package managers
3. **Include patterns**: Copy only necessary files early
4. **Cache IDs**: Name caches for better organization

### Exercise

Add a dev dependency installation step that only runs when needed.

**Solution:**

```lua
-- After dependencies, install devDependencies if present
local with_dev = deps:run("npm ci --only=production", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm", { id = "npm" }) }
})

-- Copy source
local src = bk.local_("context")
local with_src = with_dev:copy(src, ".", "/app")

-- Only build if src/index.js exists
local with_src:mkdir("/dist")
local built = with_src:run("npm run build || true", {
    cwd = "/app"
})

bk.export(built, {
    entrypoint = {"node", "index.js"},
    workdir = "/app"
})
```

---

## Tutorial 4: Advanced Patterns

**Goal:** Use advanced Luakit features for complex builds.

### Scenario

Build a Python data science application with:

1. Multi-stage build
2. Parallel operations
3. Secret management
4. Git source
5. Custom platform

### Step 1: Build Script

Create `advanced.lua`:

```lua
-- Import shared library (if you have one)
-- local utils = require("utils")

-- Stage 1: System dependencies (parallel-friendly)
local base = bk.image("python:3.11-slim")

-- Install system packages
local sys_deps = base:run({
    "apt-get", "update"
})
local sys_installed = sys_deps:run({
    "apt-get", "install", "-y",
    "--no-install-recommends",
    "build-essential",
    "git",
    "curl"
})

-- Stage 2: Python packages
local py_files = bk.local_("context", {
    include_patterns = { "requirements*.txt" }
})
local with_py = sys_installed:copy(py_files, ".", "/app")

local py_deps = with_py:run({
    "pip", "install", "--no-cache-dir", "--upgrade", "pip"
}, {
    cwd = "/app"
})

local installed = py_deps:run({
    "pip", "install", "--no-cache-dir", "-r", "requirements.txt"
}, {
    cwd = "/app",
    mounts = {
        bk.cache("/root/.cache/pip", { id = "pip" })
    }
})

-- Stage 3: Source code from git
local repo = bk.git("https://github.com/user/data-science-app.git", {
    ref = "v1.0.0"
})

-- Stage 4: Application
local src = bk.local_("context")
local app = installed:copy(src, ".", "/app")

-- Parallel operations: lint, test, type-check
local linted = app:run({
    "ruff", "check", "src/"
}, {
    cwd = "/app"
})

local tested = app:run({
    "pytest", "tests/"
}, {
    cwd = "/app"
})

local typed = app:run({
    "mypy", "src/"
}, {
    cwd = "/app"
})

-- Merge parallel results
local verified = bk.merge(linted, tested, typed)

-- Stage 5: Final runtime
local runtime = bk.image("python:3.11-slim")

-- Copy application and dependencies
local final = runtime:copy(verified, "/app", "/app")
    :copy(installed, "/usr/local/lib/python3.11/site-packages", "/usr/local/lib/python3.11/site-packages")

-- Add secret for production (only if needed)
local with_config = final:mkfile("/app/config.json", os.getenv("CONFIG") or "{}")

-- Create non-root user
local with_user = with_config:run({
    "sh", "-c",
    "groupadd -r appuser && " ..
    "useradd -r -g appuser -u 1000 appuser && " ..
    "chown -R appuser:appuser /app"
})

-- Export
bk.export(with_user, {
    entrypoint = {"python", "-m", "src.main"},
    workdir = "/app",
    user = "appuser",
    env = {
        PYTHONUNBUFFERED = "1",
        PATH = "/usr/local/bin:/usr/bin:/bin"
    },
    expose = {"8000/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Data Science App",
        ["org.opencontainers.image.version"] = "1.0.0",
        ["org.opencontainers.image.authors"] = "Your Name"
    }
})
```

### Step 2: Build with Platform

```bash
# Build for ARM64
luakit build advanced.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/arm64 \
    -t data-science:arm64
```

### Step 3: Build with Secret

```bash
# Create a secret file
echo '{"api_key":"secret"}' > config.json

# Build with secret
buildctl build \
  --local context=. \
  --secret id=CONFIG,src=config.json
```

### Step 4: Visualize DAG

```bash
luakit dag advanced.lua | dot -Tsvg > dag.svg
```

Open `dag.svg` in a browser to see the build graph.

### Key Concepts

1. **Parallel operations**: Lint, test, and type-check run simultaneously
2. **Merge results**: `bk.merge()` combines independent branches
3. **Git sources**: `bk.git()` pulls from version control
4. **Cross-platform**: Build for different architectures
5. **Secrets**: Pass sensitive data at build time
6. **DAG visualization**: Inspect build structure

### Exercise

Add a production vs debug build mode that:

- Production: Builds optimized version
- Debug: Installs dev dependencies and enables debug logging

**Solution:**

```lua
-- Build mode
local is_production = os.getenv("BUILD_MODE") ~= "debug"

if is_production then
    -- Production build
    local installed = py_deps:run({
        "pip", "install", "--no-cache-dir", "-r", "requirements.txt"
    }, {
        cwd = "/app",
        mounts = { bk.cache("/root/.cache/pip", { id = "pip" }) }
    })

    app = installed:copy(src, ".", "/app")
else
    -- Debug build
    local installed = py_deps:run({
        "pip", "install", "--no-cache-dir",
        "-r", "requirements.txt",
        "-r", "requirements-dev.txt"
    }, {
        cwd = "/app",
        mounts = { bk.cache("/root/.cache/pip", { id = "pip" }) }
    })

    app = installed:copy(src, ".", "/app")
end
```

Build with:

```bash
# Production (default)
luakit build advanced.lua | buildctl build --no-frontend --local context=.

# Debug
BUILD_MODE=debug luakit build advanced.lua | buildctl build --no-frontend --local context=.
```

---

## Summary

You've learned:

1. **Simple builds**: Basic image creation and file manipulation
2. **Multi-stage builds**: Separate build and runtime phases
3. **Caching strategies**: Optimize rebuild times
4. **Advanced patterns**: Parallel operations, secrets, Git sources

## Next Steps

- [API Reference](api-reference.md) - Complete function documentation
- [Best Practices](best-practices.md) - Production-ready patterns
- [Common Patterns](patterns.md) - Reusable build patterns
- [Migration Guide](migration.md) - Converting from Dockerfile
