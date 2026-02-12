package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNewEdgeWithZeroIndex(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	edge := NewEdge(op, 0)

	if edge.Node() != op {
		t.Error("Expected edge node to be op")
	}

	if edge.OutputIndex() != 0 {
		t.Error("Expected output index to be 0")
	}
}

func TestNewEdgeWithNegativeIndex(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	edge := NewEdge(op, -1)

	if edge.Node() != op {
		t.Error("Expected edge node to be op")
	}

	if edge.OutputIndex() != -1 {
		t.Error("Expected output index to be -1")
	}
}

func TestNewEdgeWithLargeIndex(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	edge := NewEdge(op, 9999)

	if edge.Node() != op {
		t.Error("Expected edge node to be op")
	}

	if edge.OutputIndex() != 9999 {
		t.Error("Expected output index to be 9999")
	}
}

func TestNewStateWithNegativeOutputIndex(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewStateWithOutput(op, -1)

	if state.OutputIndex() != -1 {
		t.Error("Expected output index to be -1")
	}
}

func TestNewStateWithLargeOutputIndex(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewStateWithOutput(op, 10000)

	if state.OutputIndex() != 10000 {
		t.Error("Expected output index to be 10000")
	}
}

func TestOpNodeAddInputNilEdge(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	op.AddInput(nil)

	// AddInput doesn't check for nil, so nil edge is added
	if len(op.Inputs()) != 1 {
		t.Error("Expected 1 input after adding nil edge")
	}

	if op.Inputs()[0] != nil {
		t.Error("Expected edge to be nil")
	}
}

func TestOpNodeAddMultipleInputs(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	op1 := NewOpNode(&pb.Op{}, "test.lua", 2)
	op2 := NewOpNode(&pb.Op{}, "test.lua", 3)
	op3 := NewOpNode(&pb.Op{}, "test.lua", 4)

	op.AddInput(NewEdge(op1, 0))
	op.AddInput(NewEdge(op2, 1))
	op.AddInput(NewEdge(op3, 2))

	if len(op.Inputs()) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(op.Inputs()))
	}

	if op.Inputs()[1].OutputIndex() != 1 {
		t.Error("Expected second input output index to be 1")
	}
}

func TestOpNodeSetMetadataOverwrite(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	metadata1 := &pb.OpMetadata{
		Description: map[string]string{"key": "value1"},
	}
	op.SetMetadata(metadata1)

	metadata2 := &pb.OpMetadata{
		Description: map[string]string{"key": "value2"},
	}
	op.SetMetadata(metadata2)

	if op.Metadata().Description["key"] != "value2" {
		t.Error("Expected metadata to be overwritten")
	}
}

func TestDigestForOpWithInputs(t *testing.T) {
	op1 := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}

	op2 := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: "sha256:abc123", Index: 0},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo test"},
				},
			},
		},
	}

	node1 := NewOpNode(op1, "test.lua", 1)
	node2 := NewOpNode(op2, "test.lua", 2)

	digest1 := node1.Digest()
	digest2 := node2.Digest()

	if digest1 == "" || digest2 == "" {
		t.Error("Expected non-empty digests")
	}

	if digest1 == digest2 {
		t.Error("Expected different digests for different ops")
	}
}

func TestDigestForExecOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo hello"},
				},
			},
		},
	}

	node := NewOpNode(op, "test.lua", 1)
	digest := node.Digest()

	if digest == "" {
		t.Error("Expected non-empty digest for ExecOp")
	}
}

func TestDigestForFileOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_File{
			File: &pb.FileOp{
				Actions: []*pb.FileAction{
					{
						Action: &pb.FileAction_Mkfile{
							Mkfile: &pb.FileActionMkFile{
								Path: "/test.txt",
								Data: []byte("test content"),
							},
						},
					},
				},
			},
		},
	}

	node := NewOpNode(op, "test.lua", 1)
	digest := node.Digest()

	if digest == "" {
		t.Error("Expected non-empty digest for FileOp")
	}
}

func TestDigestForMergeOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: "sha256:abc123", Index: 0},
			{Digest: "sha256:def456", Index: 0},
		},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}

	node := NewOpNode(op, "test.lua", 1)
	digest := node.Digest()

	if digest == "" {
		t.Error("Expected non-empty digest for MergeOp")
	}
}

func TestDigestForDiffOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: "sha256:abc123", Index: 0},
			{Digest: "sha256:def456", Index: 0},
		},
		Op: &pb.Op_Diff{
			Diff: &pb.DiffOp{},
		},
	}

	node := NewOpNode(op, "test.lua", 1)
	digest := node.Digest()

	if digest == "" {
		t.Error("Expected non-empty digest for DiffOp")
	}
}

func TestDigestIdenticalForSameOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}

	node1 := NewOpNode(op, "test.lua", 1)
	node2 := NewOpNode(op, "test.lua", 2)

	digest1 := node1.Digest()
	digest2 := node2.Digest()

	if digest1 != digest2 {
		t.Error("Expected identical digests for identical ops")
	}
}

func TestDigestDifferentForDifferentArgs(t *testing.T) {
	op1 := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/echo", "test1"},
				},
			},
		},
	}

	op2 := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/echo", "test2"},
				},
			},
		},
	}

	node1 := NewOpNode(op1, "test.lua", 1)
	node2 := NewOpNode(op2, "test.lua", 2)

	digest1 := node1.Digest()
	digest2 := node2.Digest()

	if digest1 == digest2 {
		t.Error("Expected different digests for different args")
	}
}

func TestStatePlatformOverride(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state1 := NewState(op)

	platform1 := &pb.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}
	platform2 := &pb.Platform{
		OS:           "linux",
		Architecture: "arm64",
	}

	state2 := state1.WithPlatform(platform1)
	state3 := state2.WithPlatform(platform2)

	if state1.Platform() != nil {
		t.Error("Expected original state platform to be nil")
	}

	if state2.Platform().Architecture != "amd64" {
		t.Error("Expected state2 platform to be amd64")
	}

	if state3.Platform().Architecture != "arm64" {
		t.Error("Expected state3 platform to be arm64")
	}
}

func TestOpNodeMetadataFields(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	metadata := &pb.OpMetadata{
		Description: map[string]string{
			"llb.custom": "my-description",
			"key2":       "value2",
		},
		ProgressGroup: &pb.ProgressGroup{
			Id:   "group-1",
			Name: "Test Group",
		},
	}

	op.SetMetadata(metadata)

	if op.Metadata().Description["llb.custom"] != "my-description" {
		t.Error("Expected custom description")
	}

	if op.Metadata().ProgressGroup.Id != "group-1" {
		t.Error("Expected progress group id")
	}
}

func TestOpNodeLuaLocationConsistency(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "build/script.lua", 100)

	if node.LuaFile() != "build/script.lua" {
		t.Errorf("Expected file 'build/script.lua', got '%s'", node.LuaFile())
	}

	if node.LuaLine() != 100 {
		t.Errorf("Expected line 100, got %d", node.LuaLine())
	}
}

func TestStateOutputIndexMutability(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	if state.OutputIndex() != 0 {
		t.Error("Expected default output index to be 0")
	}

	newState := NewStateWithOutput(op, 5)

	if newState.OutputIndex() != 5 {
		t.Error("Expected output index to be 5")
	}

	if state.OutputIndex() != 0 {
		t.Error("Expected original state output index to remain 0")
	}
}
