package integration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	containerStructureTestTimeout = 10 * time.Minute
)

func skipIfNoBuildctl(t *testing.T) {
	if _, err := exec.LookPath("buildctl"); err != nil {
		t.Skip("buildctl not found in PATH. Install from https://github.com/moby/buildkit")
	}
}

func skipIfNoContainerStructureTest(t *testing.T) {
	if _, err := exec.LookPath("container-structure-test"); err != nil {
		t.Skip("container-structure-test not found in PATH. Install from https://github.com/GoogleContainerTools/container-structure-test")
	}
}

func buildImageWithLuakit(t *testing.T, script, contextDir, imageName string) {
	t.Helper()

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), containerStructureTestTimeout)
	defer cancel()

	args := []string{
		"build",
		"--progress=plain",
		"--frontend", "gateway.v0",
		"--opt", "source=lua-frontend",
		"--opt", "ref=" + imageName,
		"--local", "context=" + contextDir,
		"--output", fmt.Sprintf("type=docker,name=%s", imageName),
	}

	cmd := exec.CommandContext(ctx, "buildctl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("buildctl output: %s", string(output))
		require.NoError(t, err, "buildctl should succeed")
	}
}

func runContainerStructureTest(t *testing.T, imageName, configYAML string) {
	t.Helper()

	skipIfNoContainerStructureTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{
		"test",
		"--image", imageName,
		"--config", "-",
	}

	cmd := exec.CommandContext(ctx, "container-structure-test", args...)
	cmd.Stdin = bytes.NewReader([]byte(configYAML))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("container-structure-test output: %s", string(output))
		require.NoError(t, err, "container-structure-test should succeed")
	}
}

func TestB01_MinimalImageContainsCreatedFile(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b01:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "greeting file exists"
    path: "/greeting.txt"
    shouldExist: true

fileContentTests:
  - name: "greeting file content"
    path: "/greeting.txt"
    expectedContents: ["hello"]
`
	runContainerStructureTest(t, "test-b01:latest", configYAML)
}

func TestB02_MultiStageGoBinaryInDistroless(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local builder = bk.image("golang:1.22-bookworm")
local src = bk.local_("context")
local deps = builder:mkdir("/app"):copy(src, "go.mod", "/app/go.mod")
                                   :copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app" })
local workspace = downloaded:copy(src, ".", "/app")
local built = workspace:run(
    "CGO_ENABLED=0 go build -o /out/server ./cmd/server",
    { cwd = "/app", mounts = { bk.cache("/root/.cache/go-build") } }
)
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
`
	contextDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "go.sum"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(contextDir, "cmd", "server"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "cmd", "server", "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"server running\")\n}\n"), 0644))

	buildImageWithLuakit(t, script, contextDir, "test-b02:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "server binary exists"
    path: "/server"
    shouldExist: true
    permissions: "-rwxr-xr-x"

fileExistenceTests:
  - name: "no Go toolchain in runtime"
    path: "/usr/local/go/bin/go"
    shouldExist: false

commandTests:
  - name: "server binary is executable"
    command: "/server"
    args: ["--version"]
    exitCode: 0

metadataTest:
  entrypoint: ["/server"]
`
	runContainerStructureTest(t, "test-b02:latest", configYAML)
}

func TestB03_MkfileCorrectContentAndPermissions(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:mkfile("/etc/app.conf", "listen=8080\nworkers=4", { mode = 0644 })
bk.export(s)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b03:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "config file exists"
    path: "/etc/app.conf"
    shouldExist: true
    permissions: "-rw-r--r--"

fileContentTests:
  - name: "config file content"
    path: "/etc/app.conf"
    expectedContents: ["listen=8080", "workers=4"]
`
	runContainerStructureTest(t, "test-b03:latest", configYAML)
}

func TestB04_MkdirCreatesDirectoryTree(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:mkdir("/app/data/logs", { mode = 0755, make_parents = true })
bk.export(s)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b04:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "nested directory exists"
    path: "/app/data/logs"
    shouldExist: true
    permissions: "drwxr-xr-x"
  - name: "parent directory exists"
    path: "/app/data"
    shouldExist: true
`
	runContainerStructureTest(t, "test-b04:latest", configYAML)
}

func TestB05_RmRemovesFile(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:mkfile("/tmp/deleteme", "gone"):rm("/tmp/deleteme")
bk.export(s)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b05:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "deleted file is gone"
    path: "/tmp/deleteme"
    shouldExist: false
`
	runContainerStructureTest(t, "test-b05:latest", configYAML)
}

