# BuildKit Integration Tests

This directory contains end-to-end integration tests for luakit that verify it works correctly with BuildKit in real scenarios.

## Prerequisites

To run these tests, you need:

1. **BuildKit daemon** running:
   ```bash
   # Using Docker (recommended)
   docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234
   export BUILDKIT_HOST=tcp://127.0.0.1:1234
   
   # Or directly
   buildkitd &
   ```

2. **buildctl** CLI (for some tests):
   ```bash
   # Install on macOS
   brew install buildkit
   
   # Install on Linux
   # See https://github.com/moby/buildkit#installing
   ```

3. **Docker** (for some tests):
   ```bash
   # Ensure docker is installed and running
   docker info
   ```

## Running Tests

### Run all integration tests:
```bash
go test ./test/integration/... -tags=e2e -v
```

### Run specific test:
```bash
go test ./test/integration/... -tags=e2e -v -run TestBuildSimpleImage
```

### Run with timeout:
```bash
go test ./test/integration/... -tags=e2e -v -timeout 10m
```

### Run with race detection:
```bash
go test ./test/integration/... -tags=e2e -race
```

## Test Coverage

The integration tests cover:

### Basic Operations
- **TestBuildSimpleImage**: Tests basic image build with exec operations
- **TestBuildMultiStage**: Tests multi-stage builds with multiple FROM operations

### Mount Types
- **TestBuildWithCacheMount**: Tests cache mount serialization and execution
- **TestBuildWithSecretMount**: Tests secret mount integration
- **TestBuildWithSSMount**: Tests SSH mount integration
- **TestBuildWithTmpfsMount**: Tests tmpfs mount integration
- **TestBuildWithBindMount**: Tests bind mount between states

### Cross-Platform Builds
- **TestBuildCrossPlatform**: Tests building for multiple platforms (linux/amd64, linux/arm64, linux/arm/v7)

### Graph Operations
- **TestBuildMerge**: Tests merge operation combining multiple branches
- **TestBuildDiff**: Tests diff operation for extracting filesystem changes
- **TestBuildParallelExecution**: Tests parallel execution of independent branches

### Source Operations
- **TestBuildWithGitSource**: Tests git source operation
- **TestBuildWithHTTPSource**: Tests HTTP source operation
- **TestBuildWithLocalContext**: Tests local context operations

### Image Configuration
- **TestBuildWithImageConfig**: Tests full image config (entrypoint, cmd, env, user, workdir, labels)

### Exec Options
- **TestBuildWithNetworkModes**: Tests different network modes (sandbox, host, none)
- **TestBuildWithSecurityModes**: Tests different security modes (sandbox, insecure)
- **TestBuildWithValidExitCodes**: Tests custom valid exit codes
- **TestBuildWithHostname**: Tests custom hostname

### File Operations
- **TestBuildWithFileOperations**: Tests all file operations (mkdir, mkfile, copy, symlink, rm)
- **TestBuildWithOwnerPermissions**: Tests file owner/group permissions
- **TestBuildWithIncludeExcludePatterns**: Tests include/exclude patterns for file copying

### Real-World Examples
- **TestBuildRealWorldGoApp**: Tests a real Go application build
- **TestBuildRealWorldNodeApp**: Tests a real Node.js application build
- **TestBuildRealWorldPythonApp**: Tests a real Python application build

### Advanced Features
- **TestBuildWithRequire**: Tests Lua module require functionality
- **TestBuildWithLargeDAG**: Tests building large DAGs (50+ operations)
- **TestBuildWithNestedMerge**: Tests nested merge operations
- **TestBuildWithComplexDiffMerge**: Tests complex diff+merge patterns

### Error Handling
- **TestBuildErrorHandling**: Tests error cases (no export, empty image ref, invalid mount type)

