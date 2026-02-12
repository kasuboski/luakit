# API Reference

Complete reference for all Luakit API functions.

## Table of Contents

- [Source Operations](#source-operations)
- [Exec Operations](#exec-operations)
- [File Operations](#file-operations)
- [Graph Operations](#graph-operations)
- [Mount Helpers](#mount-helpers)
- [Export & Metadata](#export--metadata)
- [Platform](#platform)

## Source Operations

### bk.image(ref, [opts]) → State

Pull a container image as a base state.

**Parameters:**

- `ref` (string): Image reference (e.g., `"alpine:3.19"`, `"golang:1.21"`)
- `opts` (table, optional): Options table

**Options:**

- `platform` (string|table|platform): Target platform

**Returns:** A new State

**Examples:**

```lua
-- Simple image
local alpine = bk.image("alpine:3.19")

-- With platform
local arm = bk.image("alpine:3.19", {
    platform = "linux/arm64"
})

-- Docker Hub implied
local ubuntu = bk.image("ubuntu:24.04")  -- docker.io/library/ubuntu:24.04
```

**LLB mapping:** `SourceOp{identifier: "docker-image://..."}`

---

### bk.local_(name, [opts]) → State

Reference a local build context directory. The trailing underscore avoids collision with Lua's `local` keyword.

**Parameters:**

- `name` (string): Context name (used with `--local name=<path>` in buildctl)
- `opts` (table, optional): Options table

**Options:**

- `include` (table of strings): Include patterns (e.g., `{"*.go", "go.mod"}`)
- `exclude` (table of strings): Exclude patterns (e.g., `{"vendor/", "*.test"}`)
- `shared_key_hint` (string): Cache sharing hint

**Returns:** A new State

**Examples:**

```lua
-- Entire context
local ctx = bk.local_("context")

-- With patterns
local src = bk.local_("context", {
    include = {"*.go", "go.mod", "go.sum"},
    exclude = {"vendor/", "*.test"}
})
```

**LLB mapping:** `SourceOp{identifier: "local://name"}`

---

### bk.git(url, [opts]) → State

Clone a git repository.

**Parameters:**

- `url` (string): Git repository URL
- `opts` (table, optional): Options table

**Options:**

- `ref` (string): Branch, tag, or commit (default: main/master)
- `keep_git_dir` (boolean): Keep .git directory (default: false)

**Returns:** A new State

**Examples:**

```lua
-- Clone default branch
local repo = bk.git("https://github.com/moby/buildkit.git")

-- Specific tag
local v012 = bk.git("https://github.com/moby/buildkit.git", {
    ref = "v0.12.0"
})

-- Keep git directory
local full = bk.git("https://github.com/user/project.git", {
    ref = "main",
    keep_git_dir = true
})
```

**LLB mapping:** `SourceOp{identifier: "git://..."}`

---

### bk.http(url, [opts]) → State
### bk.https(url, [opts]) → State

Fetch a remote file via HTTP/HTTPS.

**Parameters:**

- `url` (string): File URL
- `opts` (table, optional): Options table

**Options:**

- `checksum` (string): SHA256 checksum (`sha256:abc123...`)
- `filename` (string): Output filename
- `chmod` (number): File permissions (octal string or decimal)
- `headers` (table): HTTP headers
- `username` (string): Basic auth username
- `password` (string): Basic auth password

**Returns:** A new State

**Examples:**

```lua
-- Simple download
local file = bk.http("https://example.com/archive.tar.gz")

-- With checksum and permissions
local file = bk.http("https://example.com/app", {
    checksum = "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
    filename = "app",
    chmod = 0755
})

-- With auth
local file = bk.https("https://internal.company.com/file.tar.gz", {
    username = "builder",
    password = "secret"
})
```

**LLB mapping:** `SourceOp{identifier: "https://..."}`

---

### bk.scratch() → State

An empty filesystem state.

**Parameters:** None

**Returns:** A new State

**Examples:**

```lua
-- Base for minimal images
local empty = bk.scratch()

-- Copy only what's needed
local result = empty:copy(built, "/app/server", "/server")
```

**LLB mapping:** `SourceOp` with empty identifier

---

## Exec Operations

### state:run(cmd, [opts]) → State

Execute a command on the current state.

**Parameters:**

- `cmd` (string|table): Command string or array
- `opts` (table, optional): Options table

**Options:**

- `env` (table): Environment variables (`{VAR="value", ...}`)
- `cwd` (string): Working directory
- `user` (string): Username or UID
- `network` (string): Network mode (`"sandbox"`, `"host"`, `"none"`)
- `security` (string): Security mode (`"sandbox"`, `"insecure"`)
- `mounts` (table of mount objects): Mounts (cache, secret, ssh, tmpfs, bind)
- `hostname` (string): Container hostname
- `valid_exit_codes` (number|table|string): Acceptable exit codes

**Returns:** A new State

**Examples:**

```lua
-- String form (via shell)
local result = base:run("apt-get update && apt-get install -y curl")

-- Array form (direct exec)
local result = base:run({"apt-get", "update"})

-- Full options
local result = base:run("make -j$(nproc)", {
    cwd = "/app/src",
    env = {
        CC = "gcc",
        CFLAGS = "-O2"
    },
    user = "builder",
    network = "none",
    security = "sandbox",
    mounts = {
        bk.cache("/root/.cache/go-build", { sharing = "shared" }),
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.ssh(),
        bk.tmpfs("/tmp", { size = 1073741824 })
    },
    hostname = "builder",
    valid_exit_codes = {0, 1}
})

-- Exit code range
local result = base:run("mise run test", {
    valid_exit_codes = "0..10"
})
```

**Exit codes:**

- `0` (number): Only exit code 0
- `{0, 1}` (table): Exit codes 0 or 1
- `"0..10"` (string): Exit codes 0 through 10

**LLB mapping:** `ExecOp`

---

## File Operations

### state:copy(from, src, dest, [opts]) → State

Copy files from one state to another.

**Parameters:**

- `from` (State): Source state
- `src` (string): Source path
- `dest` (string): Destination path
- `opts` (table, optional): Options table

**Options:**

- `mode` (string|number): File permissions (octal string recommended)
- `owner` (table): Owner specification
- `follow_symlink` (boolean): Follow symlinks
- `create_dest_path` (boolean): Create destination directories
- `allow_wildcard` (boolean): Allow wildcards in src
- `include` (table): Include patterns
- `exclude` (table): Exclude patterns

**Owner format:**

```lua
owner = {
    user = "appuser",  -- or user = 1000
    group = "appgroup" -- or group = 1000
}
```

**Examples:**

```lua
-- Simple copy
local final = runtime:copy(built, "/app/build/server", "/usr/local/bin/server")

-- With ownership and permissions
local final = runtime:copy(built, "/app/build/", "/app/", {
    mode = "0755",
    owner = { user = "app", group = "app" }
})

-- Wildcards and patterns
local final = base:copy(src, "src/*.lua", "/app/", {
    allow_wildcard = true,
    include = {"*.lua"},
    exclude = {"*_test.lua"}
})
```

**LLB mapping:** `FileOp` containing `FileActionCopy`

---

### state:mkdir(path, [opts]) → State

Create a directory.

**Parameters:**

- `path` (string): Directory path
- `opts` (table, optional): Options table

**Options:**

- `mode` (string|number): Directory permissions (octal string recommended)
- `make_parents` (boolean): Create parent directories
- `owner` (table): Owner specification

**Examples:**

```lua
-- Simple directory
local s = base:mkdir("/app/data")

-- With options
local s = base:mkdir("/app/data", {
    mode = "0755",
    make_parents = true,
    owner = { user = "appuser", group = "appgroup" }
})
```

**LLB mapping:** `FileOp` containing `FileActionMkDir`

---

### state:mkfile(path, data, [opts]) → State

Create a file with content.

**Parameters:**

- `path` (string): File path
- `data` (string): File content
- `opts` (table, optional): Options table

**Options:**

- `mode` (string|number): File permissions (octal string recommended)
- `owner` (table): Owner specification

**Examples:**

```lua
-- Simple file
local s = base:mkfile("/app/config.json", '{"key": "value"}')

-- With permissions
local s = base:mkfile("/app/config.json", '{"key": "value"}', {
    mode = "0644",
    owner = { user = "root", group = "root" }
})

-- Script with execute bit
local s = base:mkfile("/app/start.sh", "#!/bin/sh\necho starting", {
    mode = "0755"
})
```

**LLB mapping:** `FileOp` containing `FileActionMkFile`

---

### state:rm(path, [opts]) → State

Remove files or directories.

**Parameters:**

- `path` (string): Path to remove
- `opts` (table, optional): Options table

**Options:**

- `allow_not_found` (boolean): Don't error if path doesn't exist
- `allow_wildcard` (boolean): Allow wildcards in path

**Examples:**

```lua
-- Simple removal
local s = base:rm("/tmp/build-artifacts")

-- With wildcards
local s = base:rm("/tmp/*.o", {
    allow_wildcard = true
})

-- Don't error if missing
local s = base:rm("/opt/optional", {
    allow_not_found = true
})
```

**LLB mapping:** `FileOp` containing `FileActionRm`

---

### state:symlink(oldpath, newpath, [opts]) → State

Create a symbolic link.

**Parameters:**

- `oldpath` (string): Link target
- `newpath` (string): Link location

**Examples:**

```lua
-- Create symlink
local s = base:symlink("/usr/bin/python3", "/usr/bin/python")

-- Node.js version symlink
local s = base:symlink("/usr/local/bin/node-v20", "/usr/local/bin/node")
```

**LLB mapping:** `FileOp` containing `FileActionSymlink`

---

## Graph Operations

### bk.merge(states...) → State

Merge multiple states into one (overlay-style).

**Parameters:**

- `states...` (State): Two or more states to merge

**Returns:** A new State

**Examples:**

```lua
-- Merge three states
local combined = bk.merge(deps, source, config)

-- Parallel builds then merge
local lint = base:run("npm run lint", { cwd = "/app" })
local test = base:run("npm run test", { cwd = "/app" })
local build = base:run("npm run build", { cwd = "/app" })
local verified = bk.merge(lint, test, build)
```

**Behavior:** Later states overlay earlier states (files in later states win)

**LLB mapping:** `MergeOp`

---

### bk.diff(lower, upper) → State

Extract filesystem differences between two states.

**Parameters:**

- `lower` (State): Base state
- `upper` (State): Modified state

**Returns:** A new State containing only the differences

**Examples:**

```lua
-- Extract just what changed
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache git vim")
local just_git_vim = bk.diff(base, installed)

-- Apply diff to different base
local alpine = bk.image("alpine:3.19")
local with_git_vim = bk.merge(alpine, just_git_vim)
```

**Use cases:**

- Reuse changes across different base images
- Extract minimal layers
- Create reusable component layers

**LLB mapping:** `DiffOp`

---

## Mount Helpers

### bk.cache(dest, [opts]) → Mount

Create a cache mount.

**Parameters:**

- `dest` (string): Mount destination path
- `opts` (table, optional): Options table

**Options:**

- `id` (string): Cache namespace ID
- `sharing` (string): Sharing mode (`"shared"`, `"private"`, `"locked"`)

**Returns:** A Mount object (used in `mounts` option)

**Examples:**

```lua
-- Simple cache
local result = base:run("go build", {
    mounts = { bk.cache("/go/pkg/mod") }
})

-- Named cache with sharing
local result = base:run("go build", {
    mounts = {
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" })
    }
})

-- Multiple caches
local result = base:run("make", {
    mounts = {
        bk.cache("/go/pkg/mod", { id = "gomod" }),
        bk.cache("/root/.cache/go-build", { id = "gobuild" })
    }
})
```

**Sharing modes:**

- `shared`: Multiple builds can access simultaneously
- `private`: Exclusive to this build
- `locked`: Sequential builds (prevents conflicts)

**LLB mapping:** `Mount{type: cache}`

---

### bk.secret(dest, [opts]) → Mount

Create a secret mount.

**Parameters:**

- `dest` (string): Mount destination path
- `opts` (table, optional): Options table

**Options:**

- `id` (string): Secret identifier (default: basename of dest)
- `uid` (number): Owner UID
- `gid` (number): Owner GID
- `mode` (number): File permissions
- `optional` (boolean): Don't fail if secret not provided

**Returns:** A Mount object

**Examples:**

```lua
-- Simple secret
local result = base:run("cat /run/secrets/password", {
    mounts = { bk.secret("/run/secrets/password") }
})

-- With options
local result = base:run({
    "sh", "-c",
    "cat /run/secrets/npmrc > ~/.npmrc && npm ci"
}, {
    mounts = {
        bk.secret("/run/secrets/npmrc", {
            id = "npmrc",
            uid = 0,
            gid = 0,
            mode = 0400
        })
    }
})
```

**LLB mapping:** `Mount{type: secret}`

---

### bk.ssh([opts]) → Mount

Create an SSH agent mount.

**Parameters:**

- `opts` (table, optional): Options table

**Options:**

- `dest` (string): Mount destination (default: `/run/ssh`)
- `id` (string): SSH agent ID (default: `"default"`)
- `uid` (number): Owner UID
- `gid` (number): Owner GID
- `mode` (number): Permissions
- `optional` (boolean): Don't fail if SSH unavailable

**Returns:** A Mount object

**Examples:**

```lua
-- Simple SSH
local result = base:run("git clone git@github.com:user/repo.git", {
    mounts = { bk.ssh() }
})

-- Custom destination
local result = base:run("git clone git@github.com:user/repo.git", {
    mounts = { bk.ssh({ dest = "/root/.ssh" }) }
})

-- With options
local result = base:run("go mod download", {
    mounts = {
        bk.ssh({
            id = "default",
            uid = 0,
            gid = 0,
            mode = 0600
        })
    }
})
```

**LLB mapping:** `Mount{type: ssh}`

---

### bk.tmpfs(dest, [opts]) → Mount

Create a tmpfs (in-memory filesystem) mount.

**Parameters:**

- `dest` (string): Mount destination path
- `opts` (table, optional): Options table

**Options:**

- `size` (number): Size in bytes

**Returns:** A Mount object

**Examples:**

```lua
-- Simple tmpfs
local result = base:run("mise run test", {
    mounts = { bk.tmpfs("/tmp") }
})

-- Sized tmpfs (1GB)
local result = base:run("mise run test", {
    mounts = { bk.tmpfs("/tmp", { size = 1073741824 }) }
})

-- Multiple tmpfs
local result = base:run("mise run test", {
    mounts = {
        bk.tmpfs("/tmp", { size = 1073741824 }),
        bk.tmpfs("/var/tmp", { size = 536870912 })
    }
})
```

**LLB mapping:** `Mount{type: tmpfs}`

---

### bk.bind(from_state, dest, [opts]) → Mount

Create a bind mount from another state.

**Parameters:**

- `from_state` (State): Source state
- `dest` (string): Mount destination path
- `opts` (table, optional): Options table

**Options:**

- `selector` (string): Subpath within source state
- `readonly` (boolean): Mount read-only (default: true)

**Returns:** A Mount object

**Examples:**

```lua
-- Simple bind
local other = bk.image("ubuntu:24.04")
local result = base:run("cat /mounted/file.txt", {
    mounts = { bk.bind(other, "/mounted") }
})

-- Read-write
local result = base:run("echo data > /mounted/file.txt", {
    mounts = { bk.bind(other, "/mounted", { readonly = false }) }
})

-- Select subpath
local result = base:run("cat /mounted/config.json", {
    mounts = { bk.bind(other, "/mounted", { selector = "/etc" }) }
})
```

**LLB mapping:** `Mount{type: bind}`

---

## Export & Metadata

### bk.export(state, [opts])

Mark a state as the final output. Must be called exactly once per script.

**Parameters:**

- `state` (State): Final state to export
- `opts` (table, optional): Image configuration options

**Options:**

- `entrypoint` (table): Entrypoint command
- `cmd` (table): Default command
- `env` (table): Environment variables
- `workdir` (string): Working directory
- `user` (string): Default user
- `expose` (table): Exposed ports
- `labels` (table): Image labels
- `os` (string): OS (default: "linux")
- `arch` (string): Architecture (default: "amd64")
- `variant` (string): Architecture variant

**Examples:**

```lua
-- Simple export
bk.export(final)

-- Full configuration
bk.export(final, {
    entrypoint = {"/app/server"},
    cmd = {"--port", "8080"},
    env = {
        NODE_ENV = "production",
        PORT = "8080"
    },
    workdir = "/app",
    user = "app",
    expose = {"8080/tcp", "8443/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "My Application",
        ["org.opencontainers.image.version"] = "1.0.0",
        ["org.opencontainers.image.source"] = "https://github.com/user/app"
    },
    os = "linux",
    arch = "amd64"
})
```

**Behavior:**

- Must be called exactly once
- Returns nothing
- Configures the final image metadata

---

## Platform

### bk.platform(os, arch, [variant]) → Platform

Create a platform specifier.

**Parameters:**

- `os` (string): OS (e.g., "linux", "darwin")
- `arch` (string): Architecture (e.g., "amd64", "arm64")
- `variant` (string, optional): Architecture variant (e.g., "v8")

**Returns:** A Platform object

**Examples:**

```lua
-- Function call
local p = bk.platform("linux", "arm64", "v8")
local arm = bk.image("alpine:3.19", { platform = p })

-- String shorthand (in other functions)
local arm = bk.image("alpine:3.19", {
    platform = "linux/arm64/v8"
})
```

**Common platforms:**

- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`
- `linux/386`
- `linux/ppc64le`
- `linux/s390x`
