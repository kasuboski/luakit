package luavm

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestRequireFromBuildContext(t *testing.T) {
	tmpDir := t.TempDir()

	moduleContent := `
local M = {}
function M.hello()
	return "hello from module"
end
return M
`

	modulePath := filepath.Join(tmpDir, "test_module.lua")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write module: %v", err)
	}

	scriptContent := `
local m = require("test_module")
result = m.hello()
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTString {
		t.Errorf("Expected string result, got %v", result.Type())
	}

	if result.String() != "hello from module" {
		t.Errorf("Expected 'hello from module', got '%s'", result.String())
	}
}

func TestRequireFromStdlib(t *testing.T) {
	tmpDir := t.TempDir()
	stdlibDir := filepath.Join(tmpDir, "stdlib")

	if err := os.Mkdir(stdlibDir, 0755); err != nil {
		t.Fatalf("Failed to create stdlib dir: %v", err)
	}

	moduleContent := `
local M = {}
function M.from_stdlib()
	return "from stdlib"
end
return M
`

	modulePath := filepath.Join(stdlibDir, "stdlib_module.lua")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write stdlib module: %v", err)
	}

	scriptContent := `
local m = require("stdlib_module")
result = m.from_stdlib()
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
		StdlibDir:       stdlibDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTString {
		t.Errorf("Expected string result, got %v", result.Type())
	}

	if result.String() != "from stdlib" {
		t.Errorf("Expected 'from stdlib', got '%s'", result.String())
	}
}

func TestRequireBuildContextOverridesStdlib(t *testing.T) {
	tmpDir := t.TempDir()
	stdlibDir := filepath.Join(tmpDir, "stdlib")

	if err := os.Mkdir(stdlibDir, 0755); err != nil {
		t.Fatalf("Failed to create stdlib dir: %v", err)
	}

	stdlibContent := `
local M = {}
function M.identify()
	return "stdlib"
end
return M
`

	if err := os.WriteFile(filepath.Join(stdlibDir, "shared.lua"), []byte(stdlibContent), 0644); err != nil {
		t.Fatalf("Failed to write stdlib module: %v", err)
	}

	contextContent := `
local M = {}
function M.identify()
	return "context"
end
return M
`

	if err := os.WriteFile(filepath.Join(tmpDir, "shared.lua"), []byte(contextContent), 0644); err != nil {
		t.Fatalf("Failed to write context module: %v", err)
	}

	scriptContent := `
local m = require("shared")
result = m.identify()
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
		StdlibDir:       stdlibDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result")
	if result.String() != "context" {
		t.Errorf("Expected 'context' to override stdlib, got '%s'", result.String())
	}
}

func TestRequireWithLuaExtension(t *testing.T) {
	tmpDir := t.TempDir()

	moduleContent := `
return "loaded without .lua extension"
`

	modulePath := filepath.Join(tmpDir, "simple.lua")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write module: %v", err)
	}

	scriptContent := `
result = require("simple")
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result")
	if result.String() != "loaded without .lua extension" {
		t.Errorf("Expected module to be loaded without .lua extension, got '%s'", result.String())
	}
}

func TestRequireNonexistentModule(t *testing.T) {
	tmpDir := t.TempDir()

	scriptContent := `
local ok, err = pcall(function()
	require("nonexistent_module")
end)
result_error = ok
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result_error")
	if result.Type() != lua.LTBool || result.String() == "true" {
		t.Errorf("Expected require of nonexistent module to fail")
	}
}

func TestRequireWithBuildkitAPI(t *testing.T) {
	defer resetExportedState()

	tmpDir := t.TempDir()

	moduleContent := `
local M = {}
function M.build_base(image_ref)
	return bk.image(image_ref)
end
return M
`

	modulePath := filepath.Join(tmpDir, "helpers.lua")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write module: %v", err)
	}

	scriptContent := `
local helpers = require("helpers")
local state = helpers.build_base("alpine:3.19")
bk.export(state)
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
	})
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	if GetExportedState() == nil {
		t.Error("Expected state to be exported")
	}
}

func TestRequireModuleCaching(t *testing.T) {
	tmpDir := t.TempDir()

	moduleContent := `
local M = {}
M.call_count = 0
function M.increment()
	M.call_count = M.call_count + 1
	return M.call_count
end
return M
`

	modulePath := filepath.Join(tmpDir, "counter.lua")
	if err := os.WriteFile(modulePath, []byte(moduleContent), 0644); err != nil {
		t.Fatalf("Failed to write module: %v", err)
	}

	scriptContent := `
local m1 = require("counter")
local count1 = m1.increment()

local m2 = require("counter")
local count2 = m2.increment()

result = count2
`

	scriptPath := filepath.Join(tmpDir, "script.lua")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	L := NewVM(&VMConfig{
		BuildContextDir: tmpDir,
	})
	defer L.Close()

	if err := L.DoFile(scriptPath); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	result := L.GetGlobal("result")
	if result.String() != "2" {
		t.Errorf("Expected module to be cached (result should be 2), got %s", result.String())
	}
}