### Definition Validation
- **TestDefinitionValidation**: Validates pb.Definition structure and content
- **TestSourceMapGeneration**: Validates source map generation for debugging
- **TestImageConfigSerialization**: Validates image config serialization
- **TestDigestComputation**: Tests digest determinism
- **TestMountSerialization**: Validates mount serialization
- **TestPlatformSerialization**: Validates platform specification
- **TestExecOptionsSerialization**: Validates exec options serialization
- **TestFileActionSerialization**: Validates file action serialization

### Performance and Reliability
- **TestBuildctlIntegration**: Tests actual buildctl integration
- **TestConcurrency**: Tests concurrent builds
- **TestLuaVMIsolation**: Tests Lua VM isolation between builds
- **TestLargeScript**: Tests performance with large scripts (100+ ops)
- **TestMemoryUsage**: Tests memory usage over multiple builds

## CI/CD Integration

These tests are designed to run in CI/CD pipelines. The recommended setup:

### GitHub Actions Example:

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      buildkitd:
        image: moby/buildkit:latest
        ports:
          - 127.0.0.1:1234:1234
        options: --privileged
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install buildctl
        run: |
          wget https://github.com/moby/buildkit/releases/download/v0.12.1/buildkit-v0.12.1.linux-amd64.tar.gz
          tar -xzvf buildkit-v0.12.1.linux-amd64.tar.gz
          sudo mv bin/buildctl /usr/local/bin/
      
      - name: Run integration tests
        run: |
          export BUILDKIT_HOST=tcp://127.0.0.1:1234
          go test ./test/integration/... -tags=e2e -v -timeout 15m
```

### GitLab CI Example:

```yaml
integration-tests:
  image: golang:1.21
  services:
    - name: moby/buildkit:latest
      alias: buildkitd
      command: ["--addr", "tcp://0.0.0.0:1234"]
  
  variables:
    BUILDKIT_HOST: tcp://buildkitd:1234
  
  before_script:
    - wget https://github.com/moby/buildkit/releases/download/v0.12.1/buildkit-v0.12.1.linux-amd64.tar.gz
    - tar -xzvf buildkit-v0.12.1.linux-amd64.tar.gz
    - mv bin/buildctl /usr/local/bin/
  
  script:
    - go test ./test/integration/... -tags=e2e -v -timeout 15m
```

## Troubleshooting

### BuildKit daemon not running
```
Error: BuildKit daemon not running at /run/buildkit/buildkitd.sock
```
**Solution**: Start BuildKit daemon:
```bash
docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234
export BUILDKIT_HOST=tcp://127.0.0.1:1234
```

### buildctl not found
```
Error: buildctl not found in PATH
```
**Solution**: Install buildctl from https://github.com/moby/buildkit#installing

### Tests timing out
```
Error: context deadline exceeded
```
**Solution**: Increase timeout:
```bash
go test ./test/integration/... -tags=e2e -timeout 20m
```

### Docker not available
```
Error: Docker not found in PATH
```
**Solution**: Install Docker or skip tests that require it:
```bash
go test ./test/integration/... -tags=e2e -skip "TestBuild.*"
```

## Adding New Tests

To add a new integration test:

1. Create a new test function in either `e2e_test.go` or `definition_validation_test.go`
2. Follow the pattern:
   ```go
   func TestMyNewFeature(t *testing.T) {
       skipIfNoBuildKit(t)
       skipIfNoDocker(t) // if needed
       
       t.Parallel()
       
       script := `-- your lua script --`
       scriptPath := filepath.Join(t.TempDir(), "build.lua")
       require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))
       
       def, err := runLuakitBuild(t, scriptPath, ".")
       require.NoError(t, err, "luakit build should succeed")
       
       // Add validation logic here
   }
   ```

3. Run the test to verify it works:
   ```bash
   go test ./test/integration/... -tags=e2e -v -run TestMyNewFeature
   ```

## Contributing

When adding new features to luakit, please add corresponding integration tests:

1. Basic functionality tests
2. Edge case tests
3. Error handling tests
4. Performance tests (if applicable)
5. Cross-platform tests (if applicable)

All tests should be parallelizable and should not interfere with each other.
