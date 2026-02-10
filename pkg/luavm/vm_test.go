package luavm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/kasuboski/luakit/pkg/dag"
)

func TestNewVM(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if L == nil {
		t.Fatal("Expected non-nil VM")
	}
}

func TestRegisterStateType(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	state := &dag.State{}
	ud := newState(L, state)

	if ud.Value != state {
		t.Error("Expected userdata to contain state")
	}
}

func TestCheckState(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	state := &dag.State{}
	ud := newState(L, state)

	L.Push(ud)
	result := checkState(L, -1)

	if result != state {
		t.Error("Expected state to be returned")
	}

	L.Pop(1)
}

func TestCheckStateError(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(lua.LString("not a state"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when checking non-state")
		}
	}()

	checkState(L, -1)
}

func TestStateToString(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	state := &dag.State{}
	ud := newState(L, state)

	L.SetGlobal("test_state", ud)

	if err := L.DoString("result = tostring(test_state)"); err != nil {
		t.Fatalf("Failed to run Lua: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTString {
		t.Errorf("Expected string type, got %v", result.Type())
	}
}

func TestAPIRegistration(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	bk := L.GetGlobal("bk")
	if bk.Type() != lua.LTTable {
		t.Error("Expected bk to be a table")
	}

	imageFunc := L.GetField(bk, "image")
	if imageFunc.Type() != lua.LTFunction {
		t.Error("Expected bk.image to be a function")
	}

	scratchFunc := L.GetField(bk, "scratch")
	if scratchFunc.Type() != lua.LTFunction {
		t.Error("Expected bk.scratch to be a function")
	}

	localFunc := L.GetField(bk, "local_")
	if localFunc.Type() != lua.LTFunction {
		t.Error("Expected bk.local_ to be a function")
	}

	exportFunc := L.GetField(bk, "export")
	if exportFunc.Type() != lua.LTFunction {
		t.Error("Expected bk.export to be a function")
	}
}

func TestBkImageEmptyString(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "image"))
	L.Remove(-2)

	L.Push(lua.LString(""))

	err := L.PCall(0, 0, nil)
	if err == nil {
		t.Error("Expected error when calling bk.image with empty string")
	}
}

func TestBkImage(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "image"))
	L.Remove(-2)

	L.Push(lua.LString("alpine:3.19"))

	if err := L.PCall(1, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.image: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkImageWithPrefix(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "image"))
	L.Remove(-2)

	L.Push(lua.LString("docker-image://alpine:3.19"))

	if err := L.PCall(1, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.image: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkImageWithPlatform(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "image"))
	L.Remove(-2)

	L.Push(lua.LString("alpine:3.19"))

	opts := L.NewTable()
	platform := L.NewTable()
	L.SetField(platform, "os", lua.LString("linux"))
	L.SetField(platform, "arch", lua.LString("arm64"))
	L.SetField(opts, "platform", platform)
	L.Push(opts)

	if err := L.PCall(2, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.image with platform: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkScratch(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "scratch"))
	L.Remove(-2)

	if err := L.PCall(0, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.scratch: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkLocal(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "local_"))
	L.Remove(-2)

	L.Push(lua.LString("context"))

	if err := L.PCall(1, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.local_: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkLocalEmptyString(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "local_"))
	L.Remove(-2)

	L.Push(lua.LString(""))

	err := L.PCall(0, 0, nil)
	if err == nil {
		t.Error("Expected error when calling bk.local_ with empty string")
	}
}

func TestStateRunMissingArg(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	if err := L.DoString(`
		local s = {type = "userdata"}
		function s:run() end
	`); err != nil {
		t.Fatalf("Failed to run Lua: %v", err)
	}

	if err := L.DoString("bk.image('alpine:3.19'):run()"); err == nil {
		t.Error("Expected error when calling run without arguments")
	}
}

