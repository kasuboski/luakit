# Lua Frontend for BuildKit LLB — Development Plan

**Project Codename:** `luakit`
**Version:** 0.1.0 (MVP)
**Date:** 2026-02-06
**Status:** Ready to implement

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Background & Motivation](#2-background--motivation)
3. [Architecture Overview](#3-architecture-overview)
4. [Lua API Specification](#4-lua-api-specification)
5. [LLB DAG Construction Internals](#5-llb-dag-construction-internals)
6. [Implementation Phases](#6-implementation-phases)
7. [Verification & Acceptance Criteria](#7-verification--acceptance-criteria)
8. [Risks & Mitigations](#8-risks--mitigations)
9. [Open Questions](#9-open-questions)
10. [Appendices](#10-appendices)

---

## 1. Executive Summary

This project delivers a Lua-based frontend for BuildKit that allows users to write container image build definitions as Lua scripts. The scripts construct a BuildKit LLB (Low-Level Builder) DAG which is serialized to protobuf and submitted to BuildKit for execution. The project ships as a Go CLI (`luakit`) that embeds a Lua VM, with a future path to packaging as a BuildKit gateway frontend image.

The primary value proposition is a lightweight, scriptable build definition language that exposes the full power of BuildKit's LLB (including `MergeOp`, `DiffOp`, cache mounts, secrets, and SSH forwarding) while remaining simpler than a full SDK (Dagger, Go client) and more expressive than Dockerfile.

---

## 2. Background & Motivation

### 2.1 How BuildKit Works

BuildKit separates *what to build* from *how to build it*. The internal representation is LLB — a directed acyclic graph defined via protobuf (see Appendix A). Each vertex is an `Op` (exec, source, file, merge, diff), and edges represent data dependencies between operations. BuildKit's solver traverses this DAG, executing independent branches in parallel and caching results content-addressably.

A *frontend* is anything that produces a serialized LLB `Definition` message. Dockerfile is the default frontend; others include Earthly (Earthfile), Dagger (Go/Python/TS SDKs), and HLB (Go-based DSL). Our contribution is a Lua frontend.

### 2.2 Why Lua

Lua occupies a unique niche for this use case:

- **Tiny and embeddable.** The reference Lua VM is ~200KB. Go embeddings (gopher-lua, golua) are mature. Startup is measured in microseconds, not seconds.
- **Minimal language surface.** ~50 keywords. A developer can learn enough Lua to be productive in an hour. This matters because build definitions should be *simple programs*, not software engineering exercises.
- **Proven as a configuration/extension language.** Nginx (OpenResty), Redis, Neovim, game engines (LÖVE, Roblox), and HAProxy all embed Lua for user-programmable behavior. Container builds are analogous — structured configuration with occasional logic.
- **Sandboxable by default.** Lua's standard library is opt-in. We can trivially remove `os.execute`, `io.open`, `loadfile`, and network access, ensuring build scripts cannot have side effects beyond constructing the DAG.
- **No dependency chain.** Unlike TypeScript/Python/Go SDKs, a `.lua` file has zero external dependencies. No `package.json`, no `go.mod`, no virtualenv. This aligns with the Unix philosophy of build definitions as plain text.

### 2.3 Comparison to Alternatives

| Frontend | Language | Startup | Dependencies | LLB Coverage | Learning Curve |
|----------|----------|---------|-------------|-------------|----------------|
| Dockerfile | Custom DSL | N/A (native) | None | Partial (no merge/diff) | Low |
| Dagger | Go/Python/TS | ~2-5s (container) | Full SDK + runtime | Full | Medium-High |
| Earthly | Earthfile DSL | ~1s | Earthly binary | Partial | Low-Medium |
| HLB | Custom DSL | ~100ms | HLB binary | Full | Medium |
| **luakit** | **Lua** | **~5ms** | **Single binary** | **Full** | **Low** |

---

## 3. Architecture Overview

### 3.1 High-Level Flow

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

### 3.2 Component Breakdown

**Component 1: `luakit` CLI (Go)**
- Entry point. Parses CLI flags, initializes Lua VM, registers API functions, evaluates user script, serializes result.
- Dependencies: `gopher-lua`, `moby/buildkit/client/llb`, `moby/buildkit/solver/pb`.

**Component 2: Lua API module (`bk`)**
- A set of Go functions registered into the Lua VM under the `bk` table.
- Each function either creates a new DAG node (source, exec, file op) or transforms an existing state.
- Returns Lua userdata wrapping a Go `*State` struct.

**Component 3: DAG Builder (Go)**
- Internal Go types representing the build graph.
- Each `State` holds a pointer to its parent `Op` node and output index.
- On `bk.export(state)`, walks the graph to produce a `pb.Definition`.

**Component 4: LLB Serializer (Go)**
- Converts the internal DAG to BuildKit's `pb.Definition` protobuf.
- Computes content-addressable digests for each `Op`.
- Attaches metadata (source locations, descriptions, cache hints).

**Component 5: Source Mapper (Go)**
- Maps Lua source file/line/column to `pb.Source` / `pb.SourceInfo` entries.
- Enables BuildKit's progress UI to show which Lua line a running operation came from.

### 3.3 Technology Choices

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Host language | Go | BuildKit's entire ecosystem is Go. Using Go lets us import `moby/buildkit/client/llb` directly rather than reimplementing protobuf serialization. |
| Lua VM | [gopher-lua](https://github.com/yuin/gopher-lua) | Pure Go, Lua 5.1 compatible, well-maintained, widely used (Terraform, etc.). Alternative: [golua](https://github.com/aarzilli/golua) (cgo, Lua 5.3) — rejected due to cgo cross-compilation complexity. |
| Protobuf handling | BuildKit's own Go types | We do NOT generate Lua protobuf bindings. The Lua layer builds a logical DAG; Go code handles all serialization. This avoids the biggest implementation risk. |
| Output mode (MVP) | Pipe to `buildctl` | `luakit build . | buildctl build --no-frontend --local context=.` or direct GRPC gateway client. |

### 3.4 Directory Structure

```
luakit/
├── cmd/
│   └── luakit/
│       └── main.go              # CLI entry point
├── pkg/
│   ├── luavm/
│   │   ├── vm.go                # Lua VM initialization, sandbox config
│   │   ├── api.go               # Register all bk.* functions into Lua
│   │   └── state.go             # Lua userdata wrapper for State
│   ├── dag/
│   │   ├── graph.go             # DAG data structures (Op, Edge, State)
│   │   ├── serialize.go         # Convert DAG → pb.Definition
│   │   ├── digest.go            # Content-addressable digest computation
│   │   └── sourcemap.go         # Lua line → pb.Source mapping
│   ├── ops/
│   │   ├── source.go            # image(), local_(), git(), http()
│   │   ├── exec.go              # run()
│   │   ├── file.go              # copy(), mkdir(), mkfile(), rm(), symlink()
│   │   ├── merge.go             # merge()
│   │   └── diff.go              # diff()
│   └── output/
│       ├── protobuf.go          # Write pb.Definition to stdout/file
│       ├── dot.go               # DAG → Graphviz DOT format
│       └── json.go              # DAG → JSON (debug)
├── lua/
│   ├── stdlib/
│   │   └── prelude.lua          # Lua-side helpers (syntactic sugar)
│   └── examples/
│       ├── simple.lua
│       ├── multistage.lua
│       ├── parallel.lua
│       └── cache_mounts.lua
├── test/
│   ├── integration/
│   │   ├── simple_test.go       # End-to-end: Lua script → pb.Definition → buildctl
│   │   ├── parallel_test.go
│   │   └── golden/              # Golden file pb.Definition outputs
│   ├── unit/
│   │   ├── api_test.go
│   │   ├── dag_test.go
│   │   └── serialize_test.go
│   └── lua/
│       ├── test_api.lua         # Lua-level tests using built-in assert
│       └── test_errors.lua
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 4. Lua API Specification

This section is the contract between the Lua script author and the `luakit` runtime. All functions are registered under the global `bk` table.

### 4.1 Source Operations

These create initial states with no parent dependencies. Each maps to a `SourceOp` in LLB.

#### `bk.image(ref, [opts])` → State

Pull a container image as a base state.

```lua
local base = bk.image("docker.io/library/ubuntu:24.04")
local base = bk.image("ubuntu:24.04")  -- shorthand, docker.io implied

-- With platform override
local arm = bk.image("ubuntu:24.04", {
    platform = "linux/arm64"
})

-- With resolve mode
local pinned = bk.image("ubuntu:24.04", {
    resolve = "always"  -- "default" | "always" | "never"
})
```

**LLB mapping:** `SourceOp{identifier: "docker-image://docker.io/library/ubuntu:24.04"}` with attrs for resolve mode.

#### `bk.local_(name, [opts])` → State

Reference a local build context directory. The trailing underscore avoids collision with Lua's `local` keyword.

```lua
local ctx = bk.local_("context")
local ctx = bk.local_("context", {
    include = {"*.go", "go.mod", "go.sum"},
    exclude = {"vendor/", "*.test"},
    shared_key_hint = "go-sources"
})
```

**LLB mapping:** `SourceOp{identifier: "local://context"}` with filter attrs.

#### `bk.git(url, [opts])` → State

Clone a git repository.

```lua
local repo = bk.git("https://github.com/moby/buildkit.git", {
    ref = "v0.12.0",         -- branch, tag, or commit
    keep_git_dir = false
})
```

**LLB mapping:** `SourceOp{identifier: "git://github.com/moby/buildkit.git#v0.12.0"}`

#### `bk.http(url, [opts])` → State

Fetch a remote file.

```lua
local file = bk.http("https://example.com/archive.tar.gz", {
    checksum = "sha256:abc123...",
    filename = "archive.tar.gz",
    chmod = 0644
})
```

**LLB mapping:** `SourceOp{identifier: "https://example.com/archive.tar.gz"}`

#### `bk.scratch()` → State

An empty filesystem state. Useful as a base for `copy` in minimal final images.

```lua
local empty = bk.scratch()
```

**LLB mapping:** `SourceOp` with empty identifier, or a sentinel recognized by BuildKit.

### 4.2 Exec Operations

#### `state:run(cmd, [opts])` → State

Execute a command on top of the current state. Returns a new state representing the filesystem after execution.

```lua
-- Simple string form (passed to /bin/sh -c)
local updated = base:run("apt-get update && apt-get install -y curl")

-- Array form (exec directly, no shell)
local updated = base:run({"apt-get", "update"})

-- With full options
local built = workspace:run("make -j$(nproc)", {
    cwd = "/app/src",
    user = "builder",
    env = {
        CC = "gcc",
        CFLAGS = "-O2",
    },
    network = "none",       -- "sandbox" (default) | "host" | "none"
    security = "sandbox",   -- "sandbox" (default) | "insecure"
    mounts = {
        bk.cache("/root/.cache/go-build", { sharing = "shared" }),
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.ssh(),
        bk.tmpfs("/tmp", { size = 1073741824 }),  -- 1GB
    },
    valid_exit_codes = {0, 1},  -- accept exit code 1 as success
    hostname = "builder",
})
```

**LLB mapping:** `ExecOp` with `Meta` populated from opts. Each mount maps to a `Mount` entry. The `cmd` string form wraps as `["/bin/sh", "-c", cmd]`.

**Mount helpers** (used inside `mounts` array):

```lua
-- Cache mount
bk.cache(dest, {
    id = "optional-namespace",        -- CacheOpt.ID
    sharing = "shared",               -- "shared" | "private" | "locked"
})

-- Secret mount
bk.secret(dest, {
    id = "secret-id",                 -- SecretOpt.ID, defaults to dest basename
    uid = 0,
    gid = 0,
    mode = 0400,
    optional = false,
})

-- SSH agent mount
bk.ssh({
    dest = "/run/ssh",                -- defaults to standard agent path
    id = "default",                   -- SSHOpt.ID
    uid = 0,
    gid = 0,
    mode = 0600,
    optional = false,
})

-- Tmpfs mount
bk.tmpfs(dest, {
    size = 67108864,                  -- 64MB
})

-- Bind mount from another state (cross-state read-only mount)
bk.bind(other_state, dest, {
    selector = "/specific/path",      -- Mount.selector
    readonly = true,
})
```

### 4.3 File Operations

These map to `FileOp` with `FileAction` variants.

#### `state:copy(from, src, dest, [opts])` → State

Copy files from one state to another.

```lua
-- Copy from another state
local final = runtime:copy(built, "/app/build/server", "/usr/local/bin/server")

-- With options
local final = runtime:copy(built, "/app/build/", "/app/", {
    owner = { user = "app", group = "app" },
    mode = "0755",
    follow_symlink = true,
    create_dest_path = true,
    include = {"*.so", "server"},
    exclude = {"*.a", "*.o"},
    allow_wildcard = true,
})
```

**LLB mapping:** `FileOp` containing `FileActionCopy`. The `from` state becomes `secondaryInput`.

#### `state:mkdir(path, [opts])` → State

```lua
local s = base:mkdir("/app/data", {
    mode = 0755,
    make_parents = true,
    owner = { user = 1000, group = 1000 },
})
```

**LLB mapping:** `FileOp` containing `FileActionMkDir`.

#### `state:mkfile(path, data, [opts])` → State

```lua
local s = base:mkfile("/etc/config.json", '{"key": "value"}', {
    mode = 0644,
    owner = { user = "root", group = "root" },
})
```

**LLB mapping:** `FileOp` containing `FileActionMkFile`.

#### `state:rm(path, [opts])` → State

```lua
local s = base:rm("/tmp/build-artifacts", {
    allow_not_found = true,
    allow_wildcard = true,
})
```

**LLB mapping:** `FileOp` containing `FileActionRm`.

#### `state:symlink(oldpath, newpath, [opts])` → State

```lua
local s = base:symlink("/usr/bin/python3", "/usr/bin/python")
```

**LLB mapping:** `FileOp` containing `FileActionSymlink`.

### 4.4 Graph Operations

#### `bk.merge(states...)` → State

Merge multiple states into one (overlay-style). Useful for combining independently-built layers.

```lua
local combined = bk.merge(go_modules, generated_proto, static_assets)
```

**LLB mapping:** `MergeOp` with an `Input` edge per state.

#### `bk.diff(lower, upper)` → State

Extract the filesystem diff between two states. Useful for extracting only what changed during a build step.

```lua
local base = bk.image("ubuntu:24.04")
local installed = base:run("apt-get update && apt-get install -y nginx")
local just_nginx = bk.diff(base, installed)

-- Apply just the nginx delta to a different base
local final = bk.merge(alpine, just_nginx)
```

**LLB mapping:** `DiffOp` with `LowerDiffInput` and `UpperDiffInput`.

### 4.5 Export & Metadata

#### `bk.export(state, [opts])`

Mark a state as the final output. Must be called exactly once.

```lua
bk.export(final)

-- With image config
bk.export(final, {
    entrypoint = {"/usr/local/bin/server"},
    cmd = {"--port", "8080"},
    env = {"PATH=/usr/local/bin:/usr/bin:/bin"},
    expose = {"8080/tcp"},
    user = "app",
    workdir = "/app",
    labels = {
        ["org.opencontainers.image.source"] = "https://github.com/myorg/myapp",
    },
})
```

#### `state:with_metadata(opts)` → State

Attach metadata to an operation for BuildKit's progress display.

```lua
local s = base:run("make"):with_metadata({
    description = "Compiling application",
    progress_group = "build",
})
```

**LLB mapping:** `OpMetadata` with `description` and `progress_group` fields.

### 4.6 Platform

#### `bk.platform(os, arch, [variant])` → Platform

Create a platform specifier for multi-platform builds.

```lua
local p = bk.platform("linux", "arm64", "v8")
local base = bk.image("ubuntu:24.04", { platform = p })

-- Or shorthand string form
local base = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
```

### 4.7 Composition via `require`

Standard Lua `require` works for loading modules from the build context. This enables shared build libraries.

```lua
-- libs/go.lua
local M = {}
function M.build(base, src, opts)
    local deps = base:run("go mod download", {
        cwd = opts.cwd or "/app",
        mounts = { bk.cache("/go/pkg/mod") },
    })
    local with_src = deps:copy(src, ".", opts.cwd or "/app")
    return with_src:run("go build -o /out/app " .. (opts.main or "."), {
        cwd = opts.cwd or "/app",
        mounts = { bk.cache("/root/.cache/go-build") },
    })
end
return M

-- build.lua
local go = require("libs/go")
local base = bk.image("golang:1.22")
local src = bk.local_("context")
local built = go.build(base, src, { main = "./cmd/server" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/app", "/server")
bk.export(final, { entrypoint = {"/server"} })
```

### 4.8 Error Handling

Lua errors are caught by the Go host and presented with Lua source file/line context. The API validates eagerly where possible.

```lua
-- This raises a Lua error immediately (not deferred to BuildKit)
bk.image("")  -- Error: build.lua:3: bk.image: identifier must not be empty

-- Invalid mount type
base:run("echo hi", {
    mounts = { "not a mount" }  -- Error: build.lua:5: bk.run: mounts[1] must be a mount object
})

-- Missing export
-- (detected after script evaluation)
-- Error: build.lua: no bk.export() call — nothing to build
```

---

## 5. LLB DAG Construction Internals

### 5.1 State Object

The core abstraction is `State`, a Go struct exposed to Lua as userdata.

```go
// State represents a filesystem state at a point in the build graph.
// It is immutable — each operation returns a new State.
type State struct {
    // op is the Op that produces this state.
    op *OpNode

    // outputIndex is which output of the Op this state represents.
    // Most ops have a single output (index 0), but FileOp can
    // have multiple via chained actions.
    outputIndex int

    // platform, if set, overrides the default platform for this state.
    platform *pb.Platform
}

// OpNode is a vertex in the DAG.
type OpNode struct {
    op       pb.Op
    metadata pb.OpMetadata
    inputs   []*Edge

    // For source mapping
    luaFile string
    luaLine int
}

// Edge represents a dependency from one OpNode to another.
type Edge struct {
    node        *OpNode
    outputIndex int
}
```

### 5.2 Digest Computation

Each `OpNode` must be assigned a content-addressable digest. This is critical because BuildKit's `Definition.metadata` is keyed by digest string, and BuildKit deduplicates Ops by digest.

```go
func (n *OpNode) Digest() digest.Digest {
    // 1. Marshal the pb.Op to bytes
    dt, _ := proto.Marshal(&n.op)
    // 2. Compute sha256 digest
    return digest.FromBytes(dt)
}
```

### 5.3 Serialization Walk

`bk.export(state)` triggers a depth-first walk of the DAG:

```
function serialize(state) → pb.Definition:
    visited = {}
    def = pb.Definition{}
    walk(state.op, visited, def)
    return def

function walk(node, visited, def):
    dig = node.Digest()
    if dig in visited: return
    visited[dig] = true
    for each input edge in node.inputs:
        walk(edge.node, visited, def)
    dt = marshal(node.op)
    def.def.append(dt)
    def.metadata[dig] = node.metadata
```

The final vertex in `def.def` is the terminal Op (the export target). BuildKit convention: the last entry in the `def` array is the output.

### 5.4 Source Mapping

When a Lua API function is called, the Go side captures the Lua call site:

```go
func luaCallSite(L *lua.LState) (file string, line int) {
    dbg, _ := L.GetStack(1)
    L.GetInfo("Sl", dbg)
    return dbg.Source, dbg.CurrentLine
}
```

This information is stored on `OpNode` and emitted as `pb.Source` / `pb.SourceInfo` in the Definition, enabling BuildKit to display progress like:

```
#4 [build.lua:12] run("make -j$(nproc)")
#4 DONE 42.3s
```

---

## 6. Implementation Phases

### Phase 1: Core Runtime (Weeks 1-3)

**Goal:** A Lua script can produce a valid `pb.Definition` that `buildctl` accepts and executes.

**Deliverables:**
1. Go project scaffolding with `gopher-lua` embedded
2. `State` and `OpNode` data structures
3. DAG serialization to `pb.Definition`
4. Lua API: `bk.image()`, `bk.scratch()`, `bk.local_()`
5. Lua API: `state:run()` with basic options (env, cwd, user)
6. Lua API: `bk.export()` (single output, no image config)
7. CLI: `luakit build build.lua` → writes `pb.Definition` to stdout
8. CLI: `luakit build build.lua | buildctl build --no-frontend --local context=.`
9. Basic error reporting with Lua source line context

**Exit criteria:**
```lua
-- This script must produce a runnable build:
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
```

### Phase 2: File Operations & Mounts (Weeks 4-5)

**Goal:** Full FileOp support and mount types.

**Deliverables:**
1. `state:copy()`, `state:mkdir()`, `state:mkfile()`, `state:rm()`, `state:symlink()`
2. Mount helpers: `bk.cache()`, `bk.secret()`, `bk.ssh()`, `bk.tmpfs()`, `bk.bind()`
3. Owner/permission support (`ChownOpt`, mode)
4. Include/exclude patterns for copy and local source

**Exit criteria:**
```lua
-- Multi-stage build with cache mount and file copy
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") },
})
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server", { mode = "0755" })
bk.export(final)
```

### Phase 3: Graph Operations & Advanced Features (Weeks 6-7)

**Goal:** Full LLB coverage including merge, diff, and all exec options.

**Deliverables:**
1. `bk.merge()` and `bk.diff()`
2. `bk.git()`, `bk.http()`
3. Network mode, security mode, valid exit codes
4. `bk.export()` with image config (entrypoint, cmd, env, expose, user, workdir, labels)
5. `state:with_metadata()` for progress display
6. Platform specification and multi-platform support
7. Source mapping (Lua line → BuildKit progress)

**Exit criteria:**
```lua
-- Parallel build with merge
local base = bk.image("node:20")
local deps = base:run("npm ci", { cwd = "/app", mounts = { bk.cache("/root/.npm") } })
local lint = deps:run("npm run lint", { cwd = "/app" })
local test = deps:run("npm run test", { cwd = "/app" })
local build = deps:run("npm run build", { cwd = "/app" })
-- lint and test are independent — BuildKit parallelizes them
local verified = bk.merge(lint, test, build)
bk.export(verified)
```

### Phase 4: Developer Experience (Weeks 8-9)

**Goal:** The tool is pleasant to use and debuggable.

**Deliverables:**
1. `luakit dag build.lua` — print DAG as Graphviz DOT
2. `luakit dag --format=json build.lua` — print DAG as JSON
3. `luakit validate build.lua` — dry-run without submitting to BuildKit
4. `luakit fmt build.lua` — opinionated Lua formatter (optional, could shell out to StyLua)
5. Comprehensive error messages with suggestions
6. `require()` resolution from build context directory
7. Sandbox hardening — remove `os`, `io`, `debug`, `loadfile` from Lua globals
8. `--progress` flag integration with BuildKit progress API
9. Prelude library (`lua/stdlib/prelude.lua`) with common patterns

### Phase 5: BuildKit Frontend Image (Week 10)

**Goal:** `luakit` works as a native BuildKit frontend.

**Deliverables:**
1. Dockerfile producing a minimal frontend image containing the `luakit` binary
2. Gateway GRPC integration — frontend reads `build.lua` from build context, returns `pb.Definition`
3. Users can invoke: `#syntax=myregistry/luakit:latest` or `buildctl build --frontend=gateway.v0 --opt source=myregistry/luakit:latest`
4. Frontend image published to a container registry

---

## 7. Verification & Acceptance Criteria

### 7.1 Unit Tests (Go)

Located in `test/unit/`. Run with `go test ./...`.

| Test Suite | What It Covers | Pass Criteria |
|------------|----------------|---------------|
| `api_test.go` | Each `bk.*` function registered in Lua VM | Function exists, returns correct userdata type, rejects invalid arguments with clear error |
| `dag_test.go` | DAG construction from sequences of API calls | Graph structure matches expected topology (node count, edge connectivity, op types) |
| `serialize_test.go` | `DAG → pb.Definition` conversion | Output matches golden protobuf files; digests are deterministic across runs |
| `sourcemap_test.go` | Lua line → `pb.Source` mapping | Source info correctly references Lua file/line for each Op |
| `sandbox_test.go` | Blocked Lua functions | `os.execute`, `io.open`, `loadfile`, `dofile` all raise errors |

### 7.2 Lua-Level Tests

Located in `test/lua/`. Run via `luakit test test/lua/`.

```lua
-- test/lua/test_api.lua

-- Source operations return State objects
local s = bk.image("alpine:3.19")
assert(type(s) == "userdata", "bk.image must return userdata")

-- Run returns a new state (immutability)
local s2 = s:run("echo hi")
assert(s ~= s2, "run must return a new state")

-- Copy requires a source state
local ok, err = pcall(function() s:copy("not a state", "/src", "/dst") end)
assert(not ok, "copy with non-state source must error")
assert(err:find("must be a state"), "error must mention state requirement")

-- Merge requires at least 2 states
local ok2, err2 = pcall(function() bk.merge(s) end)
assert(not ok2, "merge with 1 state must error")

-- Export can only be called once
bk.export(s)
local ok3, err3 = pcall(function() bk.export(s) end)
assert(not ok3, "double export must error")
```

### 7.3 Golden File Tests

Located in `test/integration/golden/`. Each test case is a pair: a `.lua` input and a `.pb.golden` expected output (or `.json.golden` for human-readable comparison).

| Test Case | Lua Input | Validates |
|-----------|-----------|-----------|
| `simple_image` | Single `bk.image()` + `bk.export()` | Minimal valid Definition with one SourceOp |
| `exec_basic` | `image → run → export` | ExecOp with correct Meta, input edge from SourceOp |
| `exec_mounts` | `run` with cache, secret, ssh, tmpfs | All mount types serialize correctly |
| `file_copy` | Two images + `copy` between them | FileOp with FileActionCopy, correct secondaryInput |
| `file_ops` | mkdir, mkfile, rm, symlink chain | All FileAction variants in a single FileOp |
| `merge_basic` | Three parallel branches merged | MergeOp with 3 inputs, independent DAG branches |
| `diff_basic` | diff(base, installed) | DiffOp with correct lower/upper |
| `multi_stage` | Builder → runtime copy pattern | Two SourceOps, ExecOp, FileOp(copy), correct edges |
| `platform` | Cross-platform image pull | Platform field set on Op |
| `complex_dag` | Real-world Go app build | Full DAG with parallel branches, cache mounts, multi-stage |

**Test execution:** `go test ./test/integration/ -update` regenerates golden files. CI runs without `-update` and fails on mismatch.

**Determinism requirement:** The same `.lua` input MUST produce byte-identical `pb.Definition` output across runs. This is verified by running golden tests twice and comparing.

### 7.4 Integration Tests (End-to-End)

These require a running BuildKit daemon. Run with `go test ./test/integration/ -tags=e2e`.

| Test | Description | Pass Criteria |
|------|-------------|---------------|
| `TestBuildSimple` | `alpine + echo` script | Build completes, output image contains `/greeting.txt` |
| `TestBuildMultiStage` | Go app build with distroless runtime | Output image contains only the binary, runs successfully |
| `TestBuildParallel` | Three independent branches | BuildKit log shows parallel execution of branches |
| `TestBuildCacheMount` | Two consecutive builds with cache mount | Second build is faster (cache hit on mount) |
| `TestBuildSecret` | Script using `bk.secret()` | Build succeeds when secret provided, fails descriptively when not |
| `TestBuildLocalContext` | Script referencing local files | Files from context directory appear in built image |
| `TestBuildGitSource` | `bk.git()` with public repo | Cloned content appears in build |
| `TestBuildMergeDiff` | merge + diff pattern | Resulting image has correct filesystem delta applied |
| `TestBuildImageConfig` | Export with entrypoint/cmd/env | `docker inspect` shows correct image config |
| `TestBuildFrontendImage` | Build via gateway frontend | `buildctl build --frontend=gateway.v0 --opt source=luakit:test` succeeds |

### 7.5 Error Handling Tests

| Scenario | Expected Behavior |
|----------|-------------------|
| Empty script (no `bk.export()`) | Error: "no bk.export() call — nothing to build" with suggestion |
| Syntax error in Lua | Lua parse error with file/line |
| `bk.image("")` | "bk.image: identifier must not be empty" at correct line |
| `state:run()` with no args | "bk.run: command argument required" |
| `bk.merge()` with 0 or 1 args | "bk.merge: requires at least 2 states" |
| `bk.diff()` with non-state arg | "bk.diff: arguments must be state objects" |
| Infinite loop in Lua | Timeout after configurable limit (default 30s) with "script evaluation timed out" |
| `require("nonexistent")` | "module 'nonexistent' not found" with search paths listed |
| `os.execute("rm -rf /")` | "os.execute is not available in build scripts (sandboxed)" |
| Mount type mismatch | "bk.run: mounts[2] must be a mount object, got string" |

### 7.6 Performance Benchmarks

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Script evaluation (100-line) | < 10ms | `go test -bench=BenchmarkEvalSimple` |
| Script evaluation (1000-line) | < 100ms | `go test -bench=BenchmarkEvalComplex` |
| DAG serialization (50 ops) | < 5ms | `go test -bench=BenchmarkSerialize` |
| CLI cold start to first output byte | < 50ms | `time luakit build simple.lua > /dev/null` |
| Memory usage (1000-op DAG) | < 50MB | Runtime profiling |

### 7.7 Definition of Done (Per Phase)

A phase is complete when ALL of the following are true:

1. **All unit tests pass.** `go test ./... -count=1` exits 0.
2. **All golden file tests pass.** No diff between generated and expected output.
3. **All Lua-level tests pass.** `luakit test test/lua/` exits 0.
4. **Integration tests pass** (for phases with e2e tests). Requires BuildKit daemon.
5. **No data races.** `go test -race ./...` exits 0.
6. **Linting passes.** `golangci-lint run` and `luacheck` (if installed) report no errors.
7. **Documentation updated.** README and example scripts reflect new functionality.
8. **Performance benchmarks meet targets.** No regressions from previous phase.

### 7.8 Project-Level Definition of Done

The project is shippable (v0.1.0) when:

1. Phases 1-4 are complete by the above criteria.
2. Phase 5 (frontend image) is functional and tested.
3. At least 3 real-world build scripts (from existing Dockerfiles) have been successfully ported to Lua and produce equivalent images.
4. A developer unfamiliar with the project can install `luakit`, read the README, and build a multi-stage Go or Node.js application within 30 minutes.
5. `luakit` binary cross-compiles for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64.

---

## 8. Risks & Mitigations

### 8.1 Technical Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| gopher-lua Lua 5.1 limitations (no integers, no goto) | Medium | High | Lua 5.1 is sufficient for build scripts. Integer semantics handled in Go layer. Document known Lua 5.1 quirks. |
| BuildKit internal Go API instability | High | Medium | Pin to a specific BuildKit release tag. Use only public `client/llb` and `solver/pb` packages. Avoid internal packages. |
| Protobuf digest computation mismatch with BuildKit | High | Low | Use BuildKit's own digest functions (`digest.FromBytes` on marshaled proto). Verify against `buildctl debug dump-llb` output. |
| Sandbox escape via Lua debug library | Medium | Low | Remove `debug` library entirely from VM. Test with fuzzing. |
| Performance regression on large DAGs | Low | Low | Benchmark suite catches regressions. DAG construction is O(n) in ops. |
| `gopher-lua` upstream abandonment | Medium | Low | Fork if needed; codebase is well-understood. Alternative: switch to Lua 5.4 via golua (cgo). |

### 8.2 Adoption Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| Lua unfamiliarity among Docker users | High | High | Excellent examples, a migration guide from Dockerfile, and a `luakit convert Dockerfile` tool (stretch goal). Lua's simplicity is an advantage — the learning curve is genuinely small. |
| Competition from established tools (Dagger, Earthly) | Medium | High | Position as complementary (lightweight, no daemon, no SDK). Target power users who want LLB access without Go boilerplate. |
| BuildKit frontend protocol changes | Medium | Low | BuildKit's gateway protocol has been stable for years. Monitor releases. |

---

## 9. Open Questions

These should be resolved by the team before or during Phase 1.

1. **Module resolution:** Should `require("foo")` search the build context directory, a `luakit_modules/` directory, or both? What about a registry for shared modules?

2. **Multi-output builds:** BuildKit supports multiple named outputs. Should `bk.export()` accept a name for multi-output scenarios, or should we use a different API like `bk.outputs({ web = state1, api = state2 })`?

3. **Build arguments / variables:** How should the user pass runtime values (like `--build-arg` in Docker)? Proposed: `bk.arg("NAME", default)` which reads from CLI flags or environment.

4. **Conditional platform builds:** For true multi-platform images (building for multiple architectures in one invocation), do we need a `bk.for_each_platform(platforms, function(plat) ... end)` construct, or is that a higher-level concern?

5. **Frontend image base:** Should the frontend image be `FROM scratch` (static Go binary) or `FROM alpine` (for debugging)? Likely scratch for production, alpine for dev.

6. **Naming:** Is `luakit` the right name? Alternatives: `lunabuild`, `luabk`, `buildlua`, `moonbuild`. Check for trademark conflicts.

---

## 10. Appendices

### Appendix A: LLB Protobuf Reference (Key Messages)

The full protobuf definition is provided separately (see `ops.proto`). Key messages relevant to this project:

- **`Op`**: Vertex in the DAG. Contains `inputs` (edges), one of `{ExecOp, SourceOp, FileOp, BuildOp, MergeOp, DiffOp}`, platform, and worker constraints.
- **`Definition`**: The complete graph. Contains `repeated bytes def` (marshaled Ops), `map<string, OpMetadata> metadata` (keyed by digest), and `Source` (source mapping).
- **`ExecOp`**: Command execution. Contains `Meta` (args, env, cwd, user, proxy, hosts, ulimit), mounts, network mode, security mode, secrets-as-env, and CDI devices.
- **`FileOp`**: File manipulation. Contains a sequence of `FileAction`s (copy, mkfile, mkdir, rm, symlink) that execute atomically.
- **`SourceOp`**: External data source. Identified by URI scheme (docker-image://, local://, git://, https://).
- **`MergeOp`**: Overlays multiple inputs.
- **`DiffOp`**: Computes filesystem delta between lower and upper.
- **`Mount`**: Describes how to mount a filesystem during ExecOp. Supports bind, secret, SSH, cache, and tmpfs types.

### Appendix B: Example — Porting a Real Dockerfile

**Original Dockerfile:**

```dockerfile
FROM golang:1.22-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

**Equivalent `build.lua`:**

```lua
local base = bk.image("golang:1.22-bookworm")
local src = bk.local_("context")

-- Copy dependency manifests first (better cache hit rate)
local deps = base:mkdir("/app"):copy(src, "go.mod", "/app/go.mod")
                               :copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app" })

-- Copy full source and build
local workspace = downloaded:copy(src, ".", "/app")
local built = workspace:run(
    "CGO_ENABLED=0 go build -o /out/server ./cmd/server",
    {
        cwd = "/app",
        mounts = { bk.cache("/root/.cache/go-build") },
    }
)

-- Minimal runtime image
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, {
    expose = {"8080/tcp"},
    entrypoint = {"/server"},
})
```

### Appendix C: CLI Reference (Planned)

```
luakit - Lua frontend for BuildKit

USAGE:
    luakit build [flags] <script>     Build from a Lua script
    luakit dag [flags] <script>       Print the LLB DAG without building
    luakit validate <script>          Validate a script without building
    luakit test <dir|file>            Run Lua test files
    luakit version                    Print version information

BUILD FLAGS:
    --output, -o <path>         Write pb.Definition to file (default: stdout)
    --context, -c <path>        Build context directory (default: .)
    --arg KEY=VALUE             Set a build argument (repeatable)
    --platform <os/arch>        Target platform (default: host)
    --progress <auto|plain|tty> Progress output format
    --timeout <duration>        Script evaluation timeout (default: 30s)
    --buildkit-addr <addr>      BuildKit daemon address (direct mode)

DAG FLAGS:
    --format <dot|json|text>    Output format (default: dot)

EXAMPLES:
    # Pipe to buildctl
    luakit build build.lua | buildctl build --no-frontend --local context=.

    # Direct mode (connects to BuildKit)
    luakit build --buildkit-addr unix:///run/buildkit/buildkitd.sock build.lua

    # Visualize DAG
    luakit dag build.lua | dot -Tsvg > dag.svg

    # Validate without building
    luakit validate build.lua
```
