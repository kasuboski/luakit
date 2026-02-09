//go:build e2e
// +build e2e

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	buildkitdAddr = "unix:///run/buildkit/buildkitd.sock"
	luakitBinary  = "./luakit"
	testTimeout   = 5 * time.Minute
	buildTimeout  = 3 * time.Minute
)

func skipIfNoBuildKit(t *testing.T) {
	if _, err := os.Stat("/run/buildkit/buildkitd.sock"); os.IsNotExist(err) {
		t.Skip("BuildKit daemon not running at /run/buildkit/buildkitd.sock. Run: buildkitd &")
	}
}

func skipIfNoDocker(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found in PATH")
	}
}

func runLuakitBuild(t *testing.T, scriptPath string, contextDir string, extraArgs ...string) ([]byte, error) {
	args := []string{"build", "-o", "/dev/stdout", scriptPath}
	args = append(args, extraArgs...)

	cmd := exec.Command(luakitBinary, args...)
	cmd.Dir = contextDir
	cmd.Env = append(os.Environ(), "LUAKIT_STDLIB_DIR=")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("luakit build failed: %w\nOutput: %s", err, string(output))
	}

	return output, nil
}

func runBuildctl(t *testing.T, def []byte, contextDir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	args := []string{
		"build",
		"--progress=plain",
		"--no-frontend",
		"--local", "context=" + contextDir,
	}

	cmd := exec.CommandContext(ctx, "buildctl", args...)
	cmd.Stdin = bytes.NewReader(def)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("buildctl failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func TestBuildSimpleImage(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/run_example.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildMultiStage(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/patterns_example.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithCacheMount(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/patterns_example.lua", ".")
	require.NoError(t, err, "luakit build should succeed")

	t.Logf("Definition size: %d bytes", len(def))
	require.Contains(t, string(def), "cache", "definition should contain cache mount reference")
}

func TestBuildWithSecretMount(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	secretFile := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(secretFile, []byte("my-secret-value"), 0600))

	def, err := runLuakitBuild(t, "../../examples/patterns_example.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
}

func TestBuildWithSSMount(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/patterns_example.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
}

func TestBuildWithTmpfsMount(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("sh -c 'echo test > /tmp/test.txt'", {
    mounts = { bk.tmpfs("/tmp", { size = 67108864 }) },
})
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithBindMount(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local data = bk.scratch()
local data_with_file = data:mkfile("/data.txt", "test content")
local result = base:run("cat /mount/data.txt", {
    mounts = { bk.bind(data_with_file, "/mount", { readonly = true }) },
})
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildCrossPlatform(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	platforms := []struct {
		name     string
		platform string
	}{
		{"linux/amd64", "linux/amd64"},
		{"linux/arm64", "linux/arm64"},
		{"linux/arm/v7", "linux/arm/v7"},
	}

	for _, tt := range platforms {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
local base = bk.image("alpine:3.19", { platform = "%s" })
local result = base:run("sh -c 'uname -m && echo hello > /hello.txt'")
bk.export(result)
`, tt.platform)

			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed for platform %s", tt.platform)
			require.NotEmpty(t, def, "definition should not be empty for platform %s", tt.platform)
		})
	}
}

func TestBuildMerge(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local branch1 = base:run("echo 'branch1' > /branch1.txt")
local branch2 = base:run("echo 'branch2' > /branch2.txt")
local branch3 = base:run("echo 'branch3' > /branch3.txt")
local merged = bk.merge(branch1, branch2, branch3)
bk.export(merged)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildDiff(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache curl")
local delta = bk.diff(base, installed)
local result = base:copy(delta, "/", "/delta")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithGitSource(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local repo = bk.git("https://github.com/alpinelinux/docker-alpine.git", { ref = "v3.19" })
local result = repo:run("ls -la")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithHTTPSource(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local file = bk.http("https://raw.githubusercontent.com/alpinelinux/docker-alpine/master/versions/x86_64/alpine-3.19.0-x86_64.iso.sha256", {
    checksum = "sha256:abc123",
    filename = "checksum.txt",
    chmod = 0644,
})
bk.export(file)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithLocalContext(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	contextDir := t.TempDir()

	testFile := filepath.Join(contextDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content from local context"), 0644))

	script := `
local ctx = bk.local_("context")
local base = bk.image("alpine:3.19")
local result = base:copy(ctx, "test.txt", "/test.txt")
bk.export(result)
`

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, contextDir)
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithImageConfig(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "echo running"},
    env = {"TEST_VAR=test", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
    user = "nobody",
    workdir = "/app",
    labels = {
        ["org.opencontainers.image.title"] = "Test Image",
        ["org.opencontainers.image.description"] = "Test image for integration tests",
    },
})
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")

	defStr := string(def)
	require.Contains(t, defStr, "entrypoint", "definition should contain entrypoint")
	require.Contains(t, defStr, "TEST_VAR=test", "definition should contain environment variable")
	require.Contains(t, defStr, "nobody", "definition should contain user")
	require.Contains(t, defStr, "/app", "definition should contain workdir")
}

func TestBuildWithNetworkModes(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	networkModes := []struct {
		name string
		mode string
	}{
		{"sandbox", "sandbox"},
		{"host", "host"},
		{"none", "none"},
	}

	for _, tt := range networkModes {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
local base = bk.image("alpine:3.19")
local result = base:run("echo test", { network = "%s" })
bk.export(result)
`, tt.mode)

			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed for network mode %s", tt.mode)
			require.NotEmpty(t, def, "definition should not be empty")
		})
	}
}

