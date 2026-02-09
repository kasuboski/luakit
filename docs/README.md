# Luakit Documentation

Welcome to the official documentation for Luakit, a Lua-based frontend for BuildKit.

## What is Luakit?

Luakit allows you to write container image build definitions as Lua scripts. The scripts construct a BuildKit LLB (Low-Level Builder) DAG which is serialized to protobuf and submitted to BuildKit for execution.

## Why Luakit?

- **Lightweight and Fast**: Single binary, ~5ms startup time, no external dependencies
- **Full LLB Coverage**: Exposes all BuildKit features including merge, diff, cache mounts, secrets, and SSH
- **Programmatic**: Use Lua's full power for conditional logic, loops, and composition
- **Simple**: Lua's minimal surface area (~50 keywords) makes build definitions clear
- **Sandboxed**: Scripts cannot execute side effects beyond constructing the DAG

## Table of Contents

- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Your First Build](#your-first-build)
- [User Guide](user-guide.md)
  - [Core Concepts](user-guide.md#core-concepts)
  - [Building Scripts](user-guide.md#building-scripts)
  - [Multi-Stage Builds](user-guide.md#multi-stage-builds)
  - [Caching](user-guide.md#caching)
  - [Networking](user-guide.md#networking)
- [API Reference](api-reference.md)
  - [Source Operations](api-reference.md#source-operations)
  - [Exec Operations](api-reference.md#exec-operations)
  - [File Operations](api-reference.md#file-operations)
  - [Graph Operations](api-reference.md#graph-operations)
  - [Mount Helpers](api-reference.md#mount-helpers)
  - [Export & Metadata](api-reference.md#export--metadata)
  - [Platform](api-reference.md#platform)
- [Tutorials](tutorials.md)
  - [Tutorial 1: Simple Image](tutorials.md#tutorial-1-simple-image)
  - [Tutorial 2: Multi-Stage Build](tutorials.md#tutorial-2-multi-stage-build)
  - [Tutorial 3: Caching Strategy](tutorials.md#tutorial-3-caching-strategy)
  - [Tutorial 4: Advanced Patterns](tutorials.md#tutorial-4-advanced-patterns)
- [Best Practices](best-practices.md)
  - [Performance](best-practices.md#performance)
  - [Security](best-practices.md#security)
  - [Maintainability](best-practices.md#maintainability)
- [CLI Reference](cli-reference.md)
  - [Commands](cli-reference.md#commands)
  - [Flags](cli-reference.md#flags)
  - [Examples](cli-reference.md#examples)
- [Integration](integration.md)
  - [With buildctl](integration.md#with-buildctl)
  - [Gateway Mode](integration.md#gateway-mode)
  - [CI/CD](integration.md#cicd)
- [Common Patterns](patterns.md)
  - [Multi-Stage Builds](patterns.md#multi-stage-builds)
  - [Parallel Builds](patterns.md#parallel-builds)
  - [Layer Optimization](patterns.md#layer-optimization)
  - [Reproducible Builds](patterns.md#reproducible-builds)
- [Troubleshooting](troubleshooting.md)
  - [Common Errors](troubleshooting.md#common-errors)
  - [Debugging](troubleshooting.md#debugging)
  - [Performance Issues](troubleshooting.md#performance-issues)
- [Migration Guide](migration.md)
  - [From Dockerfile](migration.md#from-dockerfile)
  - [Feature Mapping](migration.md#feature-mapping)
  - [Patterns](migration.md#patterns)

## Getting Started

### Installation

#### From Source

```bash
git clone https://github.com/kasuboski/luakit.git
cd luakit
go build -o luakit ./cmd/luakit
```

#### Using Go Install

```bash
go install github.com/kasuboski/luakit/cmd/luakit@latest
```

#### System Requirements

- Go 1.21 or later (for building from source)
- BuildKit daemon (for building images)
- Optional: Docker CLI (for convenience)

### Your First Build

Create a file named `build.lua`:

```lua
local base = bk.image("alpine:3.19")
local result = base:run("echo 'Hello from Luakit!' > /greeting.txt")
bk.export(result)
```

Build the image:

```bash
luakit build build.lua | buildctl build --no-frontend --local context=.
```

Or using Docker:

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  -t hello:latest
```

## Key Concepts

### States

A **State** represents a filesystem state at a point in the build graph. States are immutable - each operation returns a new state.

```lua
local base = bk.image("alpine:3.19")    -- Initial state
local updated = base:run("apk add git")  -- New state, base unchanged
```

### Operations

Operations transform states:

- **Source operations**: Create initial states (`bk.image()`, `bk.local_()`, `bk.git()`, `bk.http()`)
- **Exec operations**: Run commands on states (`state:run()`)
- **File operations**: Manipulate files (`state:copy()`, `state:mkdir()`, etc.)
- **Graph operations**: Combine states (`bk.merge()`, `bk.diff()`)

### Export

`bk.export()` marks the final state and configures the image:

```lua
bk.export(final, {
    entrypoint = {"/app/server"},
    env = {"PORT=8080"},
    user = "app",
    workdir = "/app"
})
```

## Resources

- [Examples](../examples/) - Example build scripts
- [Specification](../SPEC.md) - Detailed technical specification
- [GitHub Issues](https://github.com/kasuboski/luakit/issues) - Bug reports and feature requests
