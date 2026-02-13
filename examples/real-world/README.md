# Real-World Luakit Examples

This directory contains production-style Dockerfiles and their Luakit equivalents to demonstrate practical viability of Luakit for container image building.

## Examples

### 1. Node.js Web Application
- **Dockerfile**: `nodejs/Dockerfile`
- **Luakit**: `nodejs/build.lua`
- **Features**: Multi-stage build, production optimizations, cache mounts, user management

### 2. Go Microservice
- **Dockerfile**: `go/Dockerfile`
- **Luakit**: `go/build.lua`
- **Features**: Multi-stage build, optimized binary, minimal runtime, Go module caching

### 3. Python Data Science Container
- **Dockerfile**: `python/Dockerfile`
- **Luakit**: `python/build.lua`
- **Features**: Complete ML environment, pip caching, system dependencies, user setup

## Running Examples

Build a specific example:
```bash
luakit build nodejs/build.lua
luakit build go/build.lua
luakit build python/build.lua
```

View the DAG without building:
```bash
luakit dag nodejs/build.lua
luakit dag go/build.lua
luakit dag python/build.lua
```

Validate a script:
```bash
luakit validate nodejs/build.lua
luakit validate go/build.lua
luakit validate python/build.lua
```

## Testing

Run tests for all examples:
```bash
go test ./pkg/luavm -run TestRealWorld -v
```

## Documentation

For detailed information about the porting process, challenges, and best practices, see [PORTING_GUIDE.md](./PORTING_GUIDE.md).

## Key Takeaways

### Advantages of Luakit
- Explicit cache management with named cache mounts
- Programmatic control (conditionals, loops, functions)
- Better multi-stage build clarity with named variables
- Composable operations (merge, diff)
- Testable build scripts

### Porting Challenges
- No direct HEALTHCHECK equivalent (can be added)
- No ARG directive for build-time arguments (can be added)
- Multi-line shell scripts require Lua string concatenation
- Pattern matching in COPY limited to include

### Best Practices
1. Use array form for commands when possible
2. Name intermediate states for clarity
3. Use cache mounts for dependencies
4. Group related operations with fluent interface
5. Export with complete metadata (env, user, workdir, labels)
