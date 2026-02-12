package luavm

import (
	"strings"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/ops"
	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	lua "github.com/yuin/gopher-lua"
)

var testVM *lua.LState

func resetExportedState() {
	if testVM != nil {
		data := getVMData(testVM)
		if data != nil {
			data.exportedState = nil
			data.exportedImageConfig = nil
		}
	}
}

func GetExportedState() *dag.State {
	if testVM == nil {
		return nil
	}
	data := getVMData(testVM)
	if data == nil {
		return nil
	}
	return data.exportedState
}

func GetExportedImageConfig() *dockerspec.DockerOCIImage {
	if testVM == nil {
		return nil
	}
	data := getVMData(testVM)
	if data == nil {
		return nil
	}
	return data.exportedImageConfig
}

func TestBasicLuaScript(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local empty = bk.scratch()
		local ctx = bk.local_("context")

		bk.export(base)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	if state.Op() == nil {
		t.Fatal("Expected state to have an Op")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "docker-image://docker.io/library/alpine:3.19" {
		t.Errorf("Expected identifier 'docker-image://docker.io/library/alpine:3.19', got '%s'", sourceOp.Identifier)
	}
}

func TestBkImageReturnsState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.image("ubuntu:24.04")`); err != nil {
		t.Fatalf("Failed to call bk.image: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	state, ok := ud.Value.(*dag.State)
	if !ok {
		t.Fatal("Expected State value")
	}

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "docker-image://docker.io/library/ubuntu:24.04" {
		t.Errorf("Expected identifier 'docker-image://docker.io/library/ubuntu:24.04', got '%s'", sourceOp.Identifier)
	}
}

func TestBkScratchReturnsState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.scratch()`); err != nil {
		t.Fatalf("Failed to call bk.scratch: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	state, ok := ud.Value.(*dag.State)
	if !ok {
		t.Fatal("Expected State value")
	}

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "scratch" {
		t.Errorf("Expected identifier 'scratch', got '%s'", sourceOp.Identifier)
	}
}

func TestBkLocalReturnsState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.local_("mycontext")`); err != nil {
		t.Fatalf("Failed to call bk.local_: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	state, ok := ud.Value.(*dag.State)
	if !ok {
		t.Fatal("Expected State value")
	}

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "local://mycontext" {
		t.Errorf("Expected identifier 'local://mycontext', got '%s'", sourceOp.Identifier)
	}
}

func TestMultipleSourceOps(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local img1 = bk.image("alpine:3.19")
		local img2 = bk.image("ubuntu:24.04")
		local ctx = bk.local_("context")
		local scr = bk.scratch()

		bk.export(img1)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "docker-image://docker.io/library/alpine:3.19" {
		t.Errorf("Expected identifier 'docker-image://docker.io/library/alpine:3.19', got '%s'", sourceOp.Identifier)
	}
}

func TestImageWithFullDockerRef(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.image("docker.io/library/alpine:3.19")`); err != nil {
		t.Fatalf("Failed to call bk.image: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	expected := "docker-image://docker.io/library/alpine:3.19"
	if sourceOp.Identifier != expected {
		t.Errorf("Expected identifier '%s', got '%s'", expected, sourceOp.Identifier)
	}
}

func TestStateRunWithStringCommand(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo hello")
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	execOp := state.Op().Op().GetExec()
	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Meta.Args) != 3 {
		t.Errorf("Expected 3 args (sh -c command), got %d", len(execOp.Meta.Args))
	}

	if execOp.Meta.Args[0] != "/bin/sh" {
		t.Errorf("Expected args[0] to be '/bin/sh', got '%s'", execOp.Meta.Args[0])
	}

	if execOp.Meta.Args[2] != "echo hello" {
		t.Errorf("Expected args[2] to be 'echo hello', got '%s'", execOp.Meta.Args[2])
	}

	if len(state.Op().Inputs()) != 1 {
		t.Errorf("Expected 1 input edge, got %d", len(state.Op().Inputs()))
	}
}