func TestSandbox(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	os := L.GetGlobal("os")
	if os.Type() == lua.LTNil {
		t.Error("Expected os to be a table after sandboxing")
	}
	osExecute := L.GetField(os, "execute")
	if osExecute.Type() != lua.LTNil {
		t.Error("Expected os.execute to be nil after sandboxing")
	}

	io := L.GetGlobal("io")
	if io.Type() == lua.LTNil {
		t.Error("Expected io to be a table after sandboxing")
	}
	ioOpen := L.GetField(io, "open")
	if ioOpen.Type() != lua.LTNil {
		t.Error("Expected io.open to be nil after sandboxing")
	}

	debug := L.GetGlobal("debug")
	if debug.Type() != lua.LTNil {
		t.Error("Expected debug to be nil after sandboxing")
	}

	loadfile := L.GetGlobal("loadfile")
	if loadfile.Type() != lua.LTNil {
		t.Error("Expected loadfile to be nil after sandboxing")
	}

	dofile := L.GetGlobal("dofile")
	if dofile.Type() != lua.LTNil {
		t.Error("Expected dofile to be nil after sandboxing")
	}
}

func TestGetExportedState(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	if GetExportedState() != nil {
		t.Error("Expected exported state to be nil before export")
	}

	if err := L.DoString(`
		local s = {type = "userdata"}
		function s:run(cmd) return s end
		bk.image = function(ref) return s end
		bk.export = function(st) end
	`); err != nil {
		t.Fatalf("Failed to run Lua: %v", err)
	}
}

func TestLuaScriptExecution(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = {}
		setmetatable(s, {__index = function(t, k)
			if k == "run" then
				return function(self, cmd) return s end
			end
		end})

		bk.image = function(ref)
			return s
		end

		bk.export = function(st)
		end

		bk.image("alpine:3.19")
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}
}

func TestLuaScriptFileExecution(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.lua")
	scriptContent := `
		local s = {}
		setmetatable(s, {__index = function(t, k)
			if k == "run" then
				return function(self, cmd) return s end
			end
		end})

		bk.image = function(ref)
			return s
		end

		bk.export = function(st)
		end

		bk.image("alpine:3.19")
	`

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script file: %v", err)
	}

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute Lua script file: %v", err)
	}
}

func TestSandboxOsExecuteBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("os.execute('echo test')")
	if err == nil {
		t.Fatal("Expected error when calling os.execute, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxOsExitBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("os.exit(0)")
	if err == nil {
		t.Fatal("Expected error when calling os.exit, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxOsRemoveBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("os.remove('/tmp/test')")
	if err == nil {
		t.Fatal("Expected error when calling os.remove, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxOsRenameBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("os.rename('/tmp/a', '/tmp/b')")
	if err == nil {
		t.Fatal("Expected error when calling os.rename, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxOsTmpnameBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("os.tmpname()")
	if err == nil {
		t.Fatal("Expected error when calling os.tmpname, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxIoOpenBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("io.open('/tmp/test', 'r')")
	if err == nil {
		t.Fatal("Expected error when calling io.open, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxIoPopenBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("io.popen('ls')")
	if err == nil {
		t.Fatal("Expected error when calling io.popen, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxIoInputBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("io.input('/tmp/test')")
	if err == nil {
		t.Fatal("Expected error when calling io.input, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxIoOutputBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("io.output('/tmp/test')")
	if err == nil {
		t.Fatal("Expected error when calling io.output, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxIoLinesBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("io.lines('/tmp/test')")
	if err == nil {
		t.Fatal("Expected error when calling io.lines, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxDebugBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("debug.debug()")
	if err == nil {
		t.Fatal("Expected error when calling debug, but it succeeded")
	}

	expectedError := "attempt to index a non-table object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxLoadfileBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("loadfile('/tmp/test.lua')")
	if err == nil {
		t.Fatal("Expected error when calling loadfile, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestSandboxDofileBlocked(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString("dofile('/tmp/test.lua')")
	if err == nil {
		t.Fatal("Expected error when calling dofile, but it succeeded")
	}

	expectedError := "attempt to call a non-function object"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}
