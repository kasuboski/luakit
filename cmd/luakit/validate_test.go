package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateValidScript(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/valid.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
`
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateMissingExport(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/no_export.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err == nil {
		t.Error("expected error about missing export, got nil")
	}
	if !strings.Contains(err.Error(), "no bk.export() call") {
		t.Errorf("expected error about missing export, got: %v", err)
	}
}

func TestValidateSyntaxError(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/syntax.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello"
bk.export(result)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err == nil {
		t.Error("expected error message, got nil")
	}
}

func TestValidateInvalidAPIArgs(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/invalid_args.lua"

	script := `local base = bk.image("")
local result = base:run("echo hello")
bk.export(result)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err == nil {
		t.Error("expected error message, got nil")
	}
	if !strings.Contains(err.Error(), "identifier must not be empty") {
		t.Errorf("expected error about empty identifier, got: %v", err)
	}
}

func TestValidateMergeOperation(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/merge.lua"

	script := `local base = bk.image("alpine:3.19")
local deps = base:run("apk add git")
local source = base:run("mkdir -p /app")
local merged = bk.merge(deps, source)
bk.export(merged)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateDiffOperation(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/diff.lua"

	script := `local base = bk.image("alpine:3.19")
local installed = base:run("apk add git")
local just_git = bk.diff(base, installed)
local clean = bk.image("alpine:3.19")
local with_git = bk.merge(clean, just_git)
bk.export(with_git)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateFileOperations(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/file_ops.lua"

	script := `local base = bk.image("alpine:3.19")
local src = bk.local_("context")
local workspace = base:copy(src, ".", "/app")
local result = workspace:run("echo hello > /greeting.txt")
bk.export(result)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateWithMounts(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/mounts.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello", {
    mounts = {
        bk.cache("/root/.cache"),
        bk.tmpfs("/tmp"),
    }
})
bk.export(result)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateComplexMultiStage(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/multi_stage.lua"

	script := `local builder = bk.image("golang:1.22")
local src = bk.local_("context")

local deps = builder:copy(src, "go.mod", "/app/go.mod"):copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app" })

local workspace = downloaded:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", {
    cwd = "/app",
    mounts = { bk.cache("/root/.cache/go-build") },
})

local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateScriptNotFound(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", "/nonexistent/script.lua"}

	err := validateScript()
	if err == nil {
		t.Error("expected error message, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read script") {
		t.Errorf("expected error about missing file, got: %v", err)
	}
}

func TestValidateNoScriptArg(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate"}

	err := validateScript()
	if err == nil {
		t.Error("expected error about missing script, got nil")
	}
	if !strings.Contains(err.Error(), "missing script file") {
		t.Errorf("expected error about missing script, got: %v", err)
	}
}

func TestValidateWithImageConfig(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/image_config.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "echo hello"},
    env = {"PATH=/usr/local/bin:/usr/bin:/bin"},
    workdir = "/app",
    user = "app",
    labels = { version = "1.0.0" },
    expose = {"8080/tcp"},
})
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateInvalidMergeArgs(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/invalid_merge.lua"

	script := `local base = bk.image("alpine:3.19")
local merged = bk.merge(base)
bk.export(merged)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err == nil {
		t.Error("expected error message, got nil")
	}
}

func TestValidateInvalidDiffArgs(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/invalid_diff.lua"

	script := `local base = bk.image("alpine:3.19")
local diff = bk.diff(base)
bk.export(diff)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err == nil {
		t.Error("expected error message, got nil")
	}
}

func TestValidateAllOperationTypes(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/all_ops.lua"

	script := `local base = bk.image("alpine:3.19")
local ctx = bk.local_("context")

local scratch = bk.scratch()

local with_dir = scratch:mkdir("/app", { mode = 0755 })
local with_file = with_dir:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local with_link = with_file:symlink("/app/config.json", "/etc/config")

local merged = bk.merge(base, ctx)

local installed = base:run("apk add git")
local diff = bk.diff(base, installed)

local final = merged:copy(ctx, ".", "/app")
bk.export(final)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSerializationErrors(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := tmpDir + "/serialize_test.lua"

	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"luakit", "validate", scriptPath}

	err := validateScript()
	if err != nil {
		t.Errorf("expected no error (should serialize successfully), got: %v", err)
	}
}
