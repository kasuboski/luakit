//go:build e2e
// +build e2e

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kasuboski/luakit/pkg/luavm"
	pb "github.com/moby/buildkit/solver/pb"
	"github.com/stretchr/testify/require"
)

func TestDefinitionValidation(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	tests := []struct {
		name     string
		script   string
		validate func(t *testing.T, def *pb.Definition)
	}{
		{
			name: "simple exec op",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`,
			validate: func(t *testing.T, def *pb.Definition) {
				require.NotEmpty(t, def.Def, "definition should have ops")
				require.NotNil(t, def.Metadata, "definition should have metadata")

				var hasExecOp bool
				var hasSourceOp bool

				for _, opBytes := range def.Def {
					var op pb.Op
					if err := op.UnmarshalVT(opBytes); err == nil {
						if op.GetExec() != nil {
							hasExecOp = true
							require.NotNil(t, op.GetExec().Meta, "exec op should have meta")
							require.NotEmpty(t, op.GetExec().Meta.Args, "exec op should have args")
						}
						if op.GetSource() != nil {
							hasSourceOp = true
							require.NotEmpty(t, op.GetSource().Identifier, "source op should have identifier")
						}
					}
				}

				require.True(t, hasSourceOp, "definition should have at least one source op")
				require.True(t, hasExecOp, "definition should have at least one exec op")
			},
		},
		{
			name: "merge op",
			script: `
local base = bk.image("alpine:3.19")
local a = base:run("echo a > /a.txt")
local b = base:run("echo b > /b.txt")
local merged = bk.merge(a, b)
bk.export(merged)
`,
			validate: func(t *testing.T, def *pb.Definition) {
				require.NotEmpty(t, def.Def, "definition should have ops")

				var hasMergeOp bool
				for _, opBytes := range def.Def {
					var op pb.Op
					if err := op.UnmarshalVT(opBytes); err == nil {
						if op.GetMerge() != nil {
							hasMergeOp = true
							require.GreaterOrEqual(t, len(op.Inputs), 2, "merge op should have at least 2 inputs")
						}
					}
				}

				require.True(t, hasMergeOp, "definition should have a merge op")
			},
		},
		{
			name: "diff op",
			script: `
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache curl")
local diff = bk.diff(base, installed)
bk.export(diff)
`,
			validate: func(t *testing.T, def *pb.Definition) {
				require.NotEmpty(t, def.Def, "definition should have ops")

				var hasDiffOp bool
				for _, opBytes := range def.Def {
					var op pb.Op
					if err := op.UnmarshalVT(opBytes); err == nil {
						if op.GetDiff() != nil {
							hasDiffOp = true
							require.Equal(t, 2, len(op.Inputs), "diff op should have exactly 2 inputs")
						}
					}
				}

				require.True(t, hasDiffOp, "definition should have a diff op")
			},
		},
		{
			name: "file op with multiple actions",
			script: `
local base = bk.image("alpine:3.19")
local with_dir = base:mkdir("/app")
local with_file = with_dir:mkfile("/app/test.txt", "content")
bk.export(with_file)
`,
			validate: func(t *testing.T, def *pb.Definition) {
				require.NotEmpty(t, def.Def, "definition should have ops")

				var hasFileOp bool
				for _, opBytes := range def.Def {
					var op pb.Op
					if err := op.UnmarshalVT(opBytes); err == nil {
						if op.GetFile() != nil {
							hasFileOp = true
							fileOp := op.GetFile()
							require.NotEmpty(t, fileOp.Actions, "file op should have actions")
						}
					}
				}

				require.True(t, hasFileOp, "definition should have a file op")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(tt.script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed")

			var pbDef pb.Definition
			require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

			tt.validate(t, &pbDef)
		})
	}
}

func TestSourceMapGeneration(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")

	var pbDef pb.Definition
	require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

	require.NotNil(t, pbDef.Source, "definition should have source info")
	require.NotNil(t, pbDef.Source.Locations, "source should have locations map")

	hasLuaSource := false
	for _, info := range pbDef.Source.Locations {
		if len(info.Infos) > 0 {
			filename := pbDef.Source.Filenames[info.Infos[0].Filename]
			if strings.Contains(filename, ".lua") {
				hasLuaSource = true
				break
			}
		}
	}

	require.True(t, hasLuaSource, "source info should reference Lua files")
}

func TestImageConfigSerialization(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "echo test"},
    env = {"VAR1=value1", "VAR2=value2"},
    user = "nobody",
    workdir = "/app",
    labels = {
        ["label1"] = "value1",
        ["label2"] = "value2",
    },
})
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")

	var pbDef pb.Definition
	require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

	require.NotEmpty(t, pbDef.Metadata, "definition should have metadata")

	foundConfig := false
	for digest, meta := range pbDef.Metadata {
		if meta.Description != nil {
			if configStr, ok := meta.Description["moby.buildkit.image.config"]; ok {
				foundConfig = true

				var config map[string]interface{}
				require.NoError(t, json.Unmarshal([]byte(configStr), &config), "config should unmarshal as JSON")

				configData := config["Config"].(map[string]interface{})

				require.NotNil(t, configData["Entrypoint"], "config should have entrypoint")
				require.NotNil(t, configData["Cmd"], "config should have cmd")
				require.NotNil(t, configData["Env"], "config should have env")
				require.NotNil(t, configData["User"], "config should have user")
				require.NotNil(t, configData["WorkingDir"], "config should have workdir")

				labels := configData["Labels"].(map[string]interface{})
				require.NotNil(t, labels["label1"], "config should have label1")
				require.NotNil(t, labels["label2"], "config should have label2")

				break
			}
		}
	}

	require.True(t, foundConfig, "definition should contain image config in metadata")
}

func TestDigestComputation(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def1, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "first build should succeed")

	def2, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "second build should succeed")

	var pbDef1 pb.Definition
	var pbDef2 pb.Definition
	require.NoError(t, pbDef1.UnmarshalVT(def1))
	require.NoError(t, pbDef2.UnmarshalVT(def2))

	require.Equal(t, pbDef1.Def, pbDef2.Def, "ops should be identical")
	require.Equal(t, pbDef1.Metadata, pbDef2.Metadata, "metadata should be identical")

	digests1 := make(map[string]bool)
	for digest := range pbDef1.Metadata {
		digests1[digest] = true
	}

	digests2 := make(map[string]bool)
	for digest := range pbDef2.Metadata {
		digests2[digest] = true
	}

	for digest := range digests1 {
		require.True(t, digests2[digest], "digest %s should be present in both definitions", digest)
	}
}

func TestMountSerialization(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	tests := []struct {
		name       string
		script     string
		mountType  string
		validateOp func(t *testing.T, mount *pb.Mount)
	}{
		{
			name: "cache mount",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo test", {
    mounts = { bk.cache("/cache", { id = "test-cache", sharing = "locked" }) },
})
bk.export(result)
`,
			mountType: "cache",
			validateOp: func(t *testing.T, mount *pb.Mount) {
				require.Equal(t, pb.MountType_CACHE, mount.MountType)
				require.NotNil(t, mount.CacheOpt)
				require.Equal(t, "test-cache", mount.CacheOpt.ID)
				require.Equal(t, pb.CacheSharingOpt_LOCKED, mount.CacheOpt.Sharing)
			},
		},
		{
			name: "secret mount",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo test", {
    mounts = { bk.secret("/run/secrets/my-secret", { id = "my-secret", mode = 0400 }) },
})
bk.export(result)
`,
			mountType: "secret",
			validateOp: func(t *testing.T, mount *pb.Mount) {
				require.Equal(t, pb.MountType_SECRET, mount.MountType)
				require.NotNil(t, mount.SecretOpt)
				require.Equal(t, "my-secret", mount.SecretOpt.ID)
				require.Equal(t, uint32(0400), mount.SecretOpt.Mode)
			},
		},
		{
			name: "ssh mount",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo test", {
    mounts = { bk.ssh({ id = "default", mode = 0600 }) },
})
bk.export(result)
`,
			mountType: "ssh",
			validateOp: func(t *testing.T, mount *pb.Mount) {
				require.Equal(t, pb.MountType_SSH, mount.MountType)
				require.NotNil(t, mount.SSHOpt)
				require.Equal(t, "default", mount.SSHOpt.ID)
				require.Equal(t, uint32(0600), mount.SSHOpt.Mode)
			},
		},
		{
			name: "tmpfs mount",
			script: `
local base = bk.image("alpine:3.19")
local result = base:run("echo test", {
    mounts = { bk.tmpfs("/tmp", { size = 67108864 }) },
})
bk.export(result)
`,
			mountType: "tmpfs",
			validateOp: func(t *testing.T, mount *pb.Mount) {
				require.Equal(t, pb.MountType_TMPFS, mount.MountType)
				require.NotNil(t, mount.TmpfsOpt)
				require.Equal(t, int64(67108864), mount.TmpfsOpt.Size)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(tt.script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed")

			var pbDef pb.Definition
			require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

			foundMount := false
			for _, opBytes := range pbDef.Def {
				var op pb.Op
				if err := op.UnmarshalVT(opBytes); err == nil {
					if exec := op.GetExec(); exec != nil {
						for _, mount := range exec.Mounts {
							if mount.Dest == "/cache" || mount.Dest == "/run/secrets/my-secret" ||
								mount.Dest == "/run/ssh" || mount.Dest == "/tmp" {
								foundMount = true
								tt.validateOp(t, mount)
								break
							}
						}
					}
				}
			}

			require.True(t, foundMount, "definition should contain expected mount type")
		})
	}
}