func TestB06_SymlinkIsCreated(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:run("echo '#!/bin/sh\necho hello' > /usr/local/bin/greet && chmod +x /usr/local/bin/greet")
local linked = s:symlink("/usr/local/bin/greet", "/usr/local/bin/hi")
bk.export(linked)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b06:latest")

	configYAML := `
schemaVersion: "2.0.0"
commandTests:
  - name: "symlink works"
    command: "/usr/local/bin/hi"
    expectedOutput: ["hello"]
    exitCode: 0
`
	runContainerStructureTest(t, "test-b06:latest", configYAML)
}

func TestB07_CopyWithModeApplied(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local builder = bk.image("alpine:3.19")
local built = builder:run("echo '#!/bin/sh\necho running' > /out/app && chmod +x /out/app")
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/usr/local/bin/app", { mode = "0755" })
bk.export(final)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b07:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "copied binary exists with correct mode"
    path: "/usr/local/bin/app"
    shouldExist: true
    permissions: "-rwxr-xr-x"

commandTests:
  - name: "copied binary runs"
    command: "/usr/local/bin/app"
    expectedOutput: ["running"]
    exitCode: 0
`
	runContainerStructureTest(t, "test-b07:latest", configYAML)
}

func TestB08_ImageConfigEntrypointCmdEnvUserWorkdirExpose(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:run("adduser -D app"):mkdir("/app")
bk.export(s, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "echo configured"},
    env = {"APP_ENV=production", "PATH=/usr/local/bin:/usr/bin:/bin"},
    expose = {"8080/tcp", "9090/tcp"},
    user = "app",
    workdir = "/app",
    labels = {
        ["org.opencontainers.image.source"] = "https://github.com/example/app",
    },
})
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b08:latest")

	configYAML := `
schemaVersion: "2.0.0"
metadataTest:
  entrypoint: ["/bin/sh"]
  cmd: ["-c", "echo configured"]
  workdir: "/app"
  user: "app"
  exposedPorts: ["8080/tcp", "9090/tcp"]
  env:
    - key: "APP_ENV"
      value: "production"
  labels:
    - key: "org.opencontainers.image.source"
      value: "https://github.com/example/app"

commandTests:
  - name: "default command output"
    command: "/bin/sh"
    args: ["-c", "echo configured"]
    expectedOutput: ["configured"]
    exitCode: 0
`
	runContainerStructureTest(t, "test-b08:latest", configYAML)
}

func TestB09_DiffMergeAppliesDeltaCorrectly(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache curl")

local target = bk.image("alpine:3.19")
local final = bk.merge(target, installed)
bk.export(final)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b09:latest")

	configYAML := `
schemaVersion: "2.0.0"
commandTests:
  - name: "curl is available from diff delta"
    command: "curl"
    args: ["--version"]
    exitCode: 0
`
	runContainerStructureTest(t, "test-b09:latest", configYAML)
}

func TestB10_LocalContextFilesIncludedInBuild(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local ctx = bk.local_("context")
local final = base:copy(ctx, "hello.txt", "/data/hello.txt")
bk.export(final)
`
	contextDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "hello.txt"), []byte("world"), 0644))

	buildImageWithLuakit(t, script, contextDir, "test-b10:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "context file copied"
    path: "/data/hello.txt"
    shouldExist: true

fileContentTests:
  - name: "context file content"
    path: "/data/hello.txt"
    expectedContents: ["world"]
`
	runContainerStructureTest(t, "test-b10:latest", configYAML)
}

func TestB11_GitSourceContentAvailable(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local repo = bk.git("https://github.com/moby/buildkit.git", { ref = "v0.12.0" })
local base = bk.image("alpine:3.19")
local final = base:copy(repo, "README.md", "/README.md")
bk.export(final)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b11:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "README from git source exists"
    path: "/README.md"
    shouldExist: true

fileContentTests:
  - name: "README has buildkit content"
    path: "/README.md"
    expectedContents: ["BuildKit"]
`
	runContainerStructureTest(t, "test-b11:latest", configYAML)
}