func TestStateRunWithArrayCommand(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run({"ls", "-la", "/app"})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	execOp := state.Op().Op().GetExec()
	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Meta.Args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(execOp.Meta.Args))
	}

	if execOp.Meta.Args[0] != "ls" {
		t.Errorf("Expected args[0] to be 'ls', got '%s'", execOp.Meta.Args[0])
	}

	if execOp.Meta.Args[1] != "-la" {
		t.Errorf("Expected args[1] to be '-la', got '%s'", execOp.Meta.Args[1])
	}

	if execOp.Meta.Args[2] != "/app" {
		t.Errorf("Expected args[2] to be '/app', got '%s'", execOp.Meta.Args[2])
	}
}

func TestStateRunWithEnv(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo $PATH", {
			env = { PATH = "/usr/bin", FOO = "bar" }
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if len(execOp.Meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(execOp.Meta.Env))
	}

	expectedEnv := map[string]bool{
		"PATH=/usr/bin": false,
		"FOO=bar":       false,
	}

	for _, env := range execOp.Meta.Env {
		if _, ok := expectedEnv[env]; ok {
			expectedEnv[env] = true
		}
	}

	for env, found := range expectedEnv {
		if !found {
			t.Errorf("Expected env var '%s' not found", env)
		}
	}
}

func TestStateRunWithCwd(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("pwd", { cwd = "/app" })
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd to be '/app', got '%s'", execOp.Meta.Cwd)
	}
}

func TestStateRunWithUser(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("whoami", { user = "nobody" })
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp.Meta.User != "nobody" {
		t.Errorf("Expected user to be 'nobody', got '%s'", execOp.Meta.User)
	}
}

func TestStateRunWithAllOptions(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			env = { PATH = "/usr/bin", HOME = "/home/user" },
			cwd = "/workspace",
			user = "builder"
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if len(execOp.Meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(execOp.Meta.Env))
	}

	if execOp.Meta.Cwd != "/workspace" {
		t.Errorf("Expected cwd to be '/workspace', got '%s'", execOp.Meta.Cwd)
	}

	if execOp.Meta.User != "builder" {
		t.Errorf("Expected user to be 'builder', got '%s'", execOp.Meta.User)
	}
}

func TestStateRunWithNetworkAndSecurityOptions(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	tests := []struct {
		name        string
		script      string
		expectedNet pb.NetMode
		expectedSec pb.SecurityMode
	}{
		{
			name: "network none",
			script: `
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", { network = "none" })
				bk.export(result)
			`,
			expectedNet: pb.NetMode_NONE,
			expectedSec: pb.SecurityMode_SANDBOX,
		},
		{
			name: "network host",
			script: `
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", { network = "host" })
				bk.export(result)
			`,
			expectedNet: pb.NetMode_HOST,
			expectedSec: pb.SecurityMode_SANDBOX,
		},
		{
			name: "network sandbox",
			script: `
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", { network = "sandbox" })
				bk.export(result)
			`,
			expectedNet: pb.NetMode_UNSET,
			expectedSec: pb.SecurityMode_SANDBOX,
		},
		{
			name: "security insecure",
			script: `
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", { security = "insecure" })
				bk.export(result)
			`,
			expectedNet: pb.NetMode_UNSET,
			expectedSec: pb.SecurityMode_INSECURE,
		},
		{
			name: "both options",
			script: `
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", { network = "none", security = "insecure" })
				bk.export(result)
			`,
			expectedNet: pb.NetMode_NONE,
			expectedSec: pb.SecurityMode_INSECURE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetExportedState()
			t.Cleanup(resetExportedState)
			L2 := NewVM(nil)
			testVM = L2
			t.Cleanup(func() { L2.Close(); testVM = nil })

			if err := L2.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute Lua script: %v", err)
			}

			state := GetExportedState()
			execOp := state.Op().Op().GetExec()

			if execOp.Network != tc.expectedNet {
				t.Errorf("Expected network mode %v, got %v", tc.expectedNet, execOp.Network)
			}

			if execOp.Security != tc.expectedSec {
				t.Errorf("Expected security mode %v, got %v", tc.expectedSec, execOp.Security)
			}
		})
	}
}

func TestStateRunReturnsNewState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo hello")
		return base ~= result
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}
}

func TestStateRunWithoutCommand(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")
		base:run()
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling run without command")
	}

	if !strings.Contains(err.Error(), "command argument required") {
		t.Errorf("Expected error about missing command, got: %v", err)
	}
}

