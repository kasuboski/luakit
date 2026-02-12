package luavm

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestBkImageWithNilRef(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.image(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.image with nil")
	}
}

func TestBkImageWithNumberRef(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.image(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.image with number")
	}
}

func TestBkImageWithTableRef(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.image({})`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.image with table")
	}
}

func TestBkImageWhitespaceRef(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.image("   ")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.image with whitespace-only string")
	}
}

func TestBkLocalWithNilName(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.local_(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.local_ with nil")
	}
}

func TestBkLocalWithNumberName(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.local_(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.local_ with number")
	}
}

func TestBkLocalWhitespaceName(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.local_("   ")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.local_ with whitespace-only string")
	}
}

func TestBkScratchWithArgs(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.scratch("extra")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.scratch with arguments")
	}
}

func TestBkExportWithNilState(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.export(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.export with nil")
	}
}

func TestBkExportWithNonState(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.export("not a state")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.export with string")
	}
}

func TestBkExportWithNumber(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.export(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.export with number")
	}
}

func TestBkExportWithTable(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.export({})`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.export with table")
	}
}

func TestBkExportCalledTwice(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.export(s)
		bk.export(s)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.export twice")
	}

	if !strings.Contains(err.Error(), "already called once") {
		t.Errorf("Expected error about already called once, got: %v", err)
	}
}

func TestStateRunWithNilCommand(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run(nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling run with nil command")
	}
}

func TestStateRunWithNumberCommand(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run(123)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling run with number command")
	}
}

func TestStateRunWithTableCommand(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run({})
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling run with empty table command")
	}
}

func TestStateRunWithTableCommandNonStringElements(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run({"echo", 123})
	`
	err := L.DoString(script)
	if err != nil {
		t.Errorf("Did not expect error with mixed table command, got: %v", err)
	}
}

func TestStateRunWithNilOptions(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", nil)
	`
	err := L.DoString(script)
	if err != nil {
		t.Errorf("Expected success with nil options, got: %v", err)
	}
}

func TestStateRunWithInvalidEnvType(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", { env = "not a table" })
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when run options has invalid env type")
	}
}

func TestStateRunWithInvalidCwdType(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", { cwd = 123 })
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when run options has invalid cwd type")
	}
}

func TestStateRunWithInvalidUserType(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", { user = {} })
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when run options has invalid user type")
	}
}

func TestStateRunWithInvalidMountsType(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", { mounts = "not a table" })
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when run options has invalid mounts type")
	}
}

func TestStateRunWithNonMountInMounts(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:run("echo test", { mounts = { "not a mount" } })
	`
	err := L.DoString(script)
	if err != nil {
		t.Errorf("Did not expect error with non-mount in mounts, got: %v", err)
	}
}

func TestBkMergeWithNonStates(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.merge("not a state", "also not a state")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling merge with non-states")
	}
}

func TestBkMergeWithMixedTypes(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.merge(s, "not a state")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling merge with mixed types")
	}
}

func TestBkMergeWithNil(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.merge(s, nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling merge with nil")
	}
}

func TestBkDiffWithNonStates(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.diff("not a state", "also not a state")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling diff with non-states")
	}
}

func TestBkDiffWithNilLower(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.diff(nil, s)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling diff with nil lower")
	}
}

func TestBkDiffWithNilUpper(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.diff(s, nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling diff with nil upper")
	}
}

func TestStateCopyWithNilFromState(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:copy(nil, "/src", "/dst")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling copy with nil from state")
	}
}

func TestStateCopyWithNonStateFrom(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:copy("not a state", "/src", "/dst")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling copy with non-state from")
	}
}

func TestStateCopyWithNilSrc(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s1 = bk.image("alpine:3.19")
		local s2 = bk.image("ubuntu:24.04")
		s1:copy(s2, nil, "/dst")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling copy with nil src")
	}
}

func TestStateCopyWithNilDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s1 = bk.image("alpine:3.19")
		local s2 = bk.image("ubuntu:24.04")
		s1:copy(s2, "/src", nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling copy with nil dest")
	}
}

func TestStateMkdirWithNilPath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:mkdir(nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling mkdir with nil path")
	}
}

func TestStateMkdirWithEmptyPath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:mkdir("")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling mkdir with empty path")
	}
}

func TestStateMkfileWithNilPath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:mkfile(nil, "data")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling mkfile with nil path")
	}
}

func TestStateMkfileWithNilData(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:mkfile("/file", nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling mkfile with nil data")
	}
}

func TestStateRmWithNilPath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:rm(nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling rm with nil path")
	}
}

func TestStateRmWithEmptyPath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:rm("")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling rm with empty path")
	}
}

func TestStateSymlinkWithNilOldpath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:symlink(nil, "/new")
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling symlink with nil oldpath")
	}
}

func TestStateSymlinkWithNilNewpath(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:symlink("/old", nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling symlink with nil newpath")
	}
}

func TestBkCacheWithNilDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.cache(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.cache with nil dest")
	}
}

func TestBkCacheWithNumberDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.cache(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.cache with number dest")
	}
}

func TestBkSecretWithNilDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.secret(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.secret with nil dest")
	}
}

func TestBkSecretWithNumberDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.secret(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.secret with number dest")
	}
}

func TestBkTmpfsWithNilDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.tmpfs(nil)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.tmpfs with nil dest")
	}
}

func TestBkTmpfsWithNumberDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.tmpfs(123)`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.tmpfs with number dest")
	}
}

