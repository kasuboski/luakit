# Lua require() Implementation

This document describes the implementation of Lua's `require()` function in luakit, which allows scripts to import helper modules and prelude libraries that ship with the project.

## Overview

The `require()` function in luakit resolves modules from two locations:

1. **Build Context Directory**: The directory containing the main build script
2. **Stdlib Directory**: The built-in standard library directory (typically `lua/stdlib`)

## Module Search Order

When a script calls `require("module_name")`, the following search paths are tried in order:

1. `{BuildContextDir}/module_name.lua`
2. `{BuildContextDir}/module_name/init.lua`
3. `{StdlibDir}/module_name.lua`
4. `{StdlibDir}/module_name/init.lua`

The first match is used, and the module is cached in `package.loaded` for subsequent requires.

## Example Usage

### Requiring from Build Context

Create a helper module in your project:

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
```

Use it in your build script:

```lua
-- build.lua
local go = require("libs.go")

local base = bk.image("golang:1.22")
local src = bk.local_("context")
local built = go.build(base, src, { main = "./cmd/server" })

local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/app", "/server")

bk.export(final, { entrypoint = {"/server"} })
```

### Requiring from Stdlib

The prelude module is included in the stdlib:

```lua
-- build.lua
local prelude = require("prelude")

local base = bk.image("alpine:3.19")
local result = base:run("echo hello")

bk.export(result)
```

## Stdlib Modules

### prelude.lua

Common build helpers and syntactic sugar:

```lua
local M = {}

function M.container(base, build_fn)
    return build_fn(base)
end

function M.multi_stage(builder_image, runtime_image, build_fn)
    local builder = bk.image(builder_image)
    local built = build_fn(builder)
    local runtime = bk.image(runtime_image)
    return runtime
end

function M.copy_all(source_state, from_path, to_path)
    return function(target_state)
        return target_state:copy(source_state, from_path, to_path)
    end
end

return M
```

## Implementation Details

### Custom Module Loader

A custom module loader is inserted at position 1 in `package.loaders`. This loader:

1. Searches for the module in the build context and stdlib directories
2. Reads the file content and registers it for source mapping
3. Compiles and executes the module using `L.Load()` and `L.PCall()`
4. Caches the result in `package.loaded`

If the loader cannot find the module, it returns 0, signaling to Lua that it should try the next loader in the chain.

### Package Path Configuration

The `package.path` is configured to include:

- `{BuildContextDir}/?.lua`
- `{BuildContextDir}/?/init.lua`
- `{StdlibDir}/?.lua`
- `{StdlibDir}/?/init.lua`
- Default Lua paths (for fallback compatibility)

### Stdlib Location

The stdlib directory is determined by:

1. The `LUAKIT_STDLIB_DIR` environment variable (if set)
2. A default location relative to the executable: `{exec_dir}/../share/luakit/stdlib`

If the directory does not exist, the stdlib is not used (no error).

## Testing

Comprehensive tests for require() functionality are included in `pkg/luavm/require_test.go`:

- Loading from build context
- Loading from stdlib
- Build context overriding stdlib
- Module caching
- Integration with BuildKit API
- Error handling for nonexistent modules

Run tests with:

```bash
go test ./pkg/luavm -run TestRequire -v
```
