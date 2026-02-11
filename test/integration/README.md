# Integration Tests

This directory contains integration tests for luakit that verify the build system generates correct BuildKit definitions and produces valid container images.

## Prerequisites

To run these tests, you need:

1. **luakit binary** built first:
   ```bash
   go build -o dist/luakit ./cmd/luakit
   ```

2. **buildctl** CLI (for TestB* tests that build actual images):
   ```bash
   # Install on macOS
   brew install buildkit

   # Install on Linux
   # See https://github.com/moby/buildkit#installing
   ```

3. **BuildKit daemon** (for TestB* tests only):
   ```bash
   # Using Docker (recommended)
   docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234
   export BUILDKIT_HOST=tcp://127.0.0.1:1234

   # Or directly
   buildkitd &
   ```

4. **container-structure-test** CLI (for TestB* tests only):
   ```bash
   # Install on macOS
   brew install container-structure-test

   # Install on Linux
   # See https://github.com/GoogleContainerTools/container-structure-test#installation
   ```

Note: TestA* tests don't require BuildKit or external tools - they only generate and validate protobuf definitions.

## Running Tests

### Run all integration tests:
```bash
go test ./test/integration/ -v
```

### Run definition validation tests (TestA* - no BuildKit required):
```bash
go test ./test/integration/ -v -run TestA
```

### Run container structure tests (TestB* - requires BuildKit):
```bash
go test ./test/integration/ -v -run TestB
```

### Run specific test:
```bash
go test ./test/integration/ -v -run TestA01
```

### Run with timeout:
```bash
go test ./test/integration/ -v -timeout 10m
```

### Run with race detection:
```bash
go test ./test/integration/ -race
```

## Test Organization

### TestA Series: Definition Validation

These tests verify that luakit generates correct BuildKit protobuf definitions. They don't require BuildKit, buildctl, or Docker to run.

**File**: `definition_validation_test.go` (31 tests)

- **TestA01-TestA04**: Source operations (image, scratch, local)
- **TestA05-TestA07**: Exec operations with various options
- **TestA08-TestA11**: DAG validation and determinism
- **TestA12-TestA15**: File operation serialization
- **TestA16-TestA20**: Mount type serialization
- **TestA21-TestA22**: Graph operations (merge, diff)
- **TestA23-TestA24**: Advanced source operations (git, http)
- **TestA25-TestA26**: Network and security modes
- **TestA27**: Platform serialization
- **TestA28-TestA29**: Exec metadata (env, user, cwd, hostname)
- **TestA30**: File mode validation
- **TestA31**: Source mapping validation

### TestA Series: Golden File Validation

These tests compare generated protobuf definitions against golden files to ensure consistency across changes.

**File**: `golden_test.go` (10 tests)

- **TestA32-TestA41**: Validates output matches expected golden files for various build scenarios

### TestB Series: Container Structure Tests

These tests build actual container images using BuildKit and validate their structure using container-structure-test. They require buildctl, BuildKit daemon, and container-structure-test.

**File**: `container_structure_test.go` (16 tests)

- **TestB01**: Minimal image with created file
- **TestB02**: Multi-stage Go binary in distroless
- **TestB03**: Mkfile correct content and permissions
- **TestB04**: Mkdir creates directory tree
- **TestB05**: Rm removes file
- **TestB06**: Symlink is created
- **TestB07**: Copy with mode applied
- **TestB08**: Image config (entrypoint, cmd, env, user, workdir, expose)
- **TestB09**: Diff/merge applies delta correctly
- **TestB10**: Local context files included in build
- **TestB11**: Git source content available
- **TestB12**: Secret mount available during build, absent from image
- **TestB13**: Network mode none blocks access
- **TestB14**: Valid exit codes accept non-zero
- **TestB15**: Real-world Node.js app
- **TestB16**: Frontend gateway image mode

## CI/CD Integration

These tests are designed to run in CI/CD pipelines.

### GitHub Actions Example:

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build luakit binary
        run: go build -o dist/luakit ./cmd/luakit

      - name: Run definition validation tests
        run: go test ./test/integration/ -v -run TestA

      - name: Install buildkitd
        run: |
          docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234
          export BUILDKIT_HOST=tcp://127.0.0.1:1234

      - name: Install buildctl and container-structure-test
        run: |
          wget https://github.com/moby/buildkit/releases/download/v0.12.1/buildkit-v0.12.1.linux-amd64.tar.gz
          tar -xzvf buildkit-v0.12.1.linux-amd64.tar.gz
          sudo mv bin/buildctl /usr/local/bin/
          wget https://github.com/GoogleContainerTools/container-structure-test/releases/download/v1.16.0/container-structure-test-linux-amd64
          chmod +x container-structure-test-linux-amd64
          sudo mv container-structure-test-linux-amd64 /usr/local/bin/container-structure-test

      - name: Run container structure tests
        run: |
          export BUILDKIT_HOST=tcp://127.0.0.1:1234
          go test ./test/integration/ -v -run TestB -timeout 15m
        env:
          BUILDKIT_HOST: tcp://127.0.0.1:1234