func TestPlatformSerialization(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	platforms := []struct {
		name     string
		platform string
		os       string
		arch     string
		variant  string
	}{
		{"linux/amd64", "linux/amd64", "linux", "amd64", ""},
		{"linux/arm64", "linux/arm64", "linux", "arm64", ""},
		{"linux/arm/v7", "linux/arm/v7", "linux", "arm", "v7"},
	}

	for _, tt := range platforms {
		t.Run(tt.name, func(t *testing.T) {
			script := fmt.Sprintf(`
local base = bk.image("alpine:3.19", { platform = "%s" })
local result = base:run("echo test")
bk.export(result)
`, tt.platform)

			scriptPath := filepath.Join(t.TempDir(), "build.lua")
			require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

			def, err := runLuakitBuild(t, scriptPath, ".")
			require.NoError(t, err, "luakit build should succeed")

			var pbDef pb.Definition
			require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

			foundPlatform := false
			for _, opBytes := range pbDef.Def {
				var op pb.Op
				if err := op.UnmarshalVT(opBytes); err == nil {
					if source := op.GetSource(); source != nil && op.Platform != nil {
						foundPlatform = true
						require.Equal(t, tt.os, op.Platform.OS, "OS should match")
						require.Equal(t, tt.arch, op.Platform.Architecture, "Architecture should match")
						if tt.variant != "" {
							require.Equal(t, tt.variant, op.Platform.Variant, "Variant should match")
						}
						break
					}
				}
			}

			require.True(t, foundPlatform, "definition should contain platform info")
		})
	}
}

