package ops

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

func TestNewMergeOp(t *testing.T) {
	inputs := []*pb.MergeInput{
		{Input: 0},
		{Input: 1},
		{Input: 2},
	}

	op := NewMergeOp(inputs)

	if len(op.Inputs) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(op.Inputs))
	}

	if op.Inputs[0].Input != 0 {
		t.Errorf("Expected first input index 0, got %d", op.Inputs[0].Input)
	}

	if op.Inputs[1].Input != 1 {
		t.Errorf("Expected second input index 1, got %d", op.Inputs[1].Input)
	}

	if op.Inputs[2].Input != 2 {
		t.Errorf("Expected third input index 2, got %d", op.Inputs[2].Input)
	}
}

func TestMerge(t *testing.T) {
	state1 := Scratch()
	state2 := Scratch()
	state3 := Scratch()

	states := []*dag.State{state1, state2, state3}
	result := Merge(states, "test.lua", 10)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if result.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", result.Op().LuaFile())
	}

	if result.Op().LuaLine() != 10 {
		t.Errorf("Expected Lua line 10, got %d", result.Op().LuaLine())
	}

	mergeOp := result.Op().Op().GetMerge()
	if mergeOp == nil {
		t.Fatal("Expected MergeOp")
	}

	if len(mergeOp.Inputs) != 3 {
		t.Errorf("Expected 3 merge inputs, got %d", len(mergeOp.Inputs))
	}

	if len(result.Op().Inputs()) != 3 {
		t.Errorf("Expected 3 node inputs, got %d", len(result.Op().Inputs()))
	}
}

func TestMergeWithTwoStates(t *testing.T) {
	state1 := Image("alpine:3.19", "test.lua", 1, nil)
	state2 := Image("ubuntu:24.04", "test.lua", 2, nil)

	states := []*dag.State{state1, state2}
	result := Merge(states, "test.lua", 10)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	mergeOp := result.Op().Op().GetMerge()
	if mergeOp == nil {
		t.Fatal("Expected MergeOp")
	}

	if len(mergeOp.Inputs) != 2 {
		t.Errorf("Expected 2 merge inputs, got %d", len(mergeOp.Inputs))
	}
}

func TestMergeWithZeroStates(t *testing.T) {
	states := []*dag.State{}
	result := Merge(states, "test.lua", 10)

	if result != nil {
		t.Error("Expected nil result for zero states")
	}
}

func TestMergeWithOneState(t *testing.T) {
	state1 := Scratch()
	states := []*dag.State{state1}
	result := Merge(states, "test.lua", 10)

	if result != nil {
		t.Error("Expected nil result for one state")
	}
}

func TestNewMergeState(t *testing.T) {
	state1 := Scratch()
	state2 := Scratch()
	state3 := Scratch()

	states := []*dag.State{state1, state2, state3}
	result := NewMergeState(states, "test.lua", 15)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", result.Op().LuaFile())
	}

	if result.Op().LuaLine() != 15 {
		t.Errorf("Expected Lua line 15, got %d", result.Op().LuaLine())
	}

	pbInputs := result.Op().Op().Inputs
	if len(pbInputs) != 3 {
		t.Errorf("Expected 3 pb inputs, got %d", len(pbInputs))
	}

	nodeInputs := result.Op().Inputs()
	if len(nodeInputs) != 3 {
		t.Errorf("Expected 3 node inputs, got %d", len(nodeInputs))
	}
}
