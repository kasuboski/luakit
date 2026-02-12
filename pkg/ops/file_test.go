package ops

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNewFileOp(t *testing.T) {
	actions := []*pb.FileAction{
		{
			Action: &pb.FileAction_Mkdir{
				Mkdir: &pb.FileActionMkDir{
					Path: "/test",
				},
			},
		},
	}

	op := NewFileOp(actions)

	if len(op.Actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(op.Actions))
	}

	if op.Actions[0].GetMkdir().Path != "/test" {
		t.Errorf("Expected path '/test', got '%s'", op.Actions[0].GetMkdir().Path)
	}
}

func TestNewFileState(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	action := &pb.FileAction{
		Action: &pb.FileAction_Mkdir{
			Mkdir: &pb.FileActionMkDir{
				Path: "/test",
			},
		},
	}

	fileOp := NewFileOp([]*pb.FileAction{action})
	fileState := NewFileState(sourceState, fileOp, "test.lua", 20)

	if fileState == nil {
		t.Fatal("Expected non-nil file state")
	}

	pbOp := fileState.Op().Op()
	if len(pbOp.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(pbOp.Inputs))
	}

	if pbOp.Inputs[0].Index != 0 {
		t.Errorf("Expected input index 0, got %d", pbOp.Inputs[0].Index)
	}

	if len(fileState.Op().Inputs()) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(fileState.Op().Inputs()))
	}
}

func TestMkdir(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	opts := &MkdirOptions{
		Mode:        0755,
		MakeParents: true,
	}

	result := Mkdir(sourceState, "/test/dir", opts, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	mkdirAction := result.Op().Op().GetFile().Actions[0].GetMkdir()
	if mkdirAction == nil {
		t.Fatal("Expected Mkdir action")
	}

	if mkdirAction.Path != "/test/dir" {
		t.Errorf("Expected path '/test/dir', got '%s'", mkdirAction.Path)
	}

	if mkdirAction.Mode != 0755 {
		t.Errorf("Expected mode 0755, got %d", mkdirAction.Mode)
	}

	if !mkdirAction.MakeParents {
		t.Error("Expected MakeParents to be true")
	}

	if result.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", result.Op().LuaFile())
	}

	if result.Op().LuaLine() != 20 {
		t.Errorf("Expected Lua line 20, got %d", result.Op().LuaLine())
	}
}