func TestExecOptionsSerialization(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run({"sh", "-c", "echo test"}, {
    cwd = "/app",
    user = "nobody",
    env = {"VAR1=value1", "VAR2=value2"},
    network = "none",
    security = "sandbox",
    hostname = "test-host",
    valid_exit_codes = {0, 1},
})
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")

	var pbDef pb.Definition
	require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

	foundExec := false
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		if err := op.UnmarshalVT(opBytes); err == nil {
			if exec := op.GetExec(); exec != nil {
				foundExec = true
				require.NotNil(t, exec.Meta, "exec op should have meta")
				require.Equal(t, "/app", exec.Meta.Cwd, "cwd should match")
				require.Equal(t, "nobody", exec.Meta.User, "user should match")
				require.Contains(t, exec.Meta.Env, "VAR1=value1", "env should contain VAR1")
				require.Contains(t, exec.Meta.Env, "VAR2=value2", "env should contain VAR2")
				require.Equal(t, "test-host", exec.Meta.Hostname, "hostname should match")
				require.Equal(t, pb.NetMode_NONE, exec.Network, "network should be none")
				require.Equal(t, pb.SecurityMode_SANDBOX, exec.Security, "security should be sandbox")
				require.Contains(t, exec.Meta.ValidExitCodes, int32(0), "valid_exit_codes should contain 0")
				require.Contains(t, exec.Meta.ValidExitCodes, int32(1), "valid_exit_codes should contain 1")
				break
			}
		}
	}

	require.True(t, foundExec, "definition should contain exec op with all options")
}

func TestFileActionSerialization(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local with_dir = base:mkdir("/app", { mode = 0755, make_parents = true })
local with_file = with_dir:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local with_copy = with_file:copy(with_file, "/app/config.json", "/app/config-backup.json")
local with_symlink = with_symlink:mkfile("/app/target.txt", "target")
local with_link = with_symlink:symlink("/app/target.txt", "/app/link.txt")
local with_rm = with_link:rm("/app/target.txt")
bk.export(with_rm)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")

	var pbDef pb.Definition
	require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")

	foundFileOp := false
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		if err := op.UnmarshalVT(opBytes); err == nil {
			if file := op.GetFile(); file != nil {
				foundFileOp = true
				require.NotEmpty(t, file.Actions, "file op should have actions")

				hasMkdir := false
				hasMkfile := false
				hasCopy := false
				hasSymlink := false
				hasRm := false

				for _, action := range file.Actions {
					switch action := (interface{})(action).(type) {
					case *pb.FileActionMkDir:
						hasMkdir = true
						require.Equal(t, uint32(0755), action.Mode)
					case *pb.FileActionMkFile:
						hasMkfile = true
						require.Equal(t, uint32(0644), action.Mode)
					case *pb.FileActionCopy:
						hasCopy = true
					case *pb.FileActionSymlink:
						hasSymlink = true
					case *pb.FileActionRm:
						hasRm = true
					}
				}

				require.True(t, hasMkdir, "file op should have mkdir action")
				require.True(t, hasMkfile, "file op should have mkfile action")
				require.True(t, hasCopy, "file op should have copy action")
				require.True(t, hasSymlink, "file op should have symlink action")
				require.True(t, hasRm, "file op should have rm action")
				break
			}
		}
	}

	require.True(t, foundFileOp, "definition should contain file op")
}

