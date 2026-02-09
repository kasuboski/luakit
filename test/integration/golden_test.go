package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	"github.com/stretchr/testify/require"
)

func TestA32_GoldenSimpleImage(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base)
`
	goldenPath := "golden/a32_simple_image.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a32_simple_image.lua")
}

func TestA33_GoldenExecBasic(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`
	goldenPath := "golden/a33_exec_basic.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a33_exec_basic.lua")
}

func TestA34_GoldenExecMounts(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("npm ci", {
    mounts = {
        bk.cache("/root/.npm", { id = "npm-cache" }),
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.ssh({ id = "default" }),
        bk.tmpfs("/tmp", { size = 1073741824 }),
    },
})
bk.export(result)
`
	goldenPath := "golden/a34_exec_mounts.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a34_exec_mounts.lua")
}

func TestA35_GoldenFileCopy(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local result = base:copy(src, "/usr/local/bin/", "/usr/local/bin/")
bk.export(result)
`
	goldenPath := "golden/a35_file_copy.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a35_file_copy.lua")
}

func TestA36_GoldenFileOps(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local s1 = base:mkdir("/app")
local s2 = s1:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local s3 = s2:rm("/app/config.json")
local s4 = s3:symlink("/usr/bin/python3", "/usr/bin/python")
bk.export(s4)
`
	goldenPath := "golden/a36_file_ops.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a36_file_ops.lua")
}

func TestA37_GoldenMergeBasic(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local a = base:run("echo a > /a.txt")
local b = base:run("echo b > /b.txt")
local c = base:run("echo c > /c.txt")
local merged = bk.merge(a, b, c)
bk.export(merged)
`
	goldenPath := "golden/a37_merge_basic.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a37_merge_basic.lua")
}

func TestA38_GoldenDiffBasic(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache nginx")
local delta = bk.diff(base, installed)
bk.export(delta)
`
	goldenPath := "golden/a38_diff_basic.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a38_diff_basic.lua")
}

func TestA39_GoldenMultiStage(t *testing.T) {
	script := `
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", { cwd = "/app" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
`
	goldenPath := "golden/a39_multi_stage.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a39_multi_stage.lua")
}

func TestA40_GoldenPlatform(t *testing.T) {
	script := `
local arm = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
bk.export(arm)
`
	goldenPath := "golden/a40_platform.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a40_platform.lua")
}

func TestA41_GoldenComplexDAG(t *testing.T) {
	script := `
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local deps = builder:copy(src, "go.mod", "/app/go.mod"):copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app", mounts = { bk.cache("/root/.cache/go-build") } })
local workspace = downloaded:copy(src, ".", "/app")
local lint = workspace:run("golangci-lint run ./...", { cwd = "/app" })
local test = workspace:run("go test ./...", { cwd = "/app" })
local build = workspace:run("go build -o /out/server ./cmd/server", { cwd = "/app" })
local merged = bk.merge(lint, test, build)
local runtime = bk.image("gcr.io/distroless/static:nonroot")
local final = runtime:copy(merged, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"}, user = "nonroot" })
`
	goldenPath := "golden/a41_complex_dag.pb"
	requireGoldenFileMatch(t, script, goldenPath, "a41_complex_dag.lua")
}

func requireGoldenFileMatch(t *testing.T, script, goldenPath, scriptName string) {
	t.Helper()

	wd, _ := os.Getwd()
	scriptDir := filepath.Join(wd, "golden_scripts")
	require.NoError(t, os.MkdirAll(scriptDir, 0755))

	scriptPath := filepath.Join(scriptDir, scriptName)
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	luakitPath := filepath.Join(wd, "..", "..", "dist", "luakit")
	cmd := exec.Command(luakitPath, "build", "-o", "-", scriptPath)
	cmd.Dir = scriptDir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "luakit build should succeed: %s", string(output))

	goldenBytes, err := os.ReadFile(filepath.Join("testdata", goldenPath))
	if os.IsNotExist(err) {
		t.Fatalf("golden file not found: testdata/%s", goldenPath)
	}
	require.NoError(t, err, "should be able to read golden file")

	normalizedOutput := normalizeDefinition(output)
	normalizedGolden := normalizeDefinition(goldenBytes)

	require.Equal(t, normalizedGolden, normalizedOutput, "definition should match golden file")
}

func normalizeDefinition(data []byte) []byte {
	var def pb.Definition
	if err := def.UnmarshalVT(data); err != nil {
		return data
	}
	def.Source = nil
	out, _ := def.MarshalVT()
	return out
}

func getScenarioNumber(testName string) int {
	var num int
	for _, c := range testName {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}
