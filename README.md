# Luakit - BuildKit Frontend Driven by Lua

[![Go Report Card](https://goreportcard.com/badge/github.com/kasuboski/luakit)](https://goreportcard.com/report/github.com/kasuboski/luakit)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Luakit is a Lua-based frontend for BuildKit that allows you to write container image build definitions as Lua scripts. The scripts construct a BuildKit LLB (Low-Level Builder) DAG which is serialized to protobuf and submitted to BuildKit for execution.

## Features

- **Lightweight**: Single binary, ~5ms startup time
- **Full LLB Coverage**: Exposes all BuildKit features including merge, diff, cache mounts, secrets, and SSH
- **Programmatic**: Use Lua's full power for conditional logic, loops, and composition
- **Simple**: Lua's minimal surface area makes build definitions clear
- **Sandboxed**: Scripts cannot execute side effects beyond constructing the DAG
- **Fast**: Optimized caching and parallel execution

## Quick Start

### Your First Build

Create `build.lua`:

```lua
local base = bk.image("alpine:3.19")
local result = base:run("echo 'Hello from Luakit!' > /greeting.txt")
bk.export(result)
```

Build the image:

```bash
luakit build build.lua | buildctl build --no-frontend --local context=.
```

### Installation

**From pre-built binaries (Recommended):**

Download from the [Releases](https://github.com/kasuboski/luakit/releases) page for your platform:

```bash
# Linux AMD64
wget https://github.com/kasuboski/luakit/releases/download/v0.1.0/luakit-0.1.0-linux-amd64
chmod +x luakit-0.1.0-linux-amd64
sudo mv luakit-0.1.0-linux-amd64 /usr/local/bin/luakit

# macOS (Intel or Apple Silicon)
curl -L -O https://github.com/kasuboski/luakit/releases/download/v0.1.0/luakit-0.1.0-darwin-$(uname -m)
chmod +x luakit-0.1.0-darwin-$(uname -m)
sudo mv luakit-0.1.0-darwin-$(uname -m) /usr/local/bin/luakit
```

**With Homebrew:**

```bash
brew install https://github.com/kasuboski/luakit/releases/download/v0.1.0/luakit.rb
```

**With Go:**

```bash
# Install CLI binary
go install github.com/kasuboski/luakit/cmd/luakit@latest

# Install gateway binary (for BuildKit frontend mode)
go install github.com/kasuboski/luakit/cmd/gateway@latest
```

### CLI Usage

```bash
# Validate a script
luakit validate build.lua

# Visualize the build graph
luakit dag build.lua | dot -Tsvg > dag.svg

# Build an image (pipes to buildctl)
luakit build build.lua | buildctl build --no-frontend --local context=.

# Build with custom output file
luakit build -o output.pb build.lua
```

## BuildKit Daemon Setup

Luakit requires BuildKit for executing builds. The BuildKit worker configuration is important for compatibility, especially when running in virtualized environments like Lima.

### Lima VM (macOS with Docker Desktop)

If using Lima with Docker Desktop, BuildKit's default `native` snapshotter may fail with permission errors due to Lima's FUSE/SSHFS filesystem sharing. Use the `overlayfs` snapshotter instead:

```bash
# Stop existing buildkitd container
limactl shell docker
docker stop buildkitd && docker rm buildkitd

# Start buildkitd with overlayfs snapshotter
docker run -d --name buildkitd \
  -v /tmp/buildkit:/run/buildkit \
  -v 69a02fc920b683a9a4ae556543230abb2d2dc387288d8dca3dfadf8b316fcc5a:/var/lib/buildkit \
  moby/buildkit:buildx-stable-1 \
  --addr unix:///run/buildkit/buildkitd.sock \
  --oci-worker-snapshotter=overlayfs

# Exit Lima shell
exit
```

Set the BuildKit socket environment variable:

```bash
export BUILDKIT_HOST=unix://$HOME/.lima/docker/sock/buildkitd.sock
```

### Native Linux / Docker Desktop (Linux)

Docker Desktop on Linux typically doesn't have the same filesystem restrictions, so BuildKit's default configuration should work:

```bash
# Verify BuildKit is running
docker ps | grep buildkitd

# If not running, start with defaults
docker run -d --name buildkitd \
  -v /tmp/buildkit:/run/buildkit \
  moby/buildkit:buildx-stable-1
```

### Standalone BuildKit

For standalone BuildKit installations:

```bash
# Using buildkitd from package manager
buildkitd --oci-worker-snapshotter=overlayfs

# Or via container
docker run -d --name buildkitd \
  -v /tmp/buildkit:/run/buildkit \
  -v buildkit-data:/var/lib/buildkit \
  moby/buildkit:buildx-stable-1 \
  --addr unix:///run/buildkit/buildkitd.sock \
  --oci-worker-snapshotter=overlayfs
```

### Why overlayfs?

The `overlayfs` snapshotter:
- Works reliably across different filesystem types
- Supports recursive bind mounts used by BuildKit
- Compatible with FUSE/SSHFS filesystems (Lima, Colima)
- Provides good performance for container builds

The `native` snapshotter can cause issues like:
- `operation not permitted` errors during mount operations
- `failed to mount /tmp/containerd-mount*` errors
- Incompatibility with FUSE-based filesystems

## Two Ways to Use Luakit

Luakit works in two modes:

### CLI Mode (Local Development)
Use the `luakit` binary directly for local builds and iteration:
```bash
luakit build build.lua | buildctl build --no-frontend --local context=.
```

The CLI reads `build.lua` from disk and outputs the LLB Definition protobuf to stdout, which you can pipe to `buildctl`.

### Gateway Frontend (CI/Production)
Package luakit as a BuildKit frontend image for container-based builds. The `gateway` binary acts as a BuildKit frontend via the gRPC protocol, receiving the build context from BuildKit and returning the LLB Definition.

```bash
# Using docker buildx
docker buildx build \
  --frontend gateway.v0 \
  --opt source=docker.io/kasuboski/luakit:latest \
  --local context=. \
  -t myapp:latest
```

Or with buildctl directly:
```bash
buildctl build \
  --frontend=gateway.v0 \
  --opt source=docker.io/kasuboski/luakit:latest \
  --local context=.
```

The gateway mode reads `build.lua` from the build context and returns the LLB Definition to BuildKit via gRPC, enabling seamless integration with Docker BuildKit and CI/CD pipelines.

**When to use which?**
- **CLI mode**: Local development, debugging, one-off builds
- **Gateway mode**: CI/CD pipelines, production builds, when you need a container image reference

## Documentation

- [Getting Started](docs/README.md) - Introduction and installation
- [User Guide](docs/user-guide.md) - Core concepts and practical usage
- [API Reference](docs/api-reference.md) - Complete function documentation
- [Tutorials](docs/tutorials.md) - Hands-on learning examples
- [Best Practices](docs/best-practices.md) - Production-ready patterns
- [CLI Reference](docs/cli-reference.md) - Command-line interface
- [Integration](docs/integration.md) - BuildKit, Docker, and CI/CD integration
- [Common Patterns](docs/patterns.md) - Reusable build patterns
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
- [Migration Guide](docs/migration.md) - Converting from Dockerfile
- [Release Process](docs/RELEASE.md) - Creating and publishing releases

## Editor Support

Luakit provides type definitions for Lua editors using LuaLS (Lua Language Server). This enables autocomplete, type checking, hover documentation, and go-to-definition.

### VSCode

Install the [Lua Language Server](https://marketplace.visualstudio.com/items?itemName=sumneko.lua) extension. The extension will automatically discover the type definitions in the `types/` directory.

### Neovim

Ensure `lua-language-server` is installed and configured with nvim-lspconfig. The type definitions will be automatically detected.

### Other Editors

Any editor supporting LuaLS will automatically detect the type definitions in the `types/` directory. The `.luarc.json` file at the project root configures the language server to use these definitions.

## Example: Go Multi-Stage Build

```lua
-- Builder stage
local builder = bk.image("golang:1.21-alpine")

local go_files = bk.local_("context", {
    include = { "go.mod", "go.sum" }
})
local with_go = builder:copy(go_files, ".", "/app")

local deps = with_go:run("go mod download", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod", { id = "gomod" }) }
})

local src = bk.local_("context")
local with_src = deps:copy(src, ".", "/app")

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
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, {
    entrypoint = {"/server"},
    user = "nobody",
    expose = {"8080/tcp"}
})
```

### Gateway Mode

```bash
# Build with Docker using the gateway frontend
docker buildx build \
  --frontend gateway.v0 \
  --opt source=docker.io/kasuboski/luakit:latest \
  --local context=. \
  -t myapp:latest

# Build with buildctl using the gateway frontend
buildctl build \
  --frontend=gateway.v0 \
  --opt source=docker.io/kasuboski/luakit:latest \
  --local context=.

# Using a specific build.lua path in context
docker buildx build \
  --frontend gateway.v0 \
  --opt source=docker.io/kasuboski/luakit:latest \
  --opt filename=builds/production.lua \
  --local context=. \
  -t myapp:latest
```

## Comparison to Alternatives

| Frontend | Language | Startup | LLB Coverage | Learning Curve |
|----------|----------|---------|--------------|----------------|
| Dockerfile | Custom DSL | N/A (native) | Partial | Low |
| Dagger | Go/Python/TS | ~2-5s (container) | Full | Medium-High |
| Earthly | Earthfile DSL | ~1s | Partial | Low-Medium |
| HLB | Custom DSL | ~100ms | Full | Medium |
| **Luakit** | **Lua** | **~5ms** | **Full** | **Low** |

## Why Lua?

Lua is ideal for build definitions:

- **Tiny and embeddable**: The reference Lua VM is ~200KB
- **Minimal language surface**: ~50 keywords, learn in an hour
- **Proven as a config language**: Used by Nginx, Redis, Neovim, and game engines
- **Sandboxable by default**: Can disable `os.execute`, `io.open`, etc.
- **No dependency chain**: A `.lua` file has zero external dependencies

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                           CLI Mode                                 │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  build.lua  ──▶  luakit CLI  ──▶  pb.Definition  ──▶  buildctl    │
│                   (cmd/luakit)              (stdout)              │
│                     ┌─────────────┐                                │
│                     │  Go binary   │                               │
│                     │  ┌─────────┐ │                                │
│                     │  │ Lua VM  │ │    Evaluates build.lua        │
│                     │  │(gopher- │ │    Lua calls Go-registered     │
│                     │  │  lua)   │ │    functions                  │
│                     │  └────┬────┘ │                                │
│                     │       │      │                                │
│                     │  ┌────▼────┐ │                                │
│                     │  │  DAG    │ │    In-memory graph of Op nodes │
│                     │  │ Builder │ │    with edges (inputs)        │
│                     │  └────┬────┘ │                                │
│                     │       │      │                                │
│                     │  ┌────▼────┐ │                                │
│                     │  │   LLB   │ │    Marshal DAG to pb.Definition│
│                     │  │Serialize│ │    using BuildKit's Go types   │
│                     │  └────┬────┘ │                                │
│                     └───────│──────┘                                │
│                             │                                       │
│                             ▼                                       │
│                         stdout                                     │
│                             │                                       │
└─────────────────────────────┼───────────────────────────────────────┘
                              │
                              ▼
                      BuildKit Daemon
                              │
┌─────────────────────────────┼───────────────────────────────────────┐
│                      Gateway Mode                                    │
├────────────────────────────────────────────────────────────────────┤
│                              │                                       │
│            BuildKit gRPC ────▼──────▶  luakit gateway               │
│         (build context + build.lua)      (cmd/gateway)               │
│                        ┌─────────────┐                             │
│                        │  Go binary   │                             │
│                        │  ┌─────────┐ │                             │
│                        │  │ Lua VM  │ │    Evaluates build.lua     │
│                        │  │(gopher- │ │    from build context      │
│                        │  │  lua)   │ │                             │
│                        │  └────┬────┘ │                             │
│                        │       │      │                             │
│                        │  ┌────▼────┐ │                             │
│                        │  │  DAG    │ │    In-memory graph         │
│                        │  │ Builder │ │                             │
│                        │  └────┬────┘ │                             │
│                        │       │      │                             │
│                        │  ┌────▼────┐ │                             │
│                        │  │   LLB   │ │    Marshal to pb.Definition│
│                        │  │Serialize│ │                             │
│                        │  └────┬────┘ │                             │
│                        └───────│──────┘                             │
│                                │                                     │
│                                ▼                                     │
│                      gRPC response (pb.Definition)                 │
│                                │                                     │
│                                ▼                                     │
│                      BuildKit Daemon                               │
└────────────────────────────────┴─────────────────────────────────────┘
```

## Development

### Components

Luakit consists of two main binaries:

- **`cmd/luakit`** - CLI tool for local development. Reads `build.lua` from disk and outputs LLB Definition to stdout.
- **`cmd/gateway`** - BuildKit frontend for CI/production. Receives build context via gRPC from BuildKit and returns LLB Definition.

### Setup

```bash
git clone https://github.com/kasuboski/luakit.git
cd luakit
go mod download
```

### Build

```bash
# Build CLI binary
go build -o luakit ./cmd/luakit

# Build gateway binary (for BuildKit frontend mode)
go build -o gateway ./cmd/gateway
```

### Test

```bash
go test ./...
```

### Run Examples

```bash
cd examples/simple
luakit build build.lua | buildctl build --no-frontend --local context=.
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details

## Acknowledgments

- [BuildKit](https://github.com/moby/buildkit) - The core build engine
- [gopher-lua](https://github.com/yuin/gopher-lua) - Lua VM for Go
- [LLB](https://github.com/moby/buildkit/tree/master/llb) - Low-Level Builder

## Links

- [Documentation](docs/README.md)
- [Examples](examples/)
- [Specification](SPEC.md)
- [GitHub Issues](https://github.com/kasuboski/luakit/issues)
- [Discussions](https://github.com/kasuboski/luakit/discussions)