func TestBuildctlIntegration(t *testing.T) {
	skipIfNoBuildKit(t)
	skipIfNoDocker(t)

	t.Parallel()

	if _, err := exec.LookPath("buildctl"); err != nil {
		t.Skip("buildctl not found in PATH. Install from https://github.com/moby/buildkit")
	}

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /hello.txt")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "luakit build should succeed")

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	outputPath := filepath.Join(t.TempDir(), "output.tar")
	args := []string{
		"build",
		"--progress=plain",
		"--no-frontend",
		"--local", "context=" + t.TempDir(),
		"--output", "type=local,dest=" + outputPath,
	}

	cmd := exec.CommandContext(ctx, "buildctl", args...)
	cmd.Stdin = bytes.NewReader(def)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("buildctl output: %s", string(output))
		require.NoError(t, err, "buildctl should succeed")
	}

	require.FileExists(t, filepath.Join(outputPath, "hello.txt"), "output should contain hello.txt")
}

func TestConcurrency(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
bk.export(base)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	const concurrency = 10
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := runLuakitBuild(t, scriptPath, ".")
			errChan <- err
		}()
	}

	for i := 0; i < concurrency; i++ {
		err := <-errChan
		require.NoError(t, err, "concurrent build should succeed")
	}
}

func TestLuaVMIsolation(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script1 := `
local base = bk.image("alpine:3.19")
local result = base:run("echo test1")
bk.export(result)
`

	script2 := `
local base = bk.image("alpine:3.19")
local result = base:run("echo test2")
bk.export(result)
`

	scriptPath1 := filepath.Join(t.TempDir(), "build1.lua")
	scriptPath2 := filepath.Join(t.TempDir(), "build2.lua")
	require.NoError(t, os.WriteFile(scriptPath1, []byte(script1), 0644))
	require.NoError(t, os.WriteFile(scriptPath2, []byte(script2), 0644))

	luavm.ResetExportedState()
	def1, err := runLuakitBuild(t, scriptPath1, ".")
	require.NoError(t, err, "first build should succeed")

	luavm.ResetExportedState()
	def2, err := runLuakitBuild(t, scriptPath2, ".")
	require.NoError(t, err, "second build should succeed")

	var pbDef1 pb.Definition
	var pbDef2 pb.Definition
	require.NoError(t, pbDef1.UnmarshalVT(def1))
	require.NoError(t, pbDef2.UnmarshalVT(def2))

	require.NotEqual(t, pbDef1.Def, pbDef2.Def, "definitions should be different")
}

func TestLargeScript(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("local base = bk.image(\"alpine:3.19\")\n")

	for i := 0; i < 100; i++ {
		scriptBuilder.WriteString(fmt.Sprintf("local step%d = base:run(\"echo step%d\")\n", i, i))
	}
	scriptBuilder.WriteString(fmt.Sprintf("bk.export(step99)\n"))

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptBuilder.String()), 0644))

	start := time.Now()
	def, err := runLuakitBuild(t, scriptPath, ".")
	duration := time.Since(start)

	require.NoError(t, err, "luakit build should succeed")
	require.NotEmpty(t, def, "definition should not be empty")
	require.Less(t, duration, 10*time.Second, "large script should build in reasonable time")

	t.Logf("Built large script with 100 ops in %v", duration)
}

func TestMemoryUsage(t *testing.T) {
	skipIfNoBuildKit(t)

	t.Parallel()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo test")
bk.export(result)
`

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 10; i++ {
		_, err := runLuakitBuild(t, scriptPath, ".")
		require.NoError(t, err, "build %d should succeed", i)
	}

	runtime.ReadMemStats(&m2)

	allocDiff := m2.Alloc - m1.Alloc
	maxAllocDiff := int64(100 * 1024 * 1024) // 100 MB

	require.Less(t, allocDiff, maxAllocDiff, "memory usage should not grow significantly")

	t.Logf("Memory usage after 10 builds: %d MB (diff: %d MB)", m2.Alloc/1024/1024, allocDiff/1024/1024)
}
