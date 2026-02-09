package ops

import (
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
)

func TestImageValidation(t *testing.T) {
	testCases := []struct {
		name        string
		ref         string
		shouldError bool
	}{
		{
			name:        "empty ref",
			ref:         "",
			shouldError: true,
		},
		{
			name:        "valid ref",
			ref:         "alpine:3.19",
			shouldError: false,
		},
		{
			name:        "full ref",
			ref:         "docker.io/library/alpine:3.19",
			shouldError: false,
		},
		{
			name:        "ref with digest",
			ref:         "alpine@sha256:9c6e07cfb197bf3fa5ba994ffecdf2588f0fef2a0945d96f9ba6373814d3a2e2",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := Image(tc.ref, "test.lua", 1, nil)
			if tc.shouldError {
				if state != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if state == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestLocalValidation(t *testing.T) {
	testCases := []struct {
		name        string
		nameStr     string
		shouldError bool
	}{
		{
			name:        "empty name",
			nameStr:     "",
			shouldError: true,
		},
		{
			name:        "valid name",
			nameStr:     "context",
			shouldError: false,
		},
		{
			name:        "name with path separator",
			nameStr:     "subdir/context",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state := Local(tc.nameStr, "test.lua", 1, nil)
			if tc.shouldError {
				if state != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if state == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestRunValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		cmd         []string
		shouldError bool
	}{
		{
			name:        "nil cmd",
			cmd:         nil,
			shouldError: true,
		},
		{
			name:        "empty cmd",
			cmd:         []string{},
			shouldError: true,
		},
		{
			name:        "valid cmd",
			cmd:         []string{"echo", "hello"},
			shouldError: false,
		},
		{
			name:        "sh command",
			cmd:         []string{"/bin/sh", "-c", "echo hello"},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Run(state, tc.cmd, nil, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestCopyValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		src         string
		dest        string
		shouldError bool
	}{
		{
			name:        "empty src",
			src:         "",
			dest:        "/dest",
			shouldError: true,
		},
		{
			name:        "empty dest",
			src:         "/src",
			dest:        "",
			shouldError: true,
		},
		{
			name:        "valid copy",
			src:         "/src",
			dest:        "/dest",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Copy(state, state, tc.src, tc.dest, nil, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestMkdirValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		path        string
		shouldError bool
	}{
		{
			name:        "empty path",
			path:        "",
			shouldError: true,
		},
		{
			name:        "valid path",
			path:        "/app",
			shouldError: false,
		},
		{
			name:        "nested path",
			path:        "/app/subdir",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mkdir(state, tc.path, nil, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestMkfileValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		path        string
		data        string
		shouldError bool
	}{
		{
			name:        "empty path",
			path:        "",
			data:        "data",
			shouldError: true,
		},
		{
			name:        "empty data",
			path:        "/file",
			data:        "",
			shouldError: false,
		},
		{
			name:        "valid mkfile",
			path:        "/file",
			data:        "content",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mkfile(state, tc.path, tc.data, nil, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestRmValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		path        string
		shouldError bool
	}{
		{
			name:        "empty path",
			path:        "",
			shouldError: true,
		},
		{
			name:        "valid path",
			path:        "/tmp",
			shouldError: false,
		},
		{
			name:        "wildcard path",
			path:        "/tmp/*",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Rm(state, tc.path, nil, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestSymlinkValidation(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name        string
		oldpath     string
		newpath     string
		shouldError bool
	}{
		{
			name:        "empty oldpath",
			oldpath:     "",
			newpath:     "/new",
			shouldError: true,
		},
		{
			name:        "empty newpath",
			oldpath:     "/old",
			newpath:     "",
			shouldError: true,
		},
		{
			name:        "valid symlink",
			oldpath:     "/old",
			newpath:     "/new",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Symlink(state, tc.oldpath, tc.newpath, "test.lua", 2)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestMergeValidation(t *testing.T) {
	s1 := Image("alpine:3.19", "test.lua", 1, nil)
	s2 := Image("ubuntu:24.04", "test.lua", 2, nil)

	testCases := []struct {
		name        string
		states      []*dag.State
		shouldError bool
	}{
		{
			name:        "no states",
			states:      []*dag.State{},
			shouldError: true,
		},
		{
			name:        "one state",
			states:      []*dag.State{s1},
			shouldError: true,
		},
		{
			name:        "two states",
			states:      []*dag.State{s1, s2},
			shouldError: false,
		},
		{
			name:        "many states",
			states:      []*dag.State{s1, s2, s1},
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Merge(tc.states, "test.lua", 10)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestDiffValidation(t *testing.T) {
	s1 := Image("alpine:3.19", "test.lua", 1, nil)
	s2 := Run(s1, []string{"echo"}, nil, "test.lua", 2)

	testCases := []struct {
		name        string
		lower       *dag.State
		upper       *dag.State
		shouldError bool
	}{
		{
			name:        "nil lower",
			lower:       nil,
			upper:       s2,
			shouldError: true,
		},
		{
			name:        "nil upper",
			lower:       s1,
			upper:       nil,
			shouldError: true,
		},
		{
			name:        "valid diff",
			lower:       s1,
			upper:       s2,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Diff(tc.lower, tc.upper, "test.lua", 10)
			if tc.shouldError {
				if result != nil {
					t.Errorf("Expected nil state, got non-nil")
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil state, got nil")
				}
			}
		})
	}
}

func TestExecOptions(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name   string
		opts   *ExecOptions
		verify func(*testing.T, *pb.ExecOp)
	}{
		{
			name: "nil options",
			opts: nil,
			verify: func(t *testing.T, execOp *pb.ExecOp) {
				if execOp.Meta.Cwd != "" {
					t.Error("Expected empty cwd")
				}
				if execOp.Meta.User != "" {
					t.Error("Expected empty user")
				}
			},
		},
		{
			name: "env options",
			opts: &ExecOptions{
				Env: []string{"PATH=/usr/bin", "HOME=/root"},
			},
			verify: func(t *testing.T, execOp *pb.ExecOp) {
				if len(execOp.Meta.Env) != 2 {
					t.Errorf("Expected 2 env vars, got %d", len(execOp.Meta.Env))
				}
			},
		},
		{
			name: "cwd option",
			opts: &ExecOptions{
				Cwd: "/app",
			},
			verify: func(t *testing.T, execOp *pb.ExecOp) {
				if execOp.Meta.Cwd != "/app" {
					t.Errorf("Expected cwd '/app', got '%s'", execOp.Meta.Cwd)
				}
			},
		},
		{
			name: "user option",
			opts: &ExecOptions{
				User: "nobody",
			},
			verify: func(t *testing.T, execOp *pb.ExecOp) {
				if execOp.Meta.User != "nobody" {
					t.Errorf("Expected user 'nobody', got '%s'", execOp.Meta.User)
				}
			},
		},
		{
			name: "all options",
			opts: &ExecOptions{
				Env:    []string{"KEY=value"},
				Cwd:    "/workspace",
				User:   "builder",
				Mounts: []*Mount{},
			},
			verify: func(t *testing.T, execOp *pb.ExecOp) {
				if execOp.Meta.Cwd != "/workspace" {
					t.Errorf("Expected cwd '/workspace'")
				}
				if execOp.Meta.User != "builder" {
					t.Errorf("Expected user 'builder'")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Run(state, []string{"echo"}, tc.opts, "test.lua", 2)
			if result == nil {
				t.Fatal("Expected non-nil state")
			}

			execOp := result.Op().Op().GetExec()
			if execOp == nil {
				t.Fatal("Expected ExecOp")
			}

			if tc.verify != nil {
				tc.verify(t, execOp)
			}
		})
	}
}

func TestCopyOptions(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name   string
		opts   *CopyOptions
		verify func(*testing.T, *pb.FileActionCopy)
	}{
		{
			name: "nil options",
			opts: nil,
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.Mode != 0 {
					t.Error("Expected mode 0")
				}
				if copyAction.FollowSymlink != false {
					t.Error("Expected FollowSymlink false")
				}
			},
		},
		{
			name: "mode option",
			opts: &CopyOptions{
				Mode: 0755,
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.Mode != 0755 {
					t.Errorf("Expected mode 0755, got %d", copyAction.Mode)
				}
			},
		},
		{
			name: "follow symlink",
			opts: &CopyOptions{
				FollowSymlink: true,
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.FollowSymlink != true {
					t.Error("Expected FollowSymlink true")
				}
			},
		},
		{
			name: "create dest path",
			opts: &CopyOptions{
				CreateDestPath: true,
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.CreateDestPath != true {
					t.Error("Expected CreateDestPath true")
				}
			},
		},
		{
			name: "allow wildcard",
			opts: &CopyOptions{
				AllowWildcard: true,
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.AllowWildcard != true {
					t.Error("Expected AllowWildcard true")
				}
			},
		},
		{
			name: "include patterns",
			opts: &CopyOptions{
				IncludePatterns: []string{"*.go", "*.mod"},
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if len(copyAction.IncludePatterns) != 2 {
					t.Errorf("Expected 2 include patterns, got %d", len(copyAction.IncludePatterns))
				}
			},
		},
		{
			name: "exclude patterns",
			opts: &CopyOptions{
				ExcludePatterns: []string{"vendor", "*.test"},
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if len(copyAction.ExcludePatterns) != 2 {
					t.Errorf("Expected 2 exclude patterns, got %d", len(copyAction.ExcludePatterns))
				}
			},
		},
		{
			name: "owner with user and group",
			opts: &CopyOptions{
				Owner: &ChownOpt{
					User:  &UserOpt{Name: "appuser"},
					Group: &UserOpt{Name: "appgroup"},
				},
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.Owner == nil {
					t.Error("Expected owner to be set")
				}
			},
		},
		{
			name: "owner with numeric user and group",
			opts: &CopyOptions{
				Owner: &ChownOpt{
					User:  &UserOpt{ID: 1000},
					Group: &UserOpt{ID: 1000},
				},
			},
			verify: func(t *testing.T, copyAction *pb.FileActionCopy) {
				if copyAction.Owner == nil {
					t.Error("Expected owner to be set")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Copy(state, state, "/src", "/dest", tc.opts, "test.lua", 2)
			if result == nil {
				t.Fatal("Expected non-nil state")
			}

			fileOp := result.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			copyAction := fileOp.Actions[0].GetCopy()
			if copyAction == nil {
				t.Fatal("Expected Copy action")
			}

			if tc.verify != nil {
				tc.verify(t, copyAction)
			}
		})
	}
}

func TestMkdirOptions(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name   string
		opts   *MkdirOptions
		verify func(*testing.T, *pb.FileActionMkDir)
	}{
		{
			name: "nil options",
			opts: nil,
			verify: func(t *testing.T, mkdirAction *pb.FileActionMkDir) {
				if mkdirAction.Mode != 0 {
					t.Error("Expected mode 0")
				}
				if mkdirAction.MakeParents != false {
					t.Error("Expected MakeParents false")
				}
			},
		},
		{
			name: "mode option",
			opts: &MkdirOptions{
				Mode: 0755,
			},
			verify: func(t *testing.T, mkdirAction *pb.FileActionMkDir) {
				if mkdirAction.Mode != 0755 {
					t.Errorf("Expected mode 0755, got %d", mkdirAction.Mode)
				}
			},
		},
		{
			name: "make parents",
			opts: &MkdirOptions{
				MakeParents: true,
			},
			verify: func(t *testing.T, mkdirAction *pb.FileActionMkDir) {
				if mkdirAction.MakeParents != true {
					t.Error("Expected MakeParents true")
				}
			},
		},
		{
			name: "owner option",
			opts: &MkdirOptions{
				Owner: &ChownOpt{
					User:  &UserOpt{Name: "root"},
					Group: &UserOpt{ID: 0},
				},
			},
			verify: func(t *testing.T, mkdirAction *pb.FileActionMkDir) {
				if mkdirAction.Owner == nil {
					t.Error("Expected owner to be set")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mkdir(state, "/app", tc.opts, "test.lua", 2)
			if result == nil {
				t.Fatal("Expected non-nil state")
			}

			fileOp := result.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			mkdirAction := fileOp.Actions[0].GetMkdir()
			if mkdirAction == nil {
				t.Fatal("Expected Mkdir action")
			}

			if tc.verify != nil {
				tc.verify(t, mkdirAction)
			}
		})
	}
}

func TestMkfileOptions(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name   string
		opts   *MkfileOptions
		verify func(*testing.T, *pb.FileActionMkFile)
	}{
		{
			name: "nil options",
			opts: nil,
			verify: func(t *testing.T, mkfileAction *pb.FileActionMkFile) {
				if mkfileAction.Mode != 0 {
					t.Error("Expected mode 0")
				}
			},
		},
		{
			name: "mode option",
			opts: &MkfileOptions{
				Mode: 0644,
			},
			verify: func(t *testing.T, mkfileAction *pb.FileActionMkFile) {
				if mkfileAction.Mode != 0644 {
					t.Errorf("Expected mode 0644, got %d", mkfileAction.Mode)
				}
			},
		},
		{
			name: "owner option",
			opts: &MkfileOptions{
				Owner: &ChownOpt{
					User:  &UserOpt{Name: "root"},
					Group: &UserOpt{ID: 0},
				},
			},
			verify: func(t *testing.T, mkfileAction *pb.FileActionMkFile) {
				if mkfileAction.Owner == nil {
					t.Error("Expected owner to be set")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Mkfile(state, "/file", "data", tc.opts, "test.lua", 2)
			if result == nil {
				t.Fatal("Expected non-nil state")
			}

			fileOp := result.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			mkfileAction := fileOp.Actions[0].GetMkfile()
			if mkfileAction == nil {
				t.Fatal("Expected Mkfile action")
			}

			if tc.verify != nil {
				tc.verify(t, mkfileAction)
			}
		})
	}
}

func TestRmOptions(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 1, nil)

	testCases := []struct {
		name   string
		opts   *RmOptions
		verify func(*testing.T, *pb.FileActionRm)
	}{
		{
			name: "nil options",
			opts: nil,
			verify: func(t *testing.T, rmAction *pb.FileActionRm) {
				if rmAction.AllowNotFound != false {
					t.Error("Expected AllowNotFound false")
				}
				if rmAction.AllowWildcard != false {
					t.Error("Expected AllowWildcard false")
				}
			},
		},
		{
			name: "allow not found",
			opts: &RmOptions{
				AllowNotFound: true,
			},
			verify: func(t *testing.T, rmAction *pb.FileActionRm) {
				if rmAction.AllowNotFound != true {
					t.Error("Expected AllowNotFound true")
				}
			},
		},
		{
			name: "allow wildcard",
			opts: &RmOptions{
				AllowWildcard: true,
			},
			verify: func(t *testing.T, rmAction *pb.FileActionRm) {
				if rmAction.AllowWildcard != true {
					t.Error("Expected AllowWildcard true")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Rm(state, "/tmp", tc.opts, "test.lua", 2)
			if result == nil {
				t.Fatal("Expected non-nil state")
			}

			fileOp := result.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			rmAction := fileOp.Actions[0].GetRm()
			if rmAction == nil {
				t.Fatal("Expected Rm action")
			}

			if tc.verify != nil {
				tc.verify(t, rmAction)
			}
		})
	}
}

func TestMountTypes(t *testing.T) {
	t.Run("Cache mount", func(t *testing.T) {
		mount := CacheMount("/cache", nil)
		if mount == nil {
			t.Fatal("Expected non-nil mount")
		}

		pbMount := mount.ToPB()
		if pbMount.Dest != "/cache" {
			t.Errorf("Expected dest '/cache', got '%s'", pbMount.Dest)
		}
	})

	t.Run("Secret mount", func(t *testing.T) {
		mount := SecretMount("/secret", nil)
		if mount == nil {
			t.Fatal("Expected non-nil mount")
		}

		pbMount := mount.ToPB()
		if pbMount.Dest != "/secret" {
			t.Errorf("Expected dest '/secret', got '%s'", pbMount.Dest)
		}
	})

	t.Run("SSH mount", func(t *testing.T) {
		mount := SSHMount(nil)
		if mount == nil {
			t.Fatal("Expected non-nil mount")
		}

		pbMount := mount.ToPB()
		if pbMount.Dest != "/run/ssh" {
			t.Errorf("Expected dest '/run/ssh', got '%s'", pbMount.Dest)
		}
	})

	t.Run("Tmpfs mount", func(t *testing.T) {
		mount := TmpfsMount("/tmp", nil)
		if mount == nil {
			t.Fatal("Expected non-nil mount")
		}

		pbMount := mount.ToPB()
		if pbMount.Dest != "/tmp" {
			t.Errorf("Expected dest '/tmp', got '%s'", pbMount.Dest)
		}
	})
}

func TestIdentifiers(t *testing.T) {
	imageTestCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "simple image",
			input:  "alpine:3.19",
			expect: "docker-image://alpine:3.19",
		},
		{
			name:   "full image",
			input:  "docker.io/library/alpine:3.19",
			expect: "docker-image://docker.io/library/alpine:3.19",
		},
	}

	for _, tc := range imageTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ImageIdentifier(tc.input)
			if result != tc.expect {
				t.Errorf("Expected '%s', got '%s'", tc.expect, result)
			}
		})
	}

	t.Run("local", func(t *testing.T) {
		result := LocalIdentifier("context")
		if result != "local://context" {
			t.Errorf("Expected 'local://context', got '%s'", result)
		}
	})

	t.Run("git without ref", func(t *testing.T) {
		result := GitIdentifier("https://github.com/user/repo", "")
		if result != "git://https://github.com/user/repo" {
			t.Errorf("Expected 'git://https://github.com/user/repo', got '%s'", result)
		}
	})

	t.Run("git with ref", func(t *testing.T) {
		result := GitIdentifier("https://github.com/user/repo", "v1.0.0")
		if result != "git://https://github.com/user/repo#v1.0.0" {
			t.Errorf("Expected 'git://https://github.com/user/repo#v1.0.0', got '%s'", result)
		}
	})

}

func TestStateChaining(t *testing.T) {
	base := Image("alpine:3.19", "test.lua", 1, nil)
	s1 := Mkdir(base, "/app", nil, "test.lua", 2)
	s2 := Mkdir(s1, "/app/data", nil, "test.lua", 3)
	s3 := Mkfile(s2, "/app/config.json", "{}", nil, "test.lua", 4)
	s4 := Rm(s3, "/app/tmp", nil, "test.lua", 5)
	s5 := Symlink(s4, "/app/link", "/app/target", "test.lua", 6)

	if s1.Op().LuaFile() != "test.lua" {
		t.Error("Expected Lua file 'test.lua'")
	}
	if s2.Op().LuaLine() != 3 {
		t.Error("Expected Lua line 3")
	}
	if s3.Op().LuaLine() != 4 {
		t.Error("Expected Lua line 4")
	}
	if s4.Op().LuaLine() != 5 {
		t.Error("Expected Lua line 5")
	}
	if s5.Op().LuaLine() != 6 {
		t.Error("Expected Lua line 6")
	}
}

func TestComplexDAGConstruction(t *testing.T) {
	base := Image("golang:1.22", "build.lua", 1, nil)
	deps := Run(base, []string{"go", "mod", "download"},
		&ExecOptions{Cwd: "/app"}, "build.lua", 2)

	workspace := Copy(deps, deps, ".", "/app", nil, "build.lua", 3)
	build1 := Run(workspace, []string{"make", "build1"}, nil, "build.lua", 4)
	build2 := Run(workspace, []string{"make", "build2"}, nil, "build.lua", 5)
	test := Run(workspace, []string{"make", "test"}, nil, "build.lua", 6)

	merged := Merge([]*dag.State{build1, build2, test}, "build.lua", 7)

	if len(merged.Op().Inputs()) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(merged.Op().Inputs()))
	}
}

func TestScratchState(t *testing.T) {
	scratch := Scratch()

	if scratch == nil {
		t.Fatal("Expected non-nil scratch state")
	}

	sourceOp := scratch.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "scratch" {
		t.Errorf("Expected identifier 'scratch', got '%s'", sourceOp.Identifier)
	}
}