func TestBkCacheMount(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.cache("/cache", { id = "mycache", sharing = "shared" })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.cache: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	mount, ok := ud.Value.(*ops.Mount)
	if !ok {
		t.Fatal("Expected Mount value")
	}

	if mount == nil {
		t.Fatal("Expected non-nil mount")
	}

	pbMount := mount.ToPB()
	if pbMount.Dest != "/cache" {
		t.Errorf("Expected dest '/cache', got '%s'", pbMount.Dest)
	}

	if pbMount.CacheOpt.ID != "mycache" {
		t.Errorf("Expected cache ID 'mycache', got '%s'", pbMount.CacheOpt.ID)
	}
}

func TestBkCacheMountWithNoOptions(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.cache("/cache")`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.cache: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/cache" {
		t.Errorf("Expected dest '/cache', got '%s'", pbMount.Dest)
	}
}

func TestBkSecretMount(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.secret("/run/secrets/secret", { id = "mysecret", uid = 1000, gid = 1000, mode = 0600, optional = true })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.secret: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/run/secrets/secret" {
		t.Errorf("Expected dest '/run/secrets/secret', got '%s'", pbMount.Dest)
	}

	if pbMount.SecretOpt.ID != "mysecret" {
		t.Errorf("Expected secret ID 'mysecret', got '%s'", pbMount.SecretOpt.ID)
	}

	if pbMount.SecretOpt.Uid != 1000 {
		t.Errorf("Expected uid 1000, got %d", pbMount.SecretOpt.Uid)
	}
}

func TestBkSecretMountDefaults(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.secret("/run/secrets/secret")`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.secret: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.SecretOpt.Uid != 0 {
		t.Errorf("Expected default uid 0, got %d", pbMount.SecretOpt.Uid)
	}

	if pbMount.SecretOpt.Mode != 0400 {
		t.Errorf("Expected default mode 0400, got %d", pbMount.SecretOpt.Mode)
	}
}

func TestBkSSHHMount(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.ssh({ dest = "/custom/ssh", id = "myssh", uid = 1000, gid = 1000, mode = 0644 })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.ssh: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/custom/ssh" {
		t.Errorf("Expected dest '/custom/ssh', got '%s'", pbMount.Dest)
	}

	if pbMount.SSHOpt.ID != "myssh" {
		t.Errorf("Expected ssh ID 'myssh', got '%s'", pbMount.SSHOpt.ID)
	}
}

