# Prelude Library

The `prelude` library provides common build patterns and convenience helpers for luakit. It can be imported with `require("prelude")` in your build scripts.

## Base Images

Create base images from common distributions:

```lua
local prelude = require("prelude")

-- Alpine (default: 3.19)
local alpine = prelude.from_alpine()
local alpine_318 = prelude.from_alpine("3.18")

-- Ubuntu (default: 24.04)
local ubuntu = prelude.from_ubuntu()
local ubuntu_2204 = prelude.from_ubuntu("22.04")

-- Debian (default: bookworm-slim)
local debian = prelude.from_debian()
local debian_bullseye = prelude.from_debian("bullseye-slim")

-- Fedora (default: 39)
local fedora = prelude.from_fedora()

-- Generic base image
local base = prelude.standard_base("alpine")
```

## Go Language Support

Build Go applications with pre-configured environments:

```lua
local prelude = require("prelude")
local src = bk.local_("context")

-- Go base with build tools
local go_base = prelude.go_base("1.22-alpine")

-- Build a Go application
local built = prelude.go_build(go_base, src, {
	cwd = "/app",
	main = "./cmd/server",
	output = "/out/server",
	ldflags = "-s -w",
})

-- Go runtime with certificates
local go_runtime = prelude.go_runtime("3.19")

-- Complete Go binary application (multi-stage)
local final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = ".",
	user = "app",
	uid = 1000,
	gid = 1000,
})

bk.export(final, {
	entrypoint = {"/app/server"},
})
```

## Node.js Language Support

Build Node.js applications:

```lua
local prelude = require("prelude")
local src = bk.local_("context")

-- Node base
local node_base = prelude.node_base("20-alpine")

-- Build Node app
local built = prelude.node_build(node_base, src, {
	cwd = "/app",
	deps_only = false,
	install_cmd = "npm run build",
})

-- Node runtime
local node_runtime = prelude.node_runtime("20-alpine")

-- Complete Node application
local final = prelude.node_app("20-alpine", src, {
	cwd = "/app",
	user = "nodejs",
})

bk.export(final, {
	user = "nodejs",
	workdir = "/app",
})
```

## Python Language Support

Build Python applications:

```lua
local prelude = require("prelude")
local src = bk.local_("context")

-- Python base with build tools
local py_base = prelude.python_base("3.11", "slim")

-- Build Python app
local built = prelude.python_build(py_base, src, {
	cwd = "/workspace",
	requirements = "requirements.txt",
	install_cmd = "pip install -e .",
})

-- Python runtime
local py_runtime = prelude.python_runtime("3.11-slim")

-- Complete Python application
local final = prelude.python_app("3.11", src, {
	cwd = "/workspace",
	user = "appuser",
})

bk.export(final, {
	user = "appuser",
	workdir = "/app",
})
```

## Package Management

Install packages easily:

```lua
local prelude = require("prelude")

-- Alpine packages
local alpine = prelude.from_alpine()
local with_git = prelude.apk_package(alpine, { "git", "curl" })
local with_git_str = prelude.apk_package(alpine, "git curl")

-- Debian/Ubuntu packages
local debian = prelude.from_debian()
local with_tools = prelude.deb_package(debian, { "git", "curl", "vim" })
local with_tools_str = prelude.deb_package(debian, "git curl vim")

-- Convenience functions
local with_git = prelude.install_git(alpine)
local with_curl = prelude.install_curl(alpine)
local with_ca_certs = prelude.install_ca_certs(alpine)

-- System dependencies (auto-detects distro)
local with_deps = prelude.install_system_deps(base, { "git", "curl" }, "alpine")
```

## Directory and User Helpers

Manage directories and users:

```lua
local prelude = require("prelude")
local base = bk.image("alpine:3.19")

-- Create working directory
local with_workdir = prelude.with_workdir(base, "/app")

-- Add user (Alpine)
local with_user = prelude.with_alpine_user(base, "appuser", 1000, 1000)

-- Add user (Debian/Ubuntu)
local deb = prelude.from_debian()
local with_user = prelude.with_user(deb, "appuser", 1000, 1000)

-- Change ownership
local app = base:run("mkdir -p /app")
local owned = prelude.chown_path(app, "/app", "appuser", "appuser")

-- Create non-root user
local non_root = prelude.as_non_root(app, "myapp", 2000)
```

