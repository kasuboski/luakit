# Include/Exclude Pattern Examples

This directory demonstrates the use of .gitignore-style include/exclude patterns for copy and local operations in luakit.

## Pattern Syntax

Patterns use .gitignore-style matching:

- `*` matches any sequence of characters except `/`
- `**` matches any sequence of characters including `/`
- `?` matches any single character
- `/` anchored patterns match from the root
- Leading `!` negates a pattern (exclude from include)
- Trailing `/` matches directories only

## Examples

### Copy with Include/Exclude Patterns

```lua
-- Go project: copy only .go files, excluding tests and vendor
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app", {
    include = {"*.go", "go.mod", "go.sum"},
    exclude = {"*_test.go", "vendor/"}
})

-- Build the application
local built = workspace:run("go build -o /out/server ./cmd/server", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") }
})
```

### Local Source with Include/Exclude Patterns

```lua
-- Filter local context before using it
local ctx = bk.local_("context", {
    include = {
        "*.js",
        "*.json",
        "public/**/*",
        "src/**/*.ts"
    },
    exclude = {
        "node_modules/**/*",
        "*.test.js",
        ".git/",
        "coverage/"
    },
    shared_key_hint = "frontend-sources"
})
```

### Multi-Stage Build with Pattern Filtering

```lua
-- Builder stage: include all source files
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app", {
    include = {"*.go", "go.mod", "go.sum"},
    exclude = {"*_test.go", "vendor/"}
})

local built = workspace:run("go build -o /out/server ./cmd/server", {
    cwd = "/app",
    mounts = { bk.cache("/go/pkg/mod") }
})

-- Runtime stage: copy only the binary
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/server", "/server", {
    include = {"server"},
    mode = "0755"
})

bk.export(final, {
    entrypoint = {"/server"}
})
```

### Advanced Pattern Examples

```lua
-- Copy all YAML files recursively
local config = base:copy(src, "/config", "/etc/app", {
    include = {"**/*.yaml", "**/*.yml"}
})

-- Copy source files excluding test and generated files
local sources = base:copy(src, "/src", "/app/src", {
    include = {
        "**/*.go",
        "**/*.ts",
        "**/*.js"
    },
    exclude = {
        "**/*_test.go",
        "**/*_gen.go",
        "**/testdata/**/*",
        "**/*.test.ts",
        "**/node_modules/**/*"
    }
})

-- Copy static assets with directory exclusions
local assets = base:copy(src, "/static", "/var/www", {
    include = {
        "*.html",
        "*.css",
        "*.js",
        "images/**/*",
        "fonts/**/*"
    },
    exclude = {
        "*.min.css",
        "*.min.js",
        "images/**/*.tmp"
    }
})
```

## Pattern Matching Rules

1. **Include patterns are evaluated first** - only files matching at least one include pattern are considered
2. **Exclude patterns filter the included files** - files matching exclude patterns are removed
3. **If no include patterns are specified**, all files are considered (subject to exclude)
4. **Patterns are relative to the source path** specified in the copy operation
5. **Directory patterns must end with `/`** to match only directories

## Common Patterns

| Pattern | Description |
|---------|-------------|
| `*.go` | All .go files in current directory |
| `**/*.go` | All .go files in any subdirectory |
| `*_test.go` | All test files |
| `vendor/` | Exclude vendor directory |
| `node_modules/**/*` | Exclude all node_modules contents |
| `.git/` | Exclude .git directory |
| `!config.yaml` | Include config.yaml (negation) |
| `/app/*.so` | Only .so files in /app directory |

## Dockerfile Equivalent

```dockerfile
# Dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /out/server ./cmd/server

FROM alpine:3.19
COPY --from=builder /out/server /server
ENTRYPOINT ["/server"]
```

```lua
-- Equivalent Lua
local builder = bk.image("golang:1.22")
local src = bk.local_("context")

local deps = builder:copy(src, "go.mod", "/app/go.mod")
                            :copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app" })

local workspace = downloaded:copy(src, "*.go", "/app", {
    exclude = {"*_test.go"}
})
local built = workspace:run("go build -o /out/server ./cmd/server", {
    cwd = "/app"
})

local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/server", "/server")

bk.export(final, { entrypoint = {"/server"} })
```

## Performance Tips

1. Use `shared_key_hint` for local sources to improve cache hit rates
2. Combine multiple includes into fewer patterns when possible
3. Exclude large directories early (e.g., `vendor/`, `node_modules/`)
4. Use `allow_wildcard: true` when copying entire directory trees
