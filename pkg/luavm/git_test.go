package luavm

import (
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	lua "github.com/yuin/gopher-lua"
)

func TestBkGit(t *testing.T) {
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "git"))
	L.Remove(-2)

	L.Push(lua.LString("https://github.com/moby/buildkit.git"))

	if err := L.PCall(1, 1, nil); err != nil {
		t.Fatalf("Failed to call bk.git: %v", err)
	}

	s := L.Get(-1)
	if s.Type() != lua.LTUserData {
		t.Errorf("Expected userdata, got %v", s.Type())
	}

	L.Pop(1)
}

func TestBkGitEmptyString(t *testing.T) {
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	L.Push(L.GetGlobal("bk"))
	L.Push(L.GetField(L.Get(-1), "git"))
	L.Remove(-2)

	L.Push(lua.LString(""))

	err := L.PCall(0, 0, nil)
	if err == nil {
		t.Error("Expected error when calling bk.git with empty string")
	}
}

func TestBkGitWhitespaceOnly(t *testing.T) {
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	if err := L.DoString(`bk.git("   ")`); err == nil {
		t.Error("Expected error when calling bk.git with whitespace-only string")
	}
}

func TestBkGitWithRef(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			ref = "v0.12.0"
		})
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#v0.12.0"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestBkGitWithKeepGitDir(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			keep_git_dir = true
		})
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Attrs["keepgitdir"] != "true" {
		t.Errorf("Expected keepgitdir attribute 'true', got '%s'", sourceOp.Attrs["keepgitdir"])
	}
}

func TestBkGitWithBothOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			ref = "main",
			keep_git_dir = true
		})
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#main"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}

	if sourceOp.Attrs["keepgitdir"] != "true" {
		t.Errorf("Expected keepgitdir attribute 'true', got '%s'", sourceOp.Attrs["keepgitdir"])
	}
}

func TestBkGitWithBranchRef(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://gitlab.com/group/project.git", {
			ref = "develop"
		})
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://gitlab.com/group/project.git#develop"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestBkGitWithCommitRef(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			ref = "abc123def456"
		})
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#abc123def456"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestBkGitWithoutOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git")
		bk.export(repo)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}

	if _, hasKeepGitDir := sourceOp.Attrs["keepgitdir"]; hasKeepGitDir {
		t.Error("Expected keepgitdir attribute to not be present")
	}
}

func TestBkGitReturnsState(t *testing.T) {
	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	if err := L.DoString(`result = bk.git("https://github.com/moby/buildkit.git")`); err != nil {
		t.Fatalf("Failed to call bk.git: %v", err)
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
	expectedIdentifier := "git://https://github.com/moby/buildkit.git"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}
