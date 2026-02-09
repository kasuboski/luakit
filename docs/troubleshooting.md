# Troubleshooting

Common issues and solutions when using Luakit.

## Table of Contents

- [Common Errors](#common-errors)
- [Debugging](#debugging)
- [Performance Issues](#performance-issues)
- [Build Issues](#build-issues)
- [Environment Issues](#environment-issues)

## Common Errors

### Error: bk.image: identifier must not be empty

**Problem:** Empty or whitespace-only image reference.

**Solution:**

```lua
-- Bad
local img = bk.image("")

-- Good
local img = bk.image("alpine:3.19")
```

---

### Error: bk.local_: name must not be empty

**Problem:** Empty local context name.

**Solution:**

```lua
-- Bad
local ctx = bk.local_("")

-- Good
local ctx = bk.local_("context")
```

---

### Error: bk.run: command argument required

**Problem:** No command provided to `state:run()`.

**Solution:**

```lua
-- Bad
local result = base:run()

-- Good
local result = base:run("echo hello")
```

---

### Error: bk.merge: requires at least 2 states

**Problem:** Trying to merge fewer than 2 states.

**Solution:**

```lua
-- Bad
local merged = bk.merge(state1)

-- Good
local merged = bk.merge(state1, state2)
```

---

### Error: bk.diff: requires lower and upper state arguments

**Problem:** Missing or invalid arguments to `bk.diff()`.

**Solution:**

```lua
-- Bad
local diffed = bk.diff(base)

-- Good
local diffed = bk.diff(base, modified)
```

---

### Error: bk.export: already called once

**Problem:** Calling `bk.export()` multiple times.

**Solution:**

```lua
-- Bad
bk.export(final)
bk.export(other)

-- Good
local merged = bk.merge(final, other)
bk.export(merged)
```

---

### Error: no bk.export() call — nothing to build

**Problem:** Script doesn't call `bk.export()`.

**Solution:**

```lua
-- Bad
local result = base:run("echo hello")

-- Good
local result = base:run("echo hello")
bk.export(result)
```

---

### Error: run: command must be string or table

**Problem:** Invalid command type.

**Solution:**

```lua
-- Bad
local result = base:run(123)

-- Good
local result = base:run("echo hello")
local result = base:run({"echo", "hello"})
```

---

### Error: module 'xyz' not found

**Problem:** Module not found in build context or stdlib.

**Solution:**

```lua
-- Check module exists
ls -la libs/xyz.lua

-- Use correct path
local xyz = require("libs.xyz")

-- Or install to stdlib
cp xyz.lua /path/to/stdlib/
```

---

### Error: unexpected symbol near '<something>'

**Problem:** Lua syntax error.

**Solution:**

```lua
-- Bad: Missing comma
local t = { a = 1 b = 2 }

-- Good
local t = { a = 1, b = 2 }

-- Bad: Missing end
if condition then
    do_something()

-- Good
if condition then
    do_something()
end
```

---

### Error: os.execute is not available

**Problem:** Trying to use disabled Lua function.

**Solution:**

```lua
-- Bad: os.execute is disabled
local result = os.execute("ls")

-- Good: Use bk.run() instead
local result = base:run("ls")
```

---

## Debugging

### Validate Script Syntax

Check script before building:

```bash
luakit validate build.lua
```

### Visualize DAG

Inspect build structure:

```bash
luakit dag build.lua | dot -Tsvg > dag.svg
```

Open `dag.svg` in a browser to see:
- Operation types and dependencies
- Parallel execution opportunities
- Layer structure

### Debug Mode with buildctl

Get verbose output:

```bash
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --debug
```

### Inspect Intermediate States

Use `luakit dag` with filtering:

```bash
# Show only Exec operations
luakit dag --filter=Exec build.lua

# Show only Source operations
luakit dag --filter=Source build.lua
```

### Check Lua Error Location

Error messages include file and line:

```
build.lua:15: bk.run: command argument required
```

Open file and check line 15.

### Add Debug Output

Use `print()` for debugging:

```lua
local base = bk.image("alpine:3.19")
print("Base image created")

local result = base:run("echo hello")
print("Result state created")

bk.export(result)
```

Output shows when lines execute.

### Step-by-Step Build

Break script into smaller parts:

```lua
-- Part 1
local base = bk.image("alpine:3.19")
bk.export(base)

-- Test with luakit validate part1.lua

-- Part 2
local result = base:run("echo hello")
bk.export(result)

-- Test with luakit validate part2.lua
```

---

## Performance Issues

### Slow Build Times

**Cause:** Poor caching or sequential operations.

**Solution:**

1. **Add cache mounts:**

```lua
local result = base:run("npm install", {
    mounts = { bk.cache("/root/.npm") }
})
```

2. **Copy dependency files first:**

```lua
-- Good: Copy package.json first
local pkg = base:copy(bk.local_("context", {
    include_patterns = { "package*.json" }
}), ".", "/app")
```

3. **Use parallel operations:**

```lua
local linted = base:run("npm run lint")
local tested = base:run("npm run test")
local verified = bk.merge(linted, tested)
```

### Large Image Size

**Cause:** Including unnecessary files or tools.

**Solution:**

1. **Use multi-stage builds:**

```lua
-- Builder with tools
local builder = bk.image("golang:1.21")
local built = builder:copy(src, ".", "/app")
    :run("go build -o /out/app .", { cwd = "/app" })

-- Runtime without tools
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/app/server")
```

2. **Use minimal base images:**

```lua
-- Good
local runtime = bk.image("alpine:3.19")

-- Better
local runtime = bk.image("gcr.io/distroless/static-debian12")

-- Best
local runtime = bk.scratch()
```

3. **Clean up in same layer:**

```lua
local result = base:run({
    "sh", "-c",
    "apt-get update && " ..
    "apt-get install -y curl && " ..
    "rm -rf /var/lib/apt/lists/*"
})
```

### Cache Not Working

**Cause:** Cache mount not configured or incorrect path.

**Solution:**

1. **Verify cache path:**

```bash
# Check common cache paths
ls -la ~/.cache/pip
ls -la ~/.npm
ls -la ~/go/pkg/mod
```

2. **Use correct cache mount:**

```lua
-- Python
local result = base:run("pip install -r requirements.txt", {
    mounts = { bk.cache("/root/.cache/pip") }
})

-- Node.js
local result = base:run("npm install", {
    mounts = { bk.cache("/root/.npm") }
})

-- Go
local result = base:run("go build", {
    mounts = { bk.cache("/go/pkg/mod") }
})
```

3. **Clear cache if needed:**

```bash
buildctl prune
```

---

## Build Issues

### Command Not Found

**Cause:** Command not available in base image.

**Solution:**

```lua
-- Bad: apk in Debian
local debian = bk.image("debian:12")
local result = debian:run("apk add curl")

-- Good: Use apt in Debian
local debian = bk.image("debian:12")
local result = debian:run("apt-get update && apt-get install -y curl")

-- Or use Alpine with apk
local alpine = bk.image("alpine:3.19")
local result = alpine:run("apk add --no-cache curl")
```

### Permission Denied

**Cause:** Running as non-root without proper setup.

**Solution:**

```lua
-- Create user first
local with_user = base:run({
    "sh", "-c",
    "groupadd -r app && " ..
    "useradd -r -g app -u 1000 app && " ..
    "chown -R app:app /app"
})

-- Then use non-root user
local result = with_user:run("npm install", {
    cwd = "/app",
    user = "app"
})
```

### File Not Found

**Cause:** Incorrect copy path or file not in context.

**Solution:**

```lua
-- Check what's in context
local src = bk.local_("context")

-- Use absolute paths
local result = base:copy(src, "/app/src/file.txt", "/app/dest.txt")

-- Or use correct relative paths
local result = base:copy(src, "src/file.txt", "/app/")
```

### Exit Code Non-Zero

**Cause:** Command failed.

**Solution:**

```lua
-- Accept specific exit codes
local result = base:run("make lint", {
    valid_exit_codes = {0, 1, 2}
})

-- Or use range
local result = base:run("make check", {
    valid_exit_codes = "0..5"
})
```

### Network Timeout

**Cause:** Network access blocked or slow.

**Solution:**

```lua
-- Use host network if needed
local result = base:run("curl https://api.example.com", {
    network = "host"
})

-- Or use no network for local builds
local result = base:run("mise run test", {
    network = "none"
})
```

---

## Environment Issues

### BuildKit Not Running

**Cause:** BuildKit daemon not started.

**Solution:**

```bash
# Check if running
buildctl debug workers

# Start BuildKit
buildkitd

# Or use systemd
sudo systemctl start buildkitd
```

### Wrong BuildKit Address

**Cause:** Incorrect `BUILDKIT_HOST` setting.

**Solution:**

```bash
# Check address
echo $BUILDKIT_HOST

# Set correct address
export BUILDKIT_HOST=unix:///run/buildkit/buildkitd.sock

# Or use TCP
export BUILDKIT_HOST=tcp://localhost:1234
```

### Permission Denied to Socket

**Cause:** User not in docker group.

**Solution:**

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Log in again
newgrp docker
```

### Stdlib Not Found

**Cause:** Stdlib directory not configured.

**Solution:**

```bash
# Check if exists
ls -la /usr/local/share/luakit/stdlib

# Or set environment variable
export LUAKIT_STDLIB_DIR=/path/to/stdlib

# Or install stdlib
mkdir -p /usr/local/share/luakit/stdlib
cp lua/stdlib/*.lua /usr/local/share/luakit/stdlib/
```

### Memory Issues

**Cause:** Build requires more memory than available.

**Solution:**

```bash
# Increase BuildKit memory limit
buildkitd --oci-worker-gc=high --oci-worker-no-process-sandbox
```

---

## Common Mistakes

### Mistake 1: Using `local` keyword incorrectly

```lua
-- Bad: Trying to use local as variable
local local = bk.image("alpine:3.19")

-- Good: Use bk.local_()
local ctx = bk.local_("context")
```

### Mistake 2: Not using array form for commands

```lua
-- Bad: Shell form may fail
local result = base:run("npm install")

-- Good: Array form more reliable
local result = base:run({"npm", "install"})
```

### Mistake 3: Forgetting to chain operations

```lua
-- Bad: Using original base
local base = bk.image("alpine:3.19")
base:run("apk add git")
local result = base:run("apk add vim")

-- Good: Chain operations
local base = bk.image("alpine:3.19")
local with_git = base:run("apk add git")
local result = with_git:run("apk add vim")
```

### Mistake 4: Using `latest` tag

```lua
-- Bad: Unpredictable
local base = bk.image("alpine:latest")

-- Good: Specific version
local base = bk.image("alpine:3.19.1")
```

### Mistake 5: Hardcoding secrets

```lua
-- Bad: Secret in script
local result = base:run("echo 'password123' > /secret")

-- Good: Use secret mount
local result = base:run("cat /run/secrets/password > /secret", {
    mounts = { bk.secret("/run/secrets/password") }
})
```

---

## Getting Help

### Validate Before Building

Always validate scripts:

```bash
luakit validate build.lua
```

### Check Examples

Look at working examples:

```bash
ls -la examples/
ls -la lua/examples/
```

### Review DAG

Visualize build structure:

```bash
luakit dag build.lua | dot -Tsvg > dag.svg
```

### Check Logs

Review BuildKit logs:

```bash
journalctl -u buildkitd -f
```

### Search Issues

Check existing issues:

```bash
# GitHub
https://github.com/kasuboski/luakit/issues

# Or search
luakit error message
```

---

## Quick Reference

| Error | Solution |
|--------|----------|
| `identifier must not be empty` | Provide valid image ref or context name |
| `command argument required` | Add command to `run()` |
| `requires at least 2 states` | Provide multiple states to `merge()` |
| `already called once` | Call `bk.export()` only once |
| `nothing to build` | Add `bk.export()` call |
| `module not found` | Check module path and install to stdlib |
| `unexpected symbol` | Check Lua syntax |
| `os.execute not available` | Use `bk.run()` instead |

---

## Summary

When troubleshooting:

1. **Validate script** first with `luakit validate`
2. **Visualize DAG** with `luakit dag`
3. **Check error messages** for file and line numbers
4. **Use debug mode** with `buildctl --debug`
5. **Review examples** for working patterns
6. **Search issues** for known problems

Most issues are:
- Missing or incorrect arguments
- Lua syntax errors
- Missing `bk.export()` call
- Improper caching
- Permission issues
