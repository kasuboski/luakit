package dag

import (
	"strings"
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestPropagateImageConfigs_AppliesWorkingDir(t *testing.T) {
	sourceOp := &pb.Op{
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{Identifier: "docker-image://golang:1.21"},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)
	sourceNode.SetImageConfig(&ImageConfig{
		Config: &ocispec.Image{
			Config: ocispec.ImageConfig{
				WorkingDir: "/go",
				Env:        []string{"PATH=/go/bin:/usr/bin"},
			},
		},
	})

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta:   &pb.Meta{Args: []string{"go", "build"}},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))

	state := NewState(execNode)

	propagateImageConfigs(state)

	if execOp.GetExec().Meta.Cwd != "/go" {
		t.Errorf("Expected cwd '/go', got '%s'", execOp.GetExec().Meta.Cwd)
	}

	if len(execOp.GetExec().Meta.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(execOp.GetExec().Meta.Env))
	}
}

func TestPropagateImageConfigs_UserCwdOverrides(t *testing.T) {
	sourceNode := NewOpNode(&pb.Op{
		Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://alpine"}},
	}, "test.lua", 1)
	sourceNode.SetImageConfig(&ImageConfig{
		Config: &ocispec.Image{Config: ocispec.ImageConfig{WorkingDir: "/go"}},
	})

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta:   &pb.Meta{Args: []string{"ls"}, Cwd: "/workspace"},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))

	propagateImageConfigs(NewState(execNode))

	if execOp.GetExec().Meta.Cwd != "/workspace" {
		t.Errorf("Expected user cwd '/workspace' to override, got '%s'", execOp.GetExec().Meta.Cwd)
	}
}

func TestPropagateImageConfigs_DefaultsToSlash(t *testing.T) {
	sourceNode := NewOpNode(&pb.Op{
		Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "scratch"}},
	}, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta:   &pb.Meta{Args: []string{"ls"}},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))

	propagateImageConfigs(NewState(execNode))

	if execOp.GetExec().Meta.Cwd != "/" {
		t.Errorf("Expected default cwd '/', got '%s'", execOp.GetExec().Meta.Cwd)
	}
}

func TestPropagateImageConfigs_EnvMerge(t *testing.T) {
	sourceNode := NewOpNode(&pb.Op{
		Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://alpine"}},
	}, "test.lua", 1)
	sourceNode.SetImageConfig(&ImageConfig{
		Config: &ocispec.Image{
			Config: ocispec.ImageConfig{
				Env: []string{"PATH=/usr/bin", "FOO=bar"},
			},
		},
	})

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"echo"},
					Env:  []string{"FOO=override", "BAZ=qux"},
				},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))

	propagateImageConfigs(NewState(execNode))

	env := execOp.GetExec().Meta.Env
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["FOO"] != "override" {
		t.Errorf("Expected FOO=override (user override), got %s", envMap["FOO"])
	}
	if envMap["PATH"] != "/usr/bin" {
		t.Errorf("Expected PATH=/usr/bin (from image), got %s", envMap["PATH"])
	}
	if envMap["BAZ"] != "qux" {
		t.Errorf("Expected BAZ=qux (user added), got %s", envMap["BAZ"])
	}
}

func TestPropagateImageConfigs_ChainedExecOps(t *testing.T) {
	sourceNode := NewOpNode(&pb.Op{
		Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://golang:1.21"}},
	}, "test.lua", 1)
	sourceNode.SetImageConfig(&ImageConfig{
		Config: &ocispec.Image{
			Config: ocispec.ImageConfig{WorkingDir: "/go"},
		},
	})

	execOp1 := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta:   &pb.Meta{Args: []string{"go", "mod", "download"}},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode1 := NewOpNode(execOp1, "test.lua", 2)
	execNode1.AddInput(NewEdge(sourceNode, 0))

	execOp2 := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta:   &pb.Meta{Args: []string{"go", "build"}},
				Mounts: []*pb.Mount{{Dest: "/", Input: 0, MountType: pb.MountType_BIND}},
			},
		},
	}
	execNode2 := NewOpNode(execOp2, "test.lua", 3)
	execNode2.AddInput(NewEdge(execNode1, 0))

	propagateImageConfigs(NewState(execNode2))

	if execOp1.GetExec().Meta.Cwd != "/go" {
		t.Errorf("Expected exec1 cwd '/go', got '%s'", execOp1.GetExec().Meta.Cwd)
	}
	if execOp2.GetExec().Meta.Cwd != "/go" {
		t.Errorf("Expected exec2 cwd '/go', got '%s'", execOp2.GetExec().Meta.Cwd)
	}
}

func TestApplyImageConfigToExec_NilConfig(t *testing.T) {
	exec := &pb.ExecOp{Meta: &pb.Meta{Args: []string{"ls"}}}

	applyImageConfigToExec(exec, nil)

	if exec.Meta.Cwd != "/" {
		t.Errorf("Expected default cwd '/', got '%s'", exec.Meta.Cwd)
	}
}

func TestApplyImageConfigToExec_NilMeta(t *testing.T) {
	exec := &pb.ExecOp{Meta: nil}

	applyImageConfigToExec(exec, nil)
}
