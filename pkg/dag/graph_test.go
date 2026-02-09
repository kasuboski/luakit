package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNewState(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	if state.Op() != op {
		t.Errorf("Expected op %v, got %v", op, state.Op())
	}

	if state.OutputIndex() != 0 {
		t.Errorf("Expected output index 0, got %d", state.OutputIndex())
	}

	if state.Platform() != nil {
		t.Errorf("Expected nil platform, got %v", state.Platform())
	}
}

func TestNewStateWithOutput(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewStateWithOutput(op, 2)

	if state.OutputIndex() != 2 {
		t.Errorf("Expected output index 2, got %d", state.OutputIndex())
	}
}

func TestStateWithPlatform(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	platform := &pb.Platform{
		OS:           "linux",
		Architecture: "arm64",
	}

	newState := state.WithPlatform(platform)

	if newState.Platform() != platform {
		t.Errorf("Expected platform %v, got %v", platform, newState.Platform())
	}

	if state.Platform() != nil {
		t.Errorf("Expected original state platform to be nil, got %v", state.Platform())
	}
}

func TestNewOpNode(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 5)

	if op.LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", op.LuaFile())
	}

	if op.LuaLine() != 5 {
		t.Errorf("Expected Lua line 5, got %d", op.LuaLine())
	}

	if len(op.Inputs()) != 0 {
		t.Errorf("Expected 0 inputs, got %d", len(op.Inputs()))
	}
}

func TestOpNodeAddInput(t *testing.T) {
	op1 := NewOpNode(&pb.Op{}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{}, "test.lua", 2)

	edge := &Edge{
		node:        op1,
		outputIndex: 0,
	}

	op2.AddInput(edge)

	if len(op2.Inputs()) != 1 {
		t.Errorf("Expected 1 input, got %d", len(op2.Inputs()))
	}

	if op2.Inputs()[0] != edge {
		t.Errorf("Expected edge %v, got %v", edge, op2.Inputs()[0])
	}
}

func TestOpNodeSetMetadata(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	metadata := &pb.OpMetadata{
		Description: map[string]string{"key": "value"},
	}

	op.SetMetadata(metadata)

	if op.Metadata().Description["key"] != "value" {
		t.Errorf("Expected description 'value', got '%s'", op.Metadata().Description["key"])
	}
}

func TestDigest(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
	}

	node := NewOpNode(op, "test.lua", 1)
	digest := node.Digest()

	if digest == "" {
		t.Error("Expected non-empty digest")
	}

	digest2 := node.Digest()
	if digest != digest2 {
		t.Error("Expected digests to be equal")
	}
}