func TestBkSSHMountDefaults(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.ssh()`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.ssh: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/run/ssh" {
		t.Errorf("Expected default dest '/run/ssh', got '%s'", pbMount.Dest)
	}
}

func TestBkTmpfsMount(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.tmpfs("/tmp", { size = 1073741824 })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.tmpfs: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/tmp" {
		t.Errorf("Expected dest '/tmp', got '%s'", pbMount.Dest)
	}

	if pbMount.TmpfsOpt.Size != 1073741824 {
		t.Errorf("Expected size 1073741824, got %d", pbMount.TmpfsOpt.Size)
	}
}

func TestBkBindMount(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local source = bk.image("alpine:3.19")
		result = bk.bind(source, "/bind/target", { selector = "/specific/path", readonly = false })
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.bind: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	mount := ud.Value.(*ops.Mount)
	pbMount := mount.ToPB()

	if pbMount.Dest != "/bind/target" {
		t.Errorf("Expected dest '/bind/target', got '%s'", pbMount.Dest)
	}

	if pbMount.Selector != "/specific/path" {
		t.Errorf("Expected selector '/specific/path', got '%s'", pbMount.Selector)
	}
}

func TestStateRunWithMounts(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			mounts = {
				bk.cache("/cache", { id = "mycache" }),
				bk.secret("/run/secrets/secret"),
				bk.ssh(),
				bk.tmpfs("/tmp", { size = 67108864 })
			}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if len(execOp.Mounts) != 5 {
		t.Errorf("Expected 5 mounts, got %d", len(execOp.Mounts))
	}

	if execOp.Mounts[1].Dest != "/cache" {
		t.Errorf("Expected second mount dest '/cache', got '%s'", execOp.Mounts[1].Dest)
	}

	if execOp.Mounts[2].Dest != "/run/secrets/secret" {
		t.Errorf("Expected third mount dest '/run/secrets/secret', got '%s'", execOp.Mounts[2].Dest)
	}

	if execOp.Mounts[3].Dest != "/run/ssh" {
		t.Errorf("Expected fourth mount dest '/run/ssh', got '%s'", execOp.Mounts[3].Dest)
	}

	if execOp.Mounts[4].Dest != "/tmp" {
		t.Errorf("Expected fifth mount dest '/tmp', got '%s'", execOp.Mounts[4].Dest)
	}
}

func TestBkMerge(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s1 = bk.image("alpine:3.19")
		local s2 = bk.image("ubuntu:24.04")
		local s3 = bk.image("node:20")
		local merged = bk.merge(s1, s2, s3)
		bk.export(merged)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	mergeOp := state.Op().Op().GetMerge()
	if mergeOp == nil {
		t.Fatal("Expected MergeOp")
	}

	if len(mergeOp.Inputs) != 3 {
		t.Errorf("Expected 3 merge inputs, got %d", len(mergeOp.Inputs))
	}

	if len(state.Op().Inputs()) != 3 {
		t.Errorf("Expected 3 node inputs, got %d", len(state.Op().Inputs()))
	}
}

func TestBkMergeWithTwoStates(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s1 = bk.scratch()
		local s2 = bk.image("alpine:3.19")
		local merged = bk.merge(s1, s2)
		bk.export(merged)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	mergeOp := state.Op().Op().GetMerge()

	if len(mergeOp.Inputs) != 2 {
		t.Errorf("Expected 2 merge inputs, got %d", len(mergeOp.Inputs))
	}
}

func TestBkMergeWithOneState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s1 = bk.image("alpine:3.19")
		local merged = bk.merge(s1)
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling merge with one state")
	}

	if !strings.Contains(err.Error(), "requires at least 2 states") {
		t.Errorf("Expected error about requiring at least 2 states, got: %v", err)
	}
}

func TestBkMergeWithZeroStates(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `local merged = bk.merge()`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling merge with zero states")
	}

	if !strings.Contains(err.Error(), "requires at least 2 states") {
		t.Errorf("Expected error about requiring at least 2 states, got: %v", err)
	}
}

func TestBkDiff(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local lower = bk.image("alpine:3.19")
		local upper = lower:run("echo hello > /greeting.txt")
		local diffed = bk.diff(lower, upper)
		bk.export(diffed)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	diffOp := state.Op().Op().GetDiff()
	if diffOp == nil {
		t.Fatal("Expected DiffOp")
	}

	if diffOp.Lower == nil {
		t.Fatal("Expected non-nil Lower")
	}

	if diffOp.Upper == nil {
		t.Fatal("Expected non-nil Upper")
	}

	if len(state.Op().Inputs()) != 2 {
		t.Errorf("Expected 2 node inputs, got %d", len(state.Op().Inputs()))
	}
}

func TestBkDiffWithOneArg(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local lower = bk.image("alpine:3.19")
		local diffed = bk.diff(lower)
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling diff with one argument")
	}

	if !strings.Contains(err.Error(), "requires lower and upper state") {
		t.Errorf("Expected error about requiring lower and upper state, got: %v", err)
	}
}

func TestBkDiffWithZeroArgs(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `local diffed = bk.diff()`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling diff with zero arguments")
	}

	if !strings.Contains(err.Error(), "requires lower and upper state") {
		t.Errorf("Expected error about requiring lower and upper state, got: %v", err)
	}
}

func TestBkHTTPReturnsState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.http("http://example.com/file.tar.gz")`); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	state, ok := ud.Value.(*dag.State)
	if !ok {
		t.Fatal("Expected State value")
	}

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "http://example.com/file.tar.gz" {
		t.Errorf("Expected identifier 'http://example.com/file.tar.gz', got '%s'", sourceOp.Identifier)
	}
}

func TestBkHTTPSReturnsState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`result = bk.https("https://example.com/file.tar.gz")`); err != nil {
		t.Fatalf("Failed to call bk.https: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", result.Type())
	}

	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Identifier != "https://example.com/file.tar.gz" {
		t.Errorf("Expected identifier 'https://example.com/file.tar.gz', got '%s'", sourceOp.Identifier)
	}
}

