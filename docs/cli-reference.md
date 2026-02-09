# CLI Reference

Complete reference for the `luakit` command-line interface.

## Table of Contents

- [Commands](#commands)
- [Global Flags](#global-flags)
- [build](#build)
- [dag](#dag)
- [validate](#validate)
- [version](#version)
- [Examples](#examples)

## Commands

```bash
luakit build [flags] <script>     Build from a Lua script
luakit dag [flags] <script>       Print the LLB DAG without building
luakit validate <script>          Validate a script without building
luakit version                    Print version information
```

## Global Flags

No global flags exist. Each command has its own flags.

---

## build

Build a container image from a Lua script.

### Usage

```bash
luakit build [flags] <script>
```

### Arguments

- `script` (required): Path to Lua build script

### Flags

#### --output, -o <path>

Write protobuf Definition to file instead of stdout.

**Default:** stdout

**Example:**

```bash
luakit build -o definition.pb build.lua
```

#### --frontend-arg KEY=VALUE

Set a frontend argument (repeatable). Arguments are available to the script via environment.

**Example:**

```bash
luakit build --frontend-arg=VERSION=1.0.0 build.lua
```

In script:

```lua
local version = os.getenv("VERSION") or "0.0.1"
local image = bk.image("myapp:" .. version)
```

#### --help, -h

Show help message for build command.

### Output

The `build` command outputs a BuildKit LLB Definition in protobuf format to stdout (or file specified with `--output`). This can be piped to `buildctl` or saved for later use.

### Exit Codes

- `0`: Success
- `1`: Error (script error, missing script, invalid flags)

### Examples

#### Basic Build

```bash
luakit build build.lua | buildctl build --no-frontend --local context=.
```

#### Build with Docker

```bash
docker buildx build \
  --frontend gateway.v0 \
  --opt source=$(pwd)/build.lua \
  --local context=. \
  -t myapp:latest
```

#### Save Definition to File

```bash
luakit build -o definition.pb build.lua
```

#### Build with Frontend Arguments

```bash
luakit build --frontend-arg=VERSION=1.2.3 --frontend-arg=TARGET=prod build.lua | \
  buildctl build --no-frontend --local context=.
```

---

## dag

Print the LLB DAG (Directed Acyclic Graph) without building. Useful for visualization and debugging.

### Usage

```bash
luakit dag [flags] <script>
```

### Arguments

- `script` (required): Path to Lua build script

### Flags

#### --format <dot|json>

Output format for the DAG.

**Default:** `dot`

**Values:**

- `dot`: Graphviz DOT format (for visualization)
- `json`: JSON representation (for programmatic use)

**Example:**

```bash
luakit dag --format=dot build.lua
luakit dag --format=json build.lua
```

#### --output, -o <path>

Write DAG to file instead of stdout.

**Default:** stdout

**Example:**

```bash
luakit dag -o dag.dot build.lua
luakit dag --format=json -o dag.json build.lua
```

#### --filter <type>

Filter operations by type.

**Values:** `Exec`, `Source`, `File`, `Merge`, `Diff`, `Build`

**Example:**

```bash
# Only show Exec operations
luakit dag --filter=Exec build.lua

# Only show Source operations
luakit dag --filter=Source build.lua
```

#### --help, -h

Show help message for dag command.

### Output Formats

#### DOT Format

Graphviz DOT format for visualization:

```bash
luakit dag build.lua | dot -Tsvg > dag.svg
```

View `dag.svg` in a browser or image viewer.

#### JSON Format

JSON representation of the DAG:

```bash
luakit dag --format=json build.lua | jq .
```

### Use Cases

- **Debugging**: Inspect build structure
- **Optimization**: Identify opportunities for parallelization
- **Documentation**: Generate build graphs for documentation
- **Testing**: Verify expected graph structure

### Exit Codes

- `0`: Success
- `1`: Error (script error, missing script, invalid flags)

---

## validate

Validate a Lua build script without building. Checks syntax and structure.

### Usage

```bash
luakit validate <script>
```

### Arguments

- `script` (required): Path to Lua build script

### Flags

None.

### Validation Checks

1. **Syntax**: Lua syntax is valid
2. **Script execution**: Script runs without errors
3. **API usage**: All `bk.*` functions used correctly
4. **Export**: `bk.export()` called exactly once
5. **State graph**: DAG is well-formed

### Output

- Success: `✓ Script is valid`
- Failure: Error message with location

### Exit Codes

- `0`: Valid
- `1`: Invalid

### Examples

#### Validate Script

```bash
luakit validate build.lua
# Output: ✓ Script is valid
```

#### Validation Error

```bash
luakit validate invalid.lua
# Error: invalid.lua:5: bk.run: command argument required
```

---

## version

Print version information.

### Usage

```bash
luakit version
luakit --version
luakit -v
```

### Output

```
luakit 0.1.0-dev
```

### Exit Codes

- `0`: Success

---

## Examples

### Complete Workflow

```bash
# 1. Validate script
luakit validate build.lua

# 2. Visualize DAG
luakit dag build.lua | dot -Tsvg > dag.svg

# 3. Build image
luakit build build.lua | buildctl build --no-frontend --local context=. -t myapp:latest

# 4. Run image
docker run --rm -p 8080:8080 myapp:latest
```

### Multi-Platform Build

```bash
# Build for ARM64
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/arm64 \
    -t myapp:arm64

# Build for multiple platforms
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/amd64,linux/arm64,linux/arm/v7 \
    -t myapp:multi
```

### With Secrets

```bash
# Create secret file
echo '{"api_key":"secret"}' > secret.json

# Build with secret
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --secret id=secret,src=secret.json \
    -t myapp:latest
```

### With SSH

```bash
# Build with SSH agent
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --ssh id=default \
    -t myapp:latest
```

### DAG Analysis

```bash
# Generate DOT
luakit dag build.lua > dag.dot

# Generate SVG
luakit dag build.lua | dot -Tsvg > dag.svg

# Filter to Exec operations
luakit dag --filter=Exec build.lua | dot -Tsvg > exec-only.svg

# Get JSON for analysis
luakit dag --format=json build.lua > dag.json
jq '.ops | length' dag.json  # Count operations
```

### CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Validate
luakit validate build.lua

# Build
luakit build build.lua > definition.pb

# Test with BuildKit
buildctl build \
  --no-frontend \
  --local context=. \
  --output type=image,name=myapp:test \
  < definition.pb

# If successful, build for production
buildctl build \
  --no-frontend \
  --local context=. \
  --output type=image,name=myapp:latest,push=true \
  --opt tag=$(git describe --tags) \
  < definition.pb
```

### Debugging

```bash
# Step 1: Validate
luakit validate build.lua

# Step 2: Check DAG structure
luakit dag build.lua | dot -Tsvg > dag.svg
# Open dag.svg in browser

# Step 3: Build with verbose output
luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=. \
  --debug

# Step 4: Inspect result
buildctl build \
  --no-frontend \
  --local context=. \
  --output type=local,dest=./output \
  <(luakit build build.lua)
ls -la output/
```

### Advanced Pattern: Module Resolution

```bash
# With custom stdlib directory
LUAKIT_STDLIB_DIR=/custom/stdlib luakit build build.lua

# With context in different directory
cd /path/to/project
luakit build scripts/build.lua
# local_("context") resolves to /path/to/project
```

### Performance Comparison

```bash
# Time script evaluation
time luakit build build.lua > /dev/null
# Output: real 0m0.005s (typical)

# Time serialization
time luakit build build.lua | wc -c
# Output: definition size in bytes

# Time full build
time luakit build build.lua | buildctl build \
  --no-frontend \
  --local context=.
```

---

## Common Patterns

### Pattern 1: Save and Reuse Definition

```bash
# Save definition
luakit build build.lua -o definition.pb

# Reuse definition
buildctl build \
  --no-frontend \
  --local context=. \
  -t myapp:v1 \
  < definition.pb

buildctl build \
  --no-frontend \
  --local context=. \
  -t myapp:v2 \
  < definition.pb
```

### Pattern 2: Conditional Builds

```bash
#!/bin/bash
if [ "$BUILD_TYPE" = "production" ]; then
    luakit build build.lua | buildctl build \
        --no-frontend \
        --local context=. \
        -t myapp:prod
else
    luakit build build.lua | buildctl build \
        --no-frontend \
        --local context=. \
        -t myapp:dev
fi
```

### Pattern 3: Parallel Builds

```bash
# Build multiple platforms in parallel
luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/amd64 \
    -t myapp:amd64 &

luakit build build.lua | \
  buildctl build \
    --no-frontend \
    --local context=. \
    --platform linux/arm64 \
    -t myapp:arm64 &

wait
```

---

## Troubleshooting CLI

### Command Not Found

```bash
# Check installation
which luakit

# Add to PATH
export PATH=$PATH:/path/to/luakit
```

### Permission Denied

```bash
# Make executable
chmod +x luakit

# Or rebuild
go build -o luakit ./cmd/luakit
```

### Script Not Found

```bash
# Use absolute path
luakit build /full/path/to/build.lua

# Or change directory
cd /path/to/project
luakit build build.lua
```

### BuildKit Connection Failed

```bash
# Check BuildKit daemon
buildctl debug workers

# Start BuildKit (if not running)
buildkitd

# Use specific address
export BUILDKIT_HOST=unix:///run/buildkit/buildkitd.sock
```

---

## Exit Codes Reference

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (generic) |
| 2 | Usage error |
| 125 | BuildKit daemon error |
| 126 | Container command not executable |
| 127 | Container command not found |
| 130 | Interrupted (SIGINT) |
| 137 | Container killed (SIGKILL) |

---

## Environment Variables

### LUAKIT_STDLIB_DIR

Path to stdlib directory for `require()`.

**Default:** `{exec_dir}/../share/luakit/stdlib`

**Example:**

```bash
export LUAKIT_STDLIB_DIR=/custom/stdlib
luakit build build.lua
```

### BUILDKIT_HOST

BuildKit daemon address.

**Default:** `unix:///run/buildkit/buildkitd.sock`

**Example:**

```bash
export BUILDKIT_HOST=tcp://localhost:1234
luakit build build.lua | buildctl build --no-frontend --local context=.
```

---

## Related Documentation

- [User Guide](user-guide.md) - Core concepts and patterns
- [API Reference](api-reference.md) - Complete API documentation
- [Integration](integration.md) - BuildKit and CI/CD integration
