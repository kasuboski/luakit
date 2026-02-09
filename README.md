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
go install github.com/kasuboski/luakit/cmd/luakit@latest
```

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

## Example: Go Multi-Stage Build

```lua
-- Builder stage
local builder = bk.image("golang:1.21-alpine")

local go_files = bk.local_("context", {
    include_patterns = { "go.mod", "go.sum" }
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

## CLI Usage

```bash
# Validate a script
luakit validate build.lua

# Visualize the build graph
luakit dag build.lua | dot -Tsvg > dag.svg

# Build an image
luakit build build.lua | buildctl build --no-frontend --local context=.

# Build with Docker
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  -t myapp:latest
```

## Comparison to Alternatives

| Frontend | Language | Startup | Dependencies | LLB Coverage | Learning Curve |
|----------|----------|---------|-------------|----------------|
| Dockerfile | Custom DSL | N/A (native) | None | Low |
| Dagger | Go/Python/TS | ~2-5s (container) | Full SDK | Medium-High |
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
build.lua  ──▶  luakit CLI  ──▶  LLB Definition (protobuf)  ──▶  BuildKit
                ┌─────────────┐
                │  Go binary   │
                │  ┌─────────┐ │
                │  │ Lua VM  │ │    Evaluates build.lua
                │  │(gopher- │ │    Lua calls Go-registered functions
                │  │  lua)   │ │    which build an internal DAG
                │  └────┬────┘ │
                │       │      │
                │  ┌────▼────┐ │
                │  │  DAG    │ │    In-memory graph of Op nodes
                │  │ Builder │ │    with edges (inputs)
                │  └────┬────┘ │
                │       │      │
                │  ┌────▼────┐ │
                │  │   LLB   │ │    Marshal DAG to pb.Definition
                │  │Serialize│ │    using BuildKit's Go types
                │  └────┬────┘ │
                └───────│──────┘
                        │
                        ▼
               buildctl / gateway GRPC
```

## Development

### Setup

```bash
git clone https://github.com/kasuboski/luakit.git
cd luakit
go mod download
```

### Build

```bash
go build -o luakit ./cmd/luakit
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

## Roadmap

- [ ] Build arguments (ARG)
- [ ] Health checks
- [ ] Volume declarations
- [ ] Stop signal
- [ ] Onbuild support
- [ ] Multi-output builds
- [ ] More prelude helpers

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
