package integration

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	"github.com/stretchr/testify/require"
)

func TestA01_ImageSourceResolvesShorthandReference(t *testing.T) {
	script := `
local base = bk.image("ubuntu:24.04")
bk.export(base)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if source := op.GetSource(); source != nil {
			requireSourceOpCount(t, pbDef, 1)
			return
		}
	}
	t.Fatal("should have source op")
}

func TestA04_ScratchSource(t *testing.T) {
	script := `
bk.export(bk.scratch())
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	require.Equal(t, 2, len(pbDef.Def), "should have exactly 2 ops (scratch source + empty exec)")
	requireSourceOpCount(t, pbDef, 1)

	foundSource := false
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		source := op.GetSource()
		if source != nil {
			foundSource = true
			identifier := source.GetIdentifier()
			require.True(t, identifier == "" || identifier == "scratch", "scratch identifier should be empty or 'scratch'")
		}
	}
	require.True(t, foundSource, "should have found a source op")
}

func TestA05_LocalSourceWithFilters(t *testing.T) {
	script := `
local ctx = bk.local_("context", {
    include = {"*.go", "go.mod"},
    exclude = {"vendor/"},
    shared_key_hint = "go-sources"
})
bk.export(ctx)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if source := op.GetSource(); source != nil {
			require.Equal(t, "local://context", source.GetIdentifier())
			attrs := source.GetAttrs()
			require.Contains(t, attrs, "includepattern0")
			require.Contains(t, attrs, "excludepattern0")
			require.Contains(t, attrs, "sharedkeyhint")
			return
		}
	}
	t.Fatal("should have local source op with filter attrs")
}

func TestA06_ExecWithStringCommandAndOptions(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local s = base:run("whoami", {
    cwd = "/tmp",
    user = "nobody",
    env = { "FOO=bar" },
})
bk.export(s)
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireExecOpCount(t, pbDef, 1)
	requireExecMeta(t, pbDef, "/tmp", "nobody", []string{"FOO=bar"})
	requireSourceOpCount(t, pbDef, 1)
}

func TestA07_ExecWithArrayCommand(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run({"ls", "-la", "/"}))
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			require.Equal(t, []string{"ls", "-la", "/"}, exec.GetMeta().GetArgs())
			return
		}
	}
	t.Fatal("should have exec op with array args")
}

func TestA08_StateImmutabilityProducesForkedDAG(t *testing.T) {
	script := `
local s1 = bk.image("alpine:3.19")
local s2 = s1:run("echo a")
local s3 = s1:run("echo b")
bk.export(bk.merge(s2, s3))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireExecOpCount(t, pbDef, 2)
	requireSourceOpCount(t, pbDef, 1)
	requireMergeOp(t, pbDef, 2)
}

func TestA09_ValidParseableProtobuf(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	require.NotEmpty(t, pbDef.Def, "def array should be non-empty")
}

func TestA10_DeterministicOutput(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`
	requireDeterministic(t, script)
}

func TestA11_CopySerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local s = base:copy(src, "/app/build/", "/app/", {
    owner = { user = "app", group = "app" },
    mode = "0755",
})
bk.export(s)
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 2)
	requireFileOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if file := op.GetFile(); file != nil {
			require.Len(t, file.Actions, 1)
			copy := file.Actions[0].GetCopy()
			require.NotNil(t, copy, "should be a copy action")
			return
		}
	}
	t.Fatal("should have file op with copy action")
}

func TestA12_MkdirSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:mkdir("/app/data/logs", {
    mode = 0755,
    make_parents = true,
    owner = { user = 1000, group = 1000 },
}))
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireFileOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if file := op.GetFile(); file != nil {
			require.Len(t, file.Actions, 1)
			mkdir := file.Actions[0].GetMkdir()
			require.NotNil(t, mkdir, "should be a mkdir action")
			require.Equal(t, "/app/data/logs", mkdir.Path)
			require.True(t, mkdir.MakeParents)
			require.Equal(t, int32(755), mkdir.Mode)
			return
		}
	}
	t.Fatal("should have file op with mkdir action")
}

func TestA13_MkfileSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:mkfile("/etc/config.json", '{"key": "value"}', { mode = 0644 }))
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireFileOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if file := op.GetFile(); file != nil {
			require.Len(t, file.Actions, 1)
			mkfile := file.Actions[0].GetMkfile()
			require.NotNil(t, mkfile, "should be a mkfile action")
			require.Equal(t, "/etc/config.json", mkfile.Path)
			require.Equal(t, `{"key": "value"}`, string(mkfile.Data))
			require.Equal(t, int32(644), mkfile.Mode)
			return
		}
	}
	t.Fatal("should have file op with mkfile action")
}

func TestA14_RmSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:rm("/tmp/junk", { allow_not_found = true }))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireFileOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if file := op.GetFile(); file != nil {
			require.Len(t, file.Actions, 1)
			rm := file.Actions[0].GetRm()
			require.NotNil(t, rm, "should be an rm action")
			require.Equal(t, "/tmp/junk", rm.Path)
			require.True(t, rm.AllowNotFound)
			return
		}
	}
	t.Fatal("should have file op with rm action")
}

func TestA15_SymlinkSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:symlink("/usr/bin/python3", "/usr/bin/python"))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireFileOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if file := op.GetFile(); file != nil {
			require.Len(t, file.Actions, 1)
			symlink := file.Actions[0].GetSymlink()
			require.NotNil(t, symlink, "should be a symlink action")
			require.Equal(t, "/usr/bin/python3", symlink.Oldpath)
			require.Equal(t, "/usr/bin/python", symlink.Newpath)
			return
		}
	}
	t.Fatal("should have file op with symlink action")
}

func TestA16_CacheMountSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("npm ci", {
    mounts = { bk.cache("/root/.npm", { sharing = "shared", id = "npm-cache" }) },
}))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireMountOfType(t, pbDef, pb.MountType_CACHE, "/root/.npm")
}

func TestA17_SecretMountSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("cat /run/secrets/npmrc", {
    mounts = { bk.secret("/run/secrets/npmrc", { id = "npmrc", mode = 0400 }) },
}))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireMountOfType(t, pbDef, pb.MountType_SECRET, "/run/secrets/npmrc")
}

func TestA18_SSHMountSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("ssh-add -l", {
    mounts = { bk.ssh({ id = "default", mode = 0600 }) },
}))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireMountOfType(t, pbDef, pb.MountType_SSH, "/run/ssh")
}

func TestA19_TmpfsMountSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("df /tmp", {
    mounts = { bk.tmpfs("/tmp", { size = 1073741824 }) },
}))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireMountOfType(t, pbDef, pb.MountType_TMPFS, "/tmp")
}

func TestA20_BindMountSerialization(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local other = bk.image("ubuntu:24.04")
bk.export(base:run("ls /mnt", {
    mounts = { bk.bind(other, "/mnt", { selector = "/etc", readonly = true }) },
}))
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 2)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			for _, mount := range exec.Mounts {
				if mount.MountType == pb.MountType_BIND && mount.Dest != "/" {
					require.Equal(t, "/mnt", mount.GetDest())
					require.True(t, mount.GetReadonly())
					return
				}
			}
		}
	}
	t.Fatal("should have bind mount")
}

func TestA21_MergeDAGTopology(t *testing.T) {
	script := `
local base = bk.image("node:20")
local deps = base:run("npm ci", { cwd = "/app" })
local lint = deps:run("npm run lint", { cwd = "/app" })
local test = deps:run("npm run test", { cwd = "/app" })
local build = deps:run("npm run build", { cwd = "/app" })
bk.export(bk.merge(lint, test, build))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireMergeOp(t, pbDef, 3)
}