func TestB12_SecretMountAvailableDuringBuildAbsentFromImage(t *testing.T) {
	skipIfNoBuildctl(t)

	secretPath := filepath.Join(t.TempDir(), "token.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte("TOKEN_CONTENT"), 0400))

	script := `
local base = bk.image("alpine:3.19")
local s = base:run("cat /run/secrets/mytoken > /proof.txt", {
    mounts = { bk.secret("/run/secrets/mytoken", { id = "mytoken" }) },
})
bk.export(s)
`
	contextDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, contextDir)
	require.NoError(t, err, "luakit build should succeed")

	args := []string{
		"build",
		"--progress=plain",
		"--no-frontend",
		"--local", "context=" + contextDir,
		"--secret", fmt.Sprintf("id=mytoken,src=%s", secretPath),
		"--output", "type=docker,name=test-b12:latest",
	}

	cmd := exec.CommandContext(ctx, "buildctl", args...)
	cmd.Stdin = bytes.NewReader(def)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("buildctl output: %s", string(output))
		require.NoError(t, err, "buildctl should succeed")
	}

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "proof file from secret exists"
    path: "/proof.txt"
    shouldExist: true
  - name: "secret is not baked into image"
    path: "/run/secrets/mytoken"
    shouldExist: false

fileContentTests:
  - name: "proof file has secret value"
    path: "/proof.txt"
    expectedContents: ["TOKEN_CONTENT"]
`
	runContainerStructureTest(t, "test-b12:latest", configYAML)
}

func TestB13_NetworkModeNoneBlocksAccess(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:run("apk add --no-cache curl || echo 'network blocked' > /result.txt", {
    network = "none",
})
bk.export(s)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b13:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileContentTests:
  - name: "network was blocked"
    path: "/result.txt"
    expectedContents: ["network blocked"]
`
	runContainerStructureTest(t, "test-b13:latest", configYAML)
}

func TestB14_ValidExitCodesAcceptNonZero(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local s = base:run("sh -c 'exit 1'", { valid_exit_codes = {0, 1} })
local final = s:mkfile("/ok.txt", "passed")
bk.export(final)
`
	contextDir := t.TempDir()

	buildImageWithLuakit(t, script, contextDir, "test-b14:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "build continued after exit code 1"
    path: "/ok.txt"
    shouldExist: true
`
	runContainerStructureTest(t, "test-b14:latest", configYAML)
}

func TestB15_RealWorldNodeJsApp(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("node:20-slim")
local src = bk.local_("context")
local workspace = base:mkdir("/app"):copy(src, "package.json", "/app/package.json")
                                     :copy(src, "package-lock.json", "/app/package-lock.json")
local deps = workspace:run("npm ci --production", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") },
})
local full = deps:copy(src, ".", "/app")
local built = full:run("npm run build", { cwd = "/app" })

local runtime = bk.image("nginx:alpine")
local final = runtime:copy(built, "/app/dist", "/usr/share/nginx/html")
bk.export(final, { expose = {"80/tcp"} })
`
	contextDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "package.json"), []byte(`{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": { "build": "mkdir -p dist && echo '<html>Hello</html>' > dist/index.html" }
}
`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "package-lock.json"), []byte(`{
  "name": "test-app",
  "version": "1.0.0",
  "lockfileVersion": 3
}
`), 0644))

	buildImageWithLuakit(t, script, contextDir, "test-b15:latest")

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "built assets exist"
    path: "/usr/share/nginx/html/index.html"
    shouldExist: true

metadataTest:
  exposedPorts: ["80/tcp"]

commandTests:
  - name: "nginx config is valid"
    command: "nginx"
    args: ["-t"]
    exitCode: 0
`
	runContainerStructureTest(t, "test-b15:latest", configYAML)
}

func TestB16_FrontendGatewayImageMode(t *testing.T) {
	skipIfNoBuildctl(t)

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
`
	contextDir := t.TempDir()

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{
		"build",
		"--frontend=gateway.v0",
		"--opt", "source=luakit:test",
		"--local", "context=" + contextDir,
		"--output", "type=docker,name=test-frontend",
	}

	cmd := exec.CommandContext(ctx, "buildctl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("buildctl output: %s", string(output))
		require.NoError(t, err, "buildctl should succeed through gateway protocol")
	}

	configYAML := `
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "frontend-built image has expected file"
    path: "/greeting.txt"
    shouldExist: true
`
	runContainerStructureTest(t, "test-frontend:latest", configYAML)
}