func TestBuildWithSecurityModes(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	securityModes := []struct {
		name string
		mode string
	}{
		{"sandbox", "sandbox"},
		{"insecure", "insecure"},
	}

	for _, tt := range securityModes {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
local base = bk.image("alpine:3.19")
local result = base:run("echo test", { security = "%s" })
bk.export(result)
`, tt.mode)

			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed for security mode %s", tt.mode)
			require.NotEmpty(t, def, "definition should not be empty")
		})
	}
}

func TestBuildWithValidExitCodes(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("sh -c 'exit 0'", { valid_exit_codes = {0} })
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithHostname(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("hostname", { hostname = "test-container" })
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithFileOperations(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local with_dir = base:mkdir("/app", { mode = 0755, make_parents = true })
local with_file = with_dir:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local with_copy = with_file:mkfile("/app/src.txt", "source content")
local with_result = with_copy:copy(with_file, "/app/config.json", "/app/config-backup.json")
local with_symlink = with_result:symlink("/app/config.json", "/app/config-link.json")
local without_file = with_symlink:rm("/app/src.txt", { allow_not_found = true })
bk.export(without_file)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildRealWorldGoApp(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/real-world/go/build.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildRealWorldNodeApp(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/real-world/nodejs/build.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildRealWorldPythonApp(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	def, err := runLuakitBuild(t, "../../examples/real-world/python/build.lua", ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildParallelExecution(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local task1 = base:run("sleep 1 && echo task1 > /task1.txt")
local task2 = base:run("sleep 1 && echo task2 > /task2.txt")
local task3 = base:run("sleep 1 && echo task3 > /task3.txt")
local merged = bk.merge(task1, task2, task3)
bk.export(merged)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestDefinitionDeterminism(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /hello.txt")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def1, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "first build should succeed")

	def2, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "second build should succeed")

	require.Equal(t, def1, def2, "definition should be deterministic")
}

func TestBuildWithOwnerPermissions(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local with_user = base:run("adduser -D -u 1000 testuser")
local result = with_user:mkdir("/app", {
    owner = { user = 1000, group = 1000 },
    mode = 0755,
})
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithIncludeExcludePatterns(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	contextDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "include.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "include_test.go"), []byte("package main"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "exclude.md"), []byte("# README"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(contextDir, "vendor/bad.go"), []byte("package vendor"), 0644))

	script := `
local ctx = bk.local_("context", {
    include = {"*.go"},
    exclude = {"*_test.go", "vendor/"},
})
local base = bk.image("alpine:3.19")
local result = base:copy(ctx, ".", "/src")
bk.export(result)
`

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, contextDir)
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildErrorHandling(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	tests := []struct {
		name        string
		script      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "no export",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
`,
			expectError: true,
			errorMsg:    "no bk.export() call",
		},
		{
			name: "empty image ref",
			script: `
local base = bk.image("")
bk.export(base)
`,
			expectError: true,
		},
		{
			name: "invalid mount type",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo test", { mounts = { "not a mount" } })
bk.export(result)
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(tt.script), 0644))

			_, err := runLuakitBuild(t, scriptPath, ".")
			if tt.expectError {
				require.Error(t, err, "expected build to fail")
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg, "error message should contain expected text")
				}
			} else {
				require.NoError(t, err, "expected build to succeed")
			}
		})
	}
}

func TestBuildWithRequire(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	contextDir := t.TempDir()

	moduleDir := filepath.Join(contextDir, "modules")
	require.NoError(t, os.MkdirAll(moduleDir, 0755))

	moduleScript := `
local M = {}
function M.hello(state)
    return state:run("echo 'hello from module'")
end
return M
`
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "helpers.lua"), []byte(moduleScript), 0644))

	mainScript := `
local helpers = require("modules/helpers")
local base = bk.image("alpine:3.19")
local result = helpers.hello(base)
bk.export(result)
`

	scriptPath := filepath.Join(contextDir, "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(mainScript), 0644))

	def, err := runLuakitBuild(t, scriptPath, contextDir)
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithLargeDAG(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("local base = bk.image(\"alpine:3.19\")\n")

	for i := 0; i < 50; i++ {
		scriptBuilder.WriteString(fmt.Sprintf("local step%d = base:run(\"echo step%d\")\n", i, i))
	}

	scriptBuilder.WriteString("bk.export(step49)\n")

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptBuilder.String()), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithNestedMerge(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local a = base:run("echo a > /a.txt")
local b = base:run("echo b > /b.txt")
local c = base:run("echo c > /c.txt")
local d = base:run("echo d > /d.txt")
local merged1 = bk.merge(a, b)
local merged2 = bk.merge(c, d)
local final = bk.merge(merged1, merged2)
bk.export(final)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}

func TestBuildWithComplexDiffMerge(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local installed1 = base:run("apk add --no-cache curl")
local installed2 = base:run("apk add --no-cache wget")
local diff1 = bk.diff(base, installed1)
local diff2 = bk.diff(base, installed2)
local merged = bk.merge(diff1, diff2)
local final = base:copy(merged, "/", "/tools")
bk.export(final)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
}
