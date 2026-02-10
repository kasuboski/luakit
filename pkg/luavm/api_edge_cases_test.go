package luavm

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestBkImageWithSpecialCharacters(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.image("registry.example.com:5000/username/image:tag-123_v2.0")`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.image with special characters: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkImageWithDigest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.image("alpine@sha256:1234567890abcdef")`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.image with digest: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkImageWithPort(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.image("localhost:5000/alpine:3.19")`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.image with port: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkImageWithTablePlatform(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.image("alpine:3.19", {
			platform = {
				os = "linux",
				arch = "arm64",
				variant = "v8"
			}
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.image with table platform: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestStateRunWithComplexEnv(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:run("echo $VAR", {
			env = {
				PATH = "/usr/local/bin:/usr/bin:/bin",
				HOME = "/home/user",
				EMPTY = "",
				["WITH.SPACE"] = "value with spaces",
				["EQUALS=SIGN"] = "equals in value"
			}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute run with complex env: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	execOp := state.Op().Op().GetExec()
	if len(execOp.Meta.Env) != 5 {
		t.Errorf("Expected 5 env vars, got %d", len(execOp.Meta.Env))
	}
}

func TestStateRunWithSpecialCharsInCwd(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:run("pwd", { cwd = "/app/path with spaces" })
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute run with special chars in cwd: %v", err)
	}

	state := GetExportedState()
	if state.Op().Op().GetExec().Meta.Cwd != "/app/path with spaces" {
		t.Errorf("Expected cwd '/app/path with spaces', got '%s'", state.Op().Op().GetExec().Meta.Cwd)
	}
}

func TestStateRunWithNumericUser(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:run("whoami", { user = "1000" })
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute run with numeric user: %v", err)
	}

	state := GetExportedState()
	if state.Op().Op().GetExec().Meta.User != "1000" {
		t.Errorf("Expected user '1000', got '%s'", state.Op().Op().GetExec().Meta.User)
	}
}

func TestStateRunWithMultipleMounts(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:run("echo test", {
			mounts = {
				bk.cache("/cache1", { id = "cache1", sharing = "shared" }),
				bk.cache("/cache2", { id = "cache2", sharing = "private" }),
				bk.secret("/secret1"),
				bk.secret("/secret2", { id = "secret2", mode = 0600 }),
				bk.ssh(),
				bk.tmpfs("/tmp1"),
				bk.tmpfs("/tmp2", { size = 1024 * 1024 }),
			}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute run with multiple mounts: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	execOp := state.Op().Op().GetExec()

	if len(execOp.Mounts) < 7 {
		t.Errorf("Expected at least 7 mounts, got %d", len(execOp.Mounts))
	}
}

func TestStateRunWithNestedTableEnv(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:run("echo test", {
			env = { KEY = "value" }
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute run with nested table env: %v", err)
	}

	state := GetExportedState()
	env := state.Op().Op().GetExec().Meta.Env
	if len(env) != 1 || env[0] != "KEY=value" {
		t.Errorf("Expected env ['KEY=value'], got %v", env)
	}
}

func TestStateCopyWithSpecialPaths(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		result = dst:copy(src, "/path with spaces", "/dest with spaces")
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute copy with special paths: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	fileOp := state.Op().Op().GetFile()
	copyAction := fileOp.Actions[0].GetCopy()
	if copyAction.Src != "/path with spaces" {
		t.Errorf("Expected src '/path with spaces', got '%s'", copyAction.Src)
	}
	if copyAction.Dest != "/dest with spaces" {
		t.Errorf("Expected dest '/dest with spaces', got '%s'", copyAction.Dest)
	}
}

func TestStateMkdirWithSpecialPaths(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:mkdir("/app/path with spaces")
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute mkdir with special path: %v", err)
	}

	state := GetExportedState()
	mkdirAction := state.Op().Op().GetFile().Actions[0].GetMkdir()
	if mkdirAction.Path != "/app/path with spaces" {
		t.Errorf("Expected path '/app/path with spaces', got '%s'", mkdirAction.Path)
	}
}

func TestStateMkfileWithSpecialChars(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:mkfile("/app/config.json", '{"key": "value with spaces", "key2": "value2"}')
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute mkfile with special chars: %v", err)
	}

	state := GetExportedState()
	mkfileAction := state.Op().Op().GetFile().Actions[0].GetMkfile()
	data := string(mkfileAction.Data)
	if data != `{"key": "value with spaces", "key2": "value2"}` {
		t.Errorf("Expected data with special chars, got '%s'", data)
	}
}

func TestStateSymlinkWithSpecialPaths(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		result = s:symlink("/path with spaces/old", "/path with spaces/new")
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute symlink with special paths: %v", err)
	}

	state := GetExportedState()
	symlinkAction := state.Op().Op().GetFile().Actions[0].GetSymlink()
	if symlinkAction.Oldpath != "/path with spaces/old" {
		t.Errorf("Expected oldpath '/path with spaces/old', got '%s'", symlinkAction.Oldpath)
	}
	if symlinkAction.Newpath != "/path with spaces/new" {
		t.Errorf("Expected newpath '/path with spaces/new', got '%s'", symlinkAction.Newpath)
	}
}

func TestBkMergeWithManyStates(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s1 = bk.image("alpine:3.19")
		local s2 = bk.image("ubuntu:24.04")
		local s3 = bk.image("node:20")
		local s4 = bk.image("golang:1.22")
		local s5 = bk.image("python:3.12")
		local merged = bk.merge(s1, s2, s3, s4, s5)
		bk.export(merged)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute merge with 5 states: %v", err)
	}

	state := GetExportedState()
	mergeOp := state.Op().Op().GetMerge()
	if len(mergeOp.Inputs) != 5 {
		t.Errorf("Expected 5 merge inputs, got %d", len(mergeOp.Inputs))
	}
}

func TestBkDiffComplex(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local installed = base:run("apk add nginx nginx-mod-http-lua")
		local diffed = bk.diff(base, installed)
		bk.export(diffed)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute diff complex: %v", err)
	}

	state := GetExportedState()
	diffOp := state.Op().Op().GetDiff()
	if diffOp == nil {
		t.Fatal("Expected DiffOp")
	}
}

func TestBkMountCacheWithAllOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.cache("/cache", {
			id = "mycache",
			sharing = "locked"
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to create cache mount with all options: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkMountSecretWithAllOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.secret("/run/secrets/secret", {
			id = "mysecret",
			uid = 1000,
			gid = 1000,
			mode = 0600,
			optional = true
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to create secret mount with all options: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkMountSSHWithAllOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.ssh({
			dest = "/custom/ssh",
			id = "myssh",
			uid = 1000,
			gid = 1000,
			mode = 0644,
			optional = true
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to create SSH mount with all options: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkMountTmpfsWithOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.tmpfs("/tmp", { size = 1024 * 1024 * 1024 })
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to create tmpfs mount with options: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkMountBindWithOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		result = bk.bind(s, "/bind/target", {
			selector = "/specific/path",
			readonly = false
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to create bind mount with options: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}
}

func TestBkExportWithComplexConfig(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		bk.export(s, {
			entrypoint = {"/bin/sh", "-c"},
			cmd = {"echo hello"},
			env = {
				PATH = "/usr/local/bin:/usr/bin:/bin",
				["KEY.WITH.DOTS"] = "value"
			},
			workdir = "/app/work dir",
			user = "appuser",
			expose = { "8080/tcp", "9090/udp" },
			labels = {
				["org.opencontainers.image.title"] = "Test",
				["com.example.custom"] = "value"
			},
			os = "linux",
			arch = "arm64",
			variant = "v8"
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute export with complex config: %v", err)
	}

	config := GetExportedImageConfig()
	if config == nil {
		t.Fatal("Expected exported image config to be non-nil")
	}

	if len(config.Config.Entrypoint) != 2 {
		t.Errorf("Expected 2 entrypoint elements, got %d", len(config.Config.Entrypoint))
	}

	if len(config.Config.ExposedPorts) != 2 {
		t.Errorf("Expected 2 exposed ports, got %d", len(config.Config.ExposedPorts))
	}
}

func TestComplexPipelineWithMounts(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local builder = bk.image("golang:1.22")
		local src = bk.local_("context")
		local deps = builder:run("go mod download", {
			cwd = "/app",
			mounts = { bk.cache("/go/pkg/mod") }
		})
		local workspace = deps:copy(src, ".", "/app")
		local built = workspace:run("go build -o /out/server ./cmd/server", {
			cwd = "/app",
			mounts = { bk.cache("/root/.cache/go-build") }
		})
		local runtime = bk.image("gcr.io/distroless/static-debian12")
		local final = runtime:copy(built, "/out/server", "/server")
		bk.export(final, { entrypoint = {"/server"} })
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute complex pipeline: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}
}

func TestErrorPropagationInChain(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		local result = s:run("echo test")
		bk.export(result)
		-- Try to call export again - should error
		bk.export(result)
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling export twice")
	}

	if !strings.Contains(err.Error(), "already called once") {
		t.Errorf("Expected error about already called once, got: %v", err)
	}
}