## Multi-Stage Builds

Use common multi-stage patterns:

```lua
local prelude = require("prelude")

-- Container helper
local result = prelude.container(base, function(s)
	return s:run("echo building")
end)

-- Multi-stage helper
local runtime, built = prelude.multi_stage("golang:1.22", "alpine:3.19", function(builder)
	return builder:run("go build -o /out/app .")
end)

-- Copy all from source
local final = prelude.copy_all(built, runtime, "/out/app", "/app/app")
```

## Parallel and Merge Operations

Build in parallel and merge results:

```lua
local prelude = require("prelude")
local base = bk.image("alpine:3.19")

-- Build multiple stages in parallel
local state1 = base:run("echo 'task1' > /task1.txt")
local state2 = base:run("echo 'task2' > /task2.txt")
local state3 = base:run("echo 'task3' > /task3.txt")

-- Merge parallel builds
local merged = prelude.parallel_build(state1, state2, state3)

-- Merge from table
local states = { state1, state2, state3 }
local merged = prelude.merge_multiple(states)

-- Layered copy
local target = bk.scratch()
local layered = prelude.layered_copy(target, {}, {
	{ from = state1, from_path = "/task1.txt", to_path = "/task1.txt" },
	{ from = state2, from_path = "/task2.txt", to_path = "/task2.txt" },
})
```

## Complete Examples

### Go Microservice

```lua
local prelude = require("prelude")
local src = bk.local_("context")

local final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = "./cmd/server",
	user = "app",
})

bk.export(final, {
	entrypoint = {"/app/server"},
	user = "app",
	workdir = "/app",
	expose = {"8080/tcp"},
})
```

### Node.js Web App

```lua
local prelude = require("prelude")
local src = bk.local_("context")

local final = prelude.node_app("20-alpine", src, {
	cwd = "/app",
	user = "nodejs",
})

bk.export(final, {
	user = "nodejs",
	workdir = "/app",
	expose = {"3000/tcp"},
	env = {
		NODE_ENV = "production",
	},
})
```

### Python Application

```lua
local prelude = require("prelude")
local src = bk.local_("context")

local final = prelude.python_app("3.11", src, {
	cwd = "/workspace",
	requirements = "requirements.txt",
	user = "appuser",
})

bk.export(final, {
	user = "appuser",
	workdir = "/app",
})
```

## API Reference

### Base Images

- `from_alpine(version)` → State
- `from_ubuntu(version)` → State  
- `from_debian(version)` → State
- `from_fedora(version)` → State
- `standard_base(distro, version)` → State

### Go

- `go_base(version)` → State
- `go_runtime(version)` → State
- `go_build(builder, src, opts)` → State
- `go_binary_app(version, src, opts)` → State

### Node.js

- `node_base(version)` → State
- `node_runtime(version)` → State
- `node_build(builder, src, opts)` → State
- `node_app(version, src, opts)` → State

### Python

- `python_base(version, variant)` → State
- `python_runtime(version)` → State
- `python_build(builder, src, opts)` → State
- `python_app(version, src, opts)` → State

### Packages

- `apk_package(base, packages)` → State
- `deb_package(base, packages)` → State
- `install_git(base)` → State
- `install_curl(base)` → State
- `install_ca_certs(base)` → State
- `install_system_deps(base, packages, distro)` → State

### Directory & User

- `with_workdir(state, path)` → State
- `with_alpine_user(state, username, uid, gid)` → State
- `with_user(state, username, uid, gid)` → State
- `chown_path(state, path, user, group)` → State
- `as_non_root(state, username, uid)` → State

### Multi-Stage & Helpers

- `container(base, build_fn)` → State
- `multi_stage(builder_image, runtime_image, build_fn)` → State, State
- `copy_all(from_state, to_state, from_path, to_path)` → State
- `parallel_build(...)` → State
- `merge_multiple(states)` → State
- `layered_copy(target, sources, mappings)` → State
