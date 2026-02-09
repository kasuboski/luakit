package ops

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNewDiffOp(t *testing.T) {
	lower := &pb.LowerDiffInput{Input: 0}
	upper := &pb.UpperDiffInput{Input: 1}

	op := NewDiffOp(lower, upper)

	if op.Lower == nil {
		t.Fatal("Expected non-nil Lower")
	}

	if op.Upper == nil {
		t.Fatal("Expected non-nil Upper")
	}

	if op.Lower.Input != 0 {
		t.Errorf("Expected lower input index 0, got %d", op.Lower.Input)
	}

	if op.Upper.Input != 1 {
		t.Errorf("Expected upper input index 1, got %d", op.Upper.Input)
	}
}

func TestDiff(t *testing.T) {
	lowerState := Scratch()
	upperState := Image("alpine:3.19", "test.lua", 5, nil)

	result := Diff(lowerState, upperState, "test.lua", 10)

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

	diffOp := result.Op().Op().GetDiff()
	if diffOp == nil {
		t.Fatal("Expected DiffOp")
	}

	if diffOp.Lower == nil {
		t.Fatal("Expected non-nil Lower in DiffOp")
	}

	if diffOp.Upper == nil {
		t.Fatal("Expected non-nil Upper in DiffOp")
	}

	if len(result.Op().Inputs()) != 2 {
		t.Errorf("Expected 2 node inputs, got %d", len(result.Op().Inputs()))
	}
}

func TestDiffWithNilLower(t *testing.T) {
	upperState := Scratch()

	result := Diff(nil, upperState, "test.lua", 10)

	if result != nil {
		t.Error("Expected nil result for nil lower state")
	}
}

func TestDiffWithNilUpper(t *testing.T) {
	lowerState := Scratch()

	result := Diff(lowerState, nil, "test.lua", 10)

	if result != nil {
		t.Error("Expected nil result for nil upper state")
	}
}

func TestNewDiffState(t *testing.T) {
	lowerState := Scratch()
	upperState := Image("alpine:3.19", "test.lua", 5, nil)

	result := NewDiffState(lowerState, upperState, "test.lua", 20)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", result.Op().LuaFile())
	}

	if result.Op().LuaLine() != 20 {
		t.Errorf("Expected Lua line 20, got %d", result.Op().LuaLine())
	}

	pbInputs := result.Op().Op().Inputs
	if len(pbInputs) != 2 {
		t.Errorf("Expected 2 pb inputs, got %d", len(pbInputs))
	}

	nodeInputs := result.Op().Inputs()
	if len(nodeInputs) != 2 {
		t.Errorf("Expected 2 node inputs, got %d", len(nodeInputs))
	}

	diffOp := result.Op().Op().GetDiff()
	if diffOp == nil {
		t.Fatal("Expected DiffOp")
	}

	if diffOp.Lower.Input != 0 {
		t.Errorf("Expected lower input index 0, got %d", diffOp.Lower.Input)
	}

	if diffOp.Upper.Input != 1 {
		t.Errorf("Expected upper input index 1, got %d", diffOp.Upper.Input)
	}
}
