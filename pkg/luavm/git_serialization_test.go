package luavm

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestGitSerialization(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			ref = "v0.12.0",
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

	opNode := state.Op()
	if opNode == nil {
		t.Fatal("Expected OpNode")
	}

	op := opNode.Op()
	if op == nil {
		t.Fatal("Expected pb.Op")
	}

	sourceOp := op.GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#v0.12.0"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}

	if sourceOp.Attrs == nil {
		t.Fatal("Expected non-nil Attrs")
	}

	if sourceOp.Attrs["keepgitdir"] != "true" {
		t.Errorf("Expected keepgitdir attribute 'true', got '%s'", sourceOp.Attrs["keepgitdir"])
	}

	if op.Platform != nil {
		t.Error("Expected nil Platform for git source")
	}

	if op.Op == nil {
		t.Fatal("Expected Op field to be set")
	}

	if _, ok := op.Op.(*pb.Op_Source); !ok {
		t.Error("Expected Op to be a SourceOp")
	}
}

func TestGitWithoutRefSerialization(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

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

func TestGitMultiStageWithGitSource(t *testing.T) {
	resetExportedState()
	t.Cleanup(resetExportedState)

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local repo = bk.git("https://github.com/moby/buildkit.git", {
			ref = "main"
		})

		local builder = bk.image("golang:1.22")
		local workspace = builder:copy(repo, ".", "/src")

		bk.export(workspace)
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
		t.Fatal("Expected FileOp in final state")
	}

	if len(state.Op().Inputs()) < 1 {
		t.Fatal("Expected at least 1 input to FileOp")
	}

	gitState := state.Op().Inputs()[0].Node()
	gitSourceOp := gitState.Op().GetSource()
	if gitSourceOp == nil {
		t.Fatal("Expected SourceOp as first input to FileOp")
	}

	if len(state.Op().Inputs()) < 2 {
		t.Fatal("Expected at least 2 inputs to FileOp")
	}

	imageState := state.Op().Inputs()[1].Node()
	imageSourceOp := imageState.Op().GetSource()
	if imageSourceOp == nil {
		t.Fatal("Expected SourceOp as second input to FileOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#main"
	if gitSourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, gitSourceOp.Identifier)
	}
}