```

## Troubleshooting

### luakit binary not found
```
Error: luakit binary not found at ../../dist/luakit
```
**Solution**: Build the binary first:
```bash
go build -o dist/luakit ./cmd/luakit
```

### buildctl not found
```
Error: buildctl not found in PATH
```
**Solution**: Install buildctl from https://github.com/moby/buildkit#installing (only required for TestB* tests)

### BuildKit daemon not running
```
Error: BuildKit daemon not running at /run/buildkit/buildkitd.sock
```
**Solution**: Start BuildKit daemon (only required for TestB* tests):
```bash
docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234
export BUILDKIT_HOST=tcp://127.0.0.1:1234
```

### container-structure-test not found
```
Error: container-structure-test not found in PATH
```
**Solution**: Install from https://github.com/GoogleContainerTools/container-structure-test#installation (only required for TestB* tests)

### Tests timing out
```
Error: context deadline exceeded
```
**Solution**: Increase timeout:
```bash
go test ./test/integration/ -timeout 20m
```

## Adding New Tests

To add a new integration test:

1. Build the luakit binary first:
   ```bash
   go build -o dist/luakit ./cmd/luakit
   ```

2. Decide which test series to add to:
   - **TestA** (definition validation): If you just need to verify the generated protobuf definition
   - **TestB** (container structure): If you need to build an actual image and validate its contents

3. For definition validation tests (TestA*), create a test in `definition_validation_test.go`:
   ```go
   func TestA32_MyNewFeature(t *testing.T) {
       script := `-- your lua script --`
       scriptPath := createTestScript(t, script)
       def, err := runLuakitBuild(t, scriptPath, ".")
       require.NoError(t, err, "luakit build should succeed")
       pbDef := requireValidDefinition(t, def)

       // Add validation logic using helpers like:
       // - requireExecOpCount(t, pbDef, expected)
       // - requireSourceOpCount(t, pbDef, expected)
       // - requireFileOpCount(t, pbDef, expected)
       // - requireMountOfType(t, pbDef, mountType, dest)
   }
   ```

4. For container structure tests (TestB*), create a test in `container_structure_test.go`:
   ```go
   func TestB17_MyNewFeature(t *testing.T) {
       skipIfNoBuildctl(t)

       script := `-- your lua script --`
       config := &structuretest.Config{
           // Define structure tests here
       }
       runContainerStructureTest(t, script, config)
   }
   ```

5. Run the test to verify it works:
   ```bash
   go test ./test/integration/ -v -run TestA32
   # or
   go test ./test/integration/ -v -run TestB17
   ```

## Test Helpers

The `helpers_test.go` file provides useful test helpers:

- `runLuakitBuild(t, scriptPath, contextDir)` - Run luakit build and return protobuf definition
- `requireValidDefinition(t, def)` - Unmarshal and validate a pb.Definition
- `requireSourceMapping(t, pbDef)` - Validate source mapping is present
- `requireDeterministic(t, script)` - Verify output is deterministic
- `requireExecOpCount(t, pbDef, count)` - Count exec operations
- `requireSourceOpCount(t, pbDef, count)` - Count source operations
- `requireFileOpCount(t, pbDef, count)` - Count file operations
- `requireMergeOp(t, pbDef, inputCount)` - Validate merge operation
- `requireDiffOp(t, pbDef)` - Validate diff operation
- `requireMountOfType(t, pbDef, mountType, dest)` - Find specific mount
- `requireExecMeta(t, pbDef, cwd, user, env)` - Validate exec metadata
- `requireNetworkMode(t, pbDef, mode)` - Validate network mode
- `requireSecurityMode(t, pbDef, mode)` - Validate security mode
- `requireSourceIdentifier(t, pbDef, identifier)` - Validate source op identifier
- `createTestScript(t, script)` - Create temporary test script

## Contributing

When adding new features to luakit, please add corresponding integration tests:

1. Add definition validation tests (TestA*) for all new operations
2. Add container structure tests (TestB*) for features that affect the final image
3. Update golden files (TestA32-TestA41) when output format changes
4. Ensure tests are parallelizable using `t.Parallel()`
5. All tests should be independent and not interfere with each other