func TestA22_DiffDAGTopology(t *testing.T) {
	script := `
local base = bk.image("ubuntu:24.04")
local installed = base:run("apt-get update && apt-get install -y nginx")
local delta = bk.diff(base, installed)
bk.export(delta)
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireDiffOp(t, pbDef)
}

func TestA23_GitSourceSerialization(t *testing.T) {
	script := `
local repo = bk.git("https://github.com/moby/buildkit.git", {
    ref = "v0.12.0",
    keep_git_dir = false,
})
bk.export(repo)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if source := op.GetSource(); source != nil {
			require.Contains(t, source.GetIdentifier(), "git://https://github.com/moby/buildkit.git#v0.12.0")
			return
		}
	}
	t.Fatal("should have git source op")
}

func TestA24_HTTPSourceSerialization(t *testing.T) {
	script := `
local file = bk.http("https://example.com/archive.tar.gz", {
    checksum = "sha256:abc123",
    filename = "archive.tar.gz",
    chmod = 0644,
})
bk.export(file)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 1)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if source := op.GetSource(); source != nil {
			require.Equal(t, "https://example.com/archive.tar.gz", source.GetIdentifier())
			attrs := source.GetAttrs()
			require.Contains(t, attrs, "checksum")
			require.Contains(t, attrs, "filename")
			return
		}
	}
	t.Fatal("should have http source op")
}

func TestA25_NetworkModeNone(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("echo offline", { network = "none" }))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireNetworkMode(t, pbDef, pb.NetMode_NONE)
}

func TestA26_SecurityModeInsecure(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("echo privileged", { security = "insecure" }))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSecurityMode(t, pbDef, pb.SecurityMode_INSECURE)
}

func TestA27_ValidExitCodes(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
bk.export(base:run("grep maybe /file", { valid_exit_codes = {0, 1} }))
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			require.Contains(t, exec.GetMeta().GetValidExitCodes(), int32(0))
			require.Contains(t, exec.GetMeta().GetValidExitCodes(), int32(1))
			return
		}
	}
	t.Fatal("should have exec op with valid exit codes")
}

func TestA28_MetadataOnOperation(t *testing.T) {
	script := `
local base = bk.image("alpine:3.19")
local s = base:run("make"):with_metadata({
    description = "Compiling application",
    progress_group = "build",
})
bk.export(s)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	require.NotEmpty(t, pbDef.Def, "should have ops")
}

func TestA29_PlatformSpecifierObject(t *testing.T) {
	script := `
local p = bk.platform("linux", "arm64", "v8")
local base = bk.image("ubuntu:24.04", { platform = p })
bk.export(base)
 `
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 1)
}

func TestA30_SourceMappingLuaLineNumbers(t *testing.T) {
	script := `
-- line 1: comment
-- line 2: comment
local base = bk.image("alpine:3.19")
-- line 4: comment
local s = base:run("echo hi")
bk.export(s)
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceMapping(t, pbDef)

	require.NotEmpty(t, pbDef.Source.Locations, "should have location mappings")
}

func TestA31_MultiStageDAGEdgeTopology(t *testing.T) {
	script := `
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server .", { cwd = "/app" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final)
`
	scriptPath := createTestScript(t, script)

	def, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err)

	pbDef := requireValidDefinition(t, def)

	requireSourceOpCount(t, pbDef, 3)
	requireExecOpCount(t, pbDef, 1)
	requireFileOpCount(t, pbDef, 2)
}
