# Owner and Permission Support for File Operations

This document describes how to control file ownership and permissions in luakit build scripts.

## Overview

Luakit provides full support for setting file ownership (`owner`) and permission modes (`mode`) in all file operations:
- `copy()`
- `mkdir()`
- `mkfile()`

This matches Dockerfile's `chown` and `chmod` capabilities.

## Owner Syntax

The `owner` option accepts a table with `user` and `group` fields. Each can be specified as either a string (name) or a number (ID).

### Examples

```lua
-- User and group by name
owner = { user = "appuser", group = "appgroup" }

-- User and group by ID
owner = { user = 1000, group = 1000 }

-- Mixed: user by name, group by ID
owner = { user = "appuser", group = 1000 }

-- User only (group unchanged)
owner = { user = "appuser" }

-- Group only (user unchanged)
owner = { group = "appgroup" }
```

## Mode Syntax

The `mode` option accepts permission modes in two formats:

### String Format (Recommended for Octal)

Use a string with leading zero for octal notation:

```lua
mode = "0755"  -- rwxr-xr-x
mode = "0644"  -- rw-r--r--
mode = "0600"  -- rw-------
mode = "0700"  -- rwx------
```

### Number Format

Use the decimal equivalent of the octal value:

```lua
mode = 493   -- 0755 octal = 493 decimal
mode = 420   -- 0644 octal = 420 decimal
mode = 384   -- 0600 octal = 384 decimal
mode = 448   -- 0700 octal = 448 decimal
```

**Note**: Lua doesn't support octal literals like Go (e.g., `0755` in Go is octal, but `0755` in Lua is decimal 755). For clarity and to avoid confusion, use the string format `"0755"` for octal values.

## File Operations

### copy()

Copy files from one state to another with ownership and mode:

```lua
local base = bk.image("alpine:3.19")
local source = bk.image("ubuntu:24.04")

local result = base:copy(source, "/src", "/dst", {
    mode = "0755",
    owner = { user = "appuser", group = "appgroup" },
    follow_symlink = true,
    create_dest_path = true
})
```

### mkdir()

Create directories with ownership and mode:

```lua
local base = bk.image("alpine:3.19")

local result = base:mkdir("/app/data", {
    mode = "0700",
    make_parents = true,
    owner = { user = "appuser", group = "appgroup" }
})
```

### mkfile()

Create files with ownership and mode:

```lua
local base = bk.image("alpine:3.19")

local result = base:mkfile("/app/config.json", '{"key":"value"}', {
    mode = "0600",
    owner = { user = "root", group = "root" }
})
```

## Complete Example

```lua
local base = bk.image("alpine:3.19")

-- Create application directory with appuser ownership
local app_dir = base:mkdir("/app", {
    mode = "0755",
    make_parents = true,
    owner = { user = 1000, group = 1000 }
})

-- Create configuration file with restricted permissions
local with_config = app_dir:mkfile("/app/config.json", '{"key":"value"}', {
    mode = "0600",
    owner = { user = "root", group = "root" }
})

-- Create executable script
local with_script = with_config:mkfile("/app/start.sh", "#!/bin/sh\n", {
    mode = "0755",
    owner = { user = 1000, group = 1000 }
})

-- Copy hosts file with specific ownership
local source = bk.image("ubuntu:24.04")
local final = with_script:copy(source, "/etc/hosts", "/app/hosts", {
    mode = "0644",
    owner = { user = "root", group = "root" }
})

bk.export(final, {
    user = "1000",
    workdir = "/app"
})
```

## Common Permission Modes

| Octal | String | Description                     |
|--------|--------|--------------------------------|
| 0755   | rwxr-xr-x | Read/write/execute for owner, read/execute for group/others |
| 0644   | rw-r--r-- | Read/write for owner, read-only for group/others |
| 0600   | rw------- | Read/write for owner only (private file) |
| 0700   | rwx------ | Read/write/execute for owner only (private directory) |
| 0777   | rwxrwxrwx | Full permissions for all |
| 0750   | rwxr-x--- | Owner full, group read/execute, others none |

## Comparison to Dockerfile

### Dockerfile

```dockerfile
RUN mkdir -p /app && \
    chown appuser:appgroup /app && \
    chmod 0755 /app
COPY --chown=appuser:appgroup --chmod=0755 src/ /app/
```

### Luakit (Equivalent)

```lua
local base = bk.image("alpine:3.19")
local src = bk.local_("context")

local app_dir = base:mkdir("/app", {
    mode = "0755",
    make_parents = true,
    owner = { user = "appuser", group = "appgroup" }
})

local result = app_dir:copy(src, ".", "/app", {
    mode = "0755",
    owner = { user = "appuser", group = "appgroup" }
})

bk.export(result)
```

## Best Practices

1. **Use string mode for octal values**: Prefer `"0755"` over `493` for clarity
2. **Use user/group names in development**: More readable than numeric IDs
3. **Use numeric IDs in production**: More portable across systems
4. **Set restrictive permissions for sensitive files**: Use `"0600"` for config files
5. **Set executable permissions for scripts**: Use `"0755"` for directories and scripts

## Implementation Notes

- Owner and mode options are optional
- When not specified, files inherit default ownership and mode from the parent image
- Changes are applied atomically during the BuildKit execution phase
- Both string and number mode formats are supported (string recommended for octal)
- User and group can be specified independently