func TestBkHTTPWithChecksum(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.http("https://example.com/file.tar.gz", { checksum = "sha256:abc123" })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["checksum"] != "sha256:abc123" {
		t.Errorf("Expected checksum 'sha256:abc123', got '%s'", sourceOp.Attrs["checksum"])
	}
}

func TestBkHTTPWithFilename(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.http("https://example.com/file", { filename = "archive.tar.gz" })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["filename"] != "archive.tar.gz" {
		t.Errorf("Expected filename 'archive.tar.gz', got '%s'", sourceOp.Attrs["filename"])
	}
}

func TestBkHTTPWithMode(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.http("https://example.com/file.tar.gz", { chmod = 0644 })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["mode"] != "644" {
		t.Errorf("Expected mode '644', got '%s'", sourceOp.Attrs["mode"])
	}
}

func TestBkHTTPWithHeaders(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.http("https://example.com/file.tar.gz", { headers = { Authorization = "Bearer token", ["User-Agent"] = "luakit/0.1.0" } })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["http.header.Authorization"] != "Bearer token" {
		t.Errorf("Expected http.header.Authorization 'Bearer token', got '%s'", sourceOp.Attrs["http.header.Authorization"])
	}

	if sourceOp.Attrs["http.header.User-Agent"] != "luakit/0.1.0" {
		t.Errorf("Expected http.header.User-Agent 'luakit/0.1.0', got '%s'", sourceOp.Attrs["http.header.User-Agent"])
	}
}

func TestBkHTTPWithBasicAuth(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `result = bk.http("https://example.com/file.tar.gz", { username = "user", password = "pass" })`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["http.basicauth"] != "user:pass" {
		t.Errorf("Expected http.basicauth 'user:pass', got '%s'", sourceOp.Attrs["http.basicauth"])
	}
}

func TestBkHTTPWithAllOptions(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		result = bk.http("https://example.com/file", {
			checksum = "sha256:abc123",
			filename = "archive.tar.gz",
			chmod = 0644,
			headers = { Authorization = "Bearer token" },
			username = "user",
			password = "pass"
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to call bk.http: %v", err)
	}

	result := L.GetGlobal("result")
	ud := result.(*lua.LUserData)
	state := ud.Value.(*dag.State)
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["checksum"] != "sha256:abc123" {
		t.Errorf("Expected checksum 'sha256:abc123', got '%s'", sourceOp.Attrs["checksum"])
	}

	if sourceOp.Attrs["filename"] != "archive.tar.gz" {
		t.Errorf("Expected filename 'archive.tar.gz', got '%s'", sourceOp.Attrs["filename"])
	}

	if sourceOp.Attrs["mode"] != "644" {
		t.Errorf("Expected mode '644', got '%s'", sourceOp.Attrs["mode"])
	}

	if sourceOp.Attrs["http.header.Authorization"] != "Bearer token" {
		t.Errorf("Expected http.header.Authorization 'Bearer token', got '%s'", sourceOp.Attrs["http.header.Authorization"])
	}

	if sourceOp.Attrs["http.basicauth"] != "user:pass" {
		t.Errorf("Expected http.basicauth 'user:pass', got '%s'", sourceOp.Attrs["http.basicauth"])
	}
}

func TestBkHTTPEmptyURL(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `bk.http("")`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling http with empty URL")
	}

	if !strings.Contains(err.Error(), "URL must not be empty") {
		t.Errorf("Expected error about empty URL, got: %v", err)
	}
}

func TestBkHTTPSEmptyURL(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `bk.https("")`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling https with empty URL")
	}

	if !strings.Contains(err.Error(), "URL must not be empty") {
		t.Errorf("Expected error about empty URL, got: %v", err)
	}
}

func TestBkHTTPInBuild(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local file = bk.http("https://example.com/archive.tar.gz", {
			checksum = "sha256:abc123",
			filename = "archive.tar.gz",
			chmod = 0644
		})
		local base = bk.image("alpine:3.19")
		local result = base:copy(file, "archive.tar.gz", "/app/archive.tar.gz")
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}
}