func TestBkBindWithNilState(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.bind(nil, "/dest")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.bind with nil state")
	}
}

func TestBkBindWithNonState(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `bk.bind("not a state", "/dest")`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.bind with non-state")
	}
}

func TestBkBindWithNilDest(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.bind(s, nil)
	`
	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling bk.bind with nil dest")
	}
}

func TestComplexMultiStageBuild(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local builder = bk.image("golang:1.22")
		local src = bk.local_("context")
		local deps = builder:run("go mod download", { cwd = "/app", mounts = { bk.cache("/go/pkg/mod") } })
		local workspace = deps:copy(src, ".", "/app")
		local built = workspace:run("go build -o /out/server ./cmd/server", { cwd = "/app" })
		local runtime = bk.image("gcr.io/distroless/static-debian12")
		local final = runtime:copy(built, "/out/server", "/server")
		bk.export(final, { entrypoint = {"/server"} })
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute multi-stage build script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	if state.Op() == nil {
		t.Fatal("Expected state to have an Op")
	}
}

func TestDAGWithMultipleBranches(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("node:20")
		local deps = base:run("npm ci", { cwd = "/app" })
		local lint = deps:run("npm run lint", { cwd = "/app" })
		local test = deps:run("npm run test", { cwd = "/app" })
		local build = deps:run("npm run build", { cwd = "/app" })
		local verified = bk.merge(lint, test, build)
		bk.export(verified)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute multi-branch script: %v", err)
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
}

func TestDAGWithDiffAndMerge(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local installed = base:run("apk add nginx")
		local just_nginx = bk.diff(base, installed)
		local alpine = bk.image("alpine:3.20")
		local final = bk.merge(alpine, just_nginx)
		bk.export(final)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute diff+merge script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	mergeOp := state.Op().Op().GetMerge()
	if mergeOp == nil {
		t.Fatal("Expected MergeOp")
	}

	if len(mergeOp.Inputs) != 2 {
		t.Errorf("Expected 2 merge inputs, got %d", len(mergeOp.Inputs))
	}

	diffOp := state.Op().Inputs()[1].Node().Op().GetDiff()
	if diffOp == nil {
		t.Fatal("Expected second input to be DiffOp")
	}
}

func TestFileOperationsChain(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local with_dir = base:mkdir("/app", { mode = 0755, make_parents = true })
		local with_file = with_dir:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
		local with_link = with_file:symlink("/app/config.json", "/app/config.link")
		bk.export(with_link)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute file operations chain: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	// Each file operation creates a new FileOp with 1 action
	if len(fileOp.Actions) != 1 {
		t.Errorf("Expected 1 file action, got %d", len(fileOp.Actions))
	}

	symlinkAction := fileOp.Actions[0].GetSymlink()
	if symlinkAction == nil {
		t.Fatal("Expected Symlink action")
	}

	if symlinkAction.Oldpath != "/app/config.json" {
		t.Errorf("Expected oldpath '/app/config.json', got '%s'", symlinkAction.Oldpath)
	}

	if symlinkAction.Newpath != "/app/config.link" {
		t.Errorf("Expected newpath '/app/config.link', got '%s'", symlinkAction.Newpath)
	}
}

func TestStateImmutability(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")
		local s1 = base:run("echo 1")
		local s2 = base:run("echo 2")
		local s3 = s1:run("echo 3")
		local s4 = s1:run("echo 4")

		if s1 == s2 then return false end
		if s1 == s3 then return false end
		if s3 == s4 then return false end
		return true
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute immutability test: %v", err)
	}

	result := L.GetGlobal("result")
	if result.Type() != lua.LTNil && !lua.LVIsFalse(result) {
		t.Error("Expected states to be immutable (different)")
	}
}

func TestMultipleExportsInDifferentScopes(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.export(s)
		local function test_func()
			bk.export(s)
		end
		test_func()
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when calling export in different scope")
	}

	if !strings.Contains(err.Error(), "already called once") {
		t.Errorf("Expected error about already called once, got: %v", err)
	}
}

func TestExportWithoutAnyOperations(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		bk.export(bk.scratch())
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to export scratch: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	if state.Op().Op().GetSource().Identifier != "scratch" {
		t.Error("Expected scratch source")
	}
}

func TestErrorPropagationFromOps(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		local result = s:run("echo test")
		if result == nil then
			error("run returned nil")
		end
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestLuaErrorHandling(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		error("intentional error")
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error from Lua script")
	}

	if !strings.Contains(err.Error(), "intentional error") {
		t.Errorf("Expected error containing 'intentional error', got: %v", err)
	}
}

func TestSyntaxErrorHandling(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19"
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected syntax error")
	}
}

func TestUnknownMethodOnState(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:unknown_method()
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error for unknown method")
	}

	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("Expected error about unknown field, got: %v", err)
	}
}

func TestUnknownFieldOnState(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		local f = s.unknown_field
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error for unknown field")
	}

	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("Expected error about unknown field, got: %v", err)
	}
}

func TestMountTypesPreserveDefaults(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local cache = bk.cache("/cache")
		local secret = bk.secret("/secret")
		local ssh = bk.ssh()
		local tmpfs = bk.tmpfs("/tmp")

		local s = bk.image("alpine:3.19")
		local result = s:run("echo test", {
			mounts = { cache, secret, ssh, tmpfs }
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script with default mounts: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Mounts) != 5 {
		t.Errorf("Expected 5 mounts, got %d", len(execOp.Mounts))
	}
}

func TestCopyWithOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			mode = "0755",
			follow_symlink = true,
			create_dest_path = true,
			allow_wildcard = true,
			include = {"*.go", "*.mod"},
			exclude = {"*.test", "vendor/"}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute copy with options: %v", err)
	}

	state := GetExportedState()
	fileOp := state.Op().Op().GetFile()

	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	copyAction := fileOp.Actions[0].GetCopy()
	if copyAction == nil {
		t.Fatal("Expected Copy action")
	}

	if len(copyAction.IncludePatterns) != 2 {
		t.Errorf("Expected 2 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}
}

func TestMkdirWithOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		local result = s:mkdir("/app/data", {
			mode = "0755",
			make_parents = true,
			owner = { user = "app", group = "app" }
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute mkdir with options: %v", err)
	}

	state := GetExportedState()
	fileOp := state.Op().Op().GetFile()

	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	mkdirAction := fileOp.Actions[0].GetMkdir()
	if mkdirAction == nil {
		t.Fatal("Expected Mkdir action")
	}

	if mkdirAction.MakeParents != true {
		t.Error("Expected MakeParents to be true")
	}
}

func TestRmWithOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		local result = s:rm("/tmp/*", {
			allow_not_found = true,
			allow_wildcard = true
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute rm with options: %v", err)
	}

	state := GetExportedState()
	fileOp := state.Op().Op().GetFile()

	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	rmAction := fileOp.Actions[0].GetRm()
	if rmAction == nil {
		t.Fatal("Expected Rm action")
	}

	if rmAction.AllowNotFound != true {
		t.Error("Expected AllowNotFound to be true")
	}

	if rmAction.AllowWildcard != true {
		t.Error("Expected AllowWildcard to be true")
	}
}

func TestDeeplyNestedDAG(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		for i = 1, 10 do
			s = s:run("echo " .. i)
		end
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute deeply nested DAG: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	node := state.Op()
	depth := 0
	for node != nil && len(node.Inputs()) > 0 {
		depth++
		node = node.Inputs()[0].Node()
	}

	if depth < 10 {
		t.Errorf("Expected DAG depth at least 10, got %d", depth)
	}
}

func TestConvergentDAG(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local s1 = base:run("echo 1")
		local s2 = base:run("echo 2")
		local s3 = s1:run("echo 3")
		local s4 = s2:run("echo 4")
		local merged = bk.merge(s3, s4)
		bk.export(merged)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute convergent DAG: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	mergeOp := state.Op().Op().GetMerge()
	if mergeOp == nil {
		t.Fatal("Expected MergeOp")
	}

	if len(mergeOp.Inputs) != 2 {
		t.Errorf("Expected 2 merge inputs, got %d", len(mergeOp.Inputs))
	}
}

func TestExportedImageConfig(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local s = bk.image("alpine:3.19")
		bk.export(s, {
			entrypoint = {"/bin/sh"},
			cmd = {"-c", "echo hello"},
			env = { PATH = "/usr/bin", FOO = "bar" },
			workdir = "/app",
			user = "appuser",
			expose = {"8080/tcp", "9090/udp"},
			labels = { ["org.opencontainers.image.title"] = "Test" }
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute export with config: %v", err)
	}

	config := GetExportedImageConfig()
	if config == nil {
		t.Fatal("Expected exported image config to be non-nil")
	}

	if len(config.Config.Entrypoint) != 1 {
		t.Errorf("Expected 1 entrypoint element, got %d", len(config.Config.Entrypoint))
	}

	if len(config.Config.Cmd) != 2 {
		t.Errorf("Expected 2 cmd elements, got %d", len(config.Config.Cmd))
	}

	if len(config.Config.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(config.Config.Env))
	}

	if config.Config.WorkingDir != "/app" {
		t.Errorf("Expected workdir '/app', got '%s'", config.Config.WorkingDir)
	}

	if config.Config.User != "appuser" {
		t.Errorf("Expected user 'appuser', got '%s'", config.Config.User)
	}

	if len(config.Config.ExposedPorts) != 2 {
		t.Errorf("Expected 2 exposed ports, got %d", len(config.Config.ExposedPorts))
	}

	if len(config.Config.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(config.Config.Labels))
	}
}