func TestMkdirWithNoOptions(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	result := Mkdir(sourceState, "/test", nil, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	mkdirAction := result.Op().Op().GetFile().Actions[0].GetMkdir()
	if mkdirAction.Path != "/test" {
		t.Errorf("Expected path '/test', got '%s'", mkdirAction.Path)
	}
}

func TestMkdirWithEmptyPath(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	result := Mkdir(sourceState, "", nil, "test.lua", 20)

	if result != nil {
		t.Error("Expected nil state for empty path")
	}
}

func TestMkfile(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	opts := &MkfileOptions{
		Mode: 0644,
	}

	data := "hello world"

	result := Mkfile(sourceState, "/test/file.txt", data, opts, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	mkfileAction := result.Op().Op().GetFile().Actions[0].GetMkfile()
	if mkfileAction == nil {
		t.Fatal("Expected Mkfile action")
	}

	if mkfileAction.Path != "/test/file.txt" {
		t.Errorf("Expected path '/test/file.txt', got '%s'", mkfileAction.Path)
	}

	if string(mkfileAction.Data) != data {
		t.Errorf("Expected data '%s', got '%s'", data, string(mkfileAction.Data))
	}

	if mkfileAction.Mode != 0644 {
		t.Errorf("Expected mode 0644, got %d", mkfileAction.Mode)
	}
}

func TestMkfileWithNoOptions(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	result := Mkfile(sourceState, "/test.txt", "content", nil, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	mkfileAction := result.Op().Op().GetFile().Actions[0].GetMkfile()
	if mkfileAction.Path != "/test.txt" {
		t.Errorf("Expected path '/test.txt', got '%s'", mkfileAction.Path)
	}
}

func TestRm(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	opts := &RmOptions{
		AllowNotFound: true,
		AllowWildcard: true,
	}

	result := Rm(sourceState, "/test/file", opts, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	rmAction := result.Op().Op().GetFile().Actions[0].GetRm()
	if rmAction == nil {
		t.Fatal("Expected Rm action")
	}

	if rmAction.Path != "/test/file" {
		t.Errorf("Expected path '/test/file', got '%s'", rmAction.Path)
	}

	if !rmAction.AllowNotFound {
		t.Error("Expected AllowNotFound to be true")
	}

	if !rmAction.AllowWildcard {
		t.Error("Expected AllowWildcard to be true")
	}
}

func TestRmWithNoOptions(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	result := Rm(sourceState, "/test/file", nil, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	rmAction := result.Op().Op().GetFile().Actions[0].GetRm()
	if rmAction.Path != "/test/file" {
		t.Errorf("Expected path '/test/file', got '%s'", rmAction.Path)
	}
}

func TestSymlink(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	result := Symlink(sourceState, "/old/path", "/new/path", "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	symlinkAction := result.Op().Op().GetFile().Actions[0].GetSymlink()
	if symlinkAction == nil {
		t.Fatal("Expected Symlink action")
	}

	if symlinkAction.Oldpath != "/old/path" {
		t.Errorf("Expected oldpath '/old/path', got '%s'", symlinkAction.Oldpath)
	}

	if symlinkAction.Newpath != "/new/path" {
		t.Errorf("Expected newpath '/new/path', got '%s'", symlinkAction.Newpath)
	}
}

func TestCopy(t *testing.T) {
	fromOp := NewSourceOp("scratch", nil)
	fromState := NewSourceState(fromOp, "test.lua", 10)

	toOp := NewSourceOp("scratch", nil)
	toState := NewSourceState(toOp, "test.lua", 15)

	opts := &CopyOptions{
		Mode:            0755,
		FollowSymlink:   true,
		CreateDestPath:  true,
		AllowWildcard:   true,
		IncludePatterns: []string{"*.go"},
		ExcludePatterns: []string{"*.test"},
	}

	result := Copy(toState, fromState, "/src", "/dst", opts, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	copyAction := result.Op().Op().GetFile().Actions[0].GetCopy()
	if copyAction == nil {
		t.Fatal("Expected Copy action")
	}

	if copyAction.Src != "/src" {
		t.Errorf("Expected src '/src', got '%s'", copyAction.Src)
	}

	if copyAction.Dest != "/dst" {
		t.Errorf("Expected dest '/dst', got '%s'", copyAction.Dest)
	}

	if copyAction.Mode != 0755 {
		t.Errorf("Expected mode 0755, got %d", copyAction.Mode)
	}

	if !copyAction.FollowSymlink {
		t.Error("Expected FollowSymlink to be true")
	}

	if !copyAction.CreateDestPath {
		t.Error("Expected CreateDestPath to be true")
	}

	if !copyAction.AllowWildcard {
		t.Error("Expected AllowWildcard to be true")
	}

	if len(copyAction.IncludePatterns) != 1 {
		t.Errorf("Expected 1 include pattern, got %d", len(copyAction.IncludePatterns))
	}

	if copyAction.IncludePatterns[0] != "*.go" {
		t.Errorf("Expected include pattern '*.go', got '%s'", copyAction.IncludePatterns[0])
	}

	if len(copyAction.ExcludePatterns) != 1 {
		t.Errorf("Expected 1 exclude pattern, got %d", len(copyAction.ExcludePatterns))
	}

	if copyAction.ExcludePatterns[0] != "*.test" {
		t.Errorf("Expected exclude pattern '*.test', got '%s'", copyAction.ExcludePatterns[0])
	}

	if len(result.Op().Inputs()) != 2 {
		t.Errorf("Expected 2 inputs for copy, got %d", len(result.Op().Inputs()))
	}
}

func TestCopyWithNoOptions(t *testing.T) {
	fromOp := NewSourceOp("scratch", nil)
	fromState := NewSourceState(fromOp, "test.lua", 10)

	toOp := NewSourceOp("scratch", nil)
	toState := NewSourceState(toOp, "test.lua", 15)

	result := Copy(toState, fromState, "/src", "/dst", nil, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	copyAction := result.Op().Op().GetFile().Actions[0].GetCopy()
	if copyAction.Src != "/src" {
		t.Errorf("Expected src '/src', got '%s'", copyAction.Src)
	}

	if copyAction.Dest != "/dst" {
		t.Errorf("Expected dest '/dst', got '%s'", copyAction.Dest)
	}
}

func TestCopyWithChownOpt(t *testing.T) {
	fromOp := NewSourceOp("scratch", nil)
	fromState := NewSourceState(fromOp, "test.lua", 10)

	toOp := NewSourceOp("scratch", nil)
	toState := NewSourceState(toOp, "test.lua", 15)

	opts := &CopyOptions{
		Owner: &ChownOpt{
			User: &UserOpt{
				Name: "root",
			},
			Group: &UserOpt{
				ID: 1000,
			},
		},
	}

	result := Copy(toState, fromState, "/src", "/dst", opts, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil state")
	}

	copyAction := result.Op().Op().GetFile().Actions[0].GetCopy()
	if copyAction.Owner == nil {
		t.Fatal("Expected Owner to be set")
	}

	if copyAction.Owner.User == nil {
		t.Fatal("Expected User to be set in Owner")
	}

	if copyAction.Owner.User.GetByName().Name != "root" {
		t.Errorf("Expected user name 'root', got '%s'", copyAction.Owner.User.GetByName().Name)
	}

	if copyAction.Owner.Group == nil {
		t.Fatal("Expected Group to be set in Owner")
	}

	if copyAction.Owner.Group.GetByID() != 1000 {
		t.Errorf("Expected group ID 1000, got %d", copyAction.Owner.Group.GetByID())
	}
}

func TestBuildChownOpt(t *testing.T) {
	t.Run("with user name", func(t *testing.T) {
		opt := &ChownOpt{
			User: &UserOpt{
				Name: "root",
			},
		}

		result := buildChownOpt(opt)
		if result == nil {
			t.Fatal("Expected non-nil chown opt")
		}

		if result.User.GetByName().Name != "root" {
			t.Errorf("Expected user name 'root', got '%s'", result.User.GetByName().Name)
		}
	})

	t.Run("with user ID", func(t *testing.T) {
		opt := &ChownOpt{
			User: &UserOpt{
				ID: 0,
			},
		}

		result := buildChownOpt(opt)
		if result == nil {
			t.Fatal("Expected non-nil chown opt")
		}

		if result.User.GetByID() != 0 {
			t.Errorf("Expected user ID 0, got %d", result.User.GetByID())
		}
	})

	t.Run("with group name", func(t *testing.T) {
		opt := &ChownOpt{
			Group: &UserOpt{
				Name: "wheel",
			},
		}

		result := buildChownOpt(opt)
		if result == nil {
			t.Fatal("Expected non-nil chown opt")
		}

		if result.Group.GetByName().Name != "wheel" {
			t.Errorf("Expected group name 'wheel', got '%s'", result.Group.GetByName().Name)
		}
	})

	t.Run("with group ID", func(t *testing.T) {
		opt := &ChownOpt{
			Group: &UserOpt{
				ID: 1000,
			},
		}

		result := buildChownOpt(opt)
		if result == nil {
			t.Fatal("Expected non-nil chown opt")
		}

		if result.Group.GetByID() != 1000 {
			t.Errorf("Expected group ID 1000, got %d", result.Group.GetByID())
		}
	})

	t.Run("with both user and group", func(t *testing.T) {
		opt := &ChownOpt{
			User: &UserOpt{
				Name: "appuser",
			},
			Group: &UserOpt{
				ID: 1000,
			},
		}

		result := buildChownOpt(opt)
		if result == nil {
			t.Fatal("Expected non-nil chown opt")
		}

		if result.User.GetByName().Name != "appuser" {
			t.Errorf("Expected user name 'appuser', got '%s'", result.User.GetByName().Name)
		}

		if result.Group.GetByID() != 1000 {
			t.Errorf("Expected group ID 1000, got %d", result.Group.GetByID())
		}
	})

	t.Run("with nil opt", func(t *testing.T) {
		result := buildChownOpt(nil)
		if result != nil {
			t.Error("Expected nil chown opt for nil input")
		}
	})
}
