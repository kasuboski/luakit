package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestEdgeWithNilNode(t *testing.T) {
	edge := &Edge{
		node:        nil,
		outputIndex: 0,
	}

	if edge.Node() != nil {
		t.Error("Expected nil node from edge")
	}

	if edge.OutputIndex() != 0 {
		t.Error("Expected output index to be 0")
	}
}

func TestNewEdgeNilNode(t *testing.T) {
	edge := NewEdge(nil, 5)

	if edge.Node() != nil {
		t.Error("Expected nil node")
	}

	if edge.OutputIndex() != 5 {
		t.Error("Expected output index to be 5")
	}
}

func TestStateNilOp(t *testing.T) {
	state := &State{
		op:          nil,
		outputIndex: 0,
	}

	if state.Op() != nil {
		t.Error("Expected nil op")
	}

	if state.OutputIndex() != 0 {
		t.Error("Expected output index to be 0")
	}
}

func TestOpNodeNilOp(t *testing.T) {
	node := &OpNode{
		op:       nil,
		metadata: &pb.OpMetadata{},
		inputs:   []*Edge{},
		luaFile:  "test.lua",
		luaLine:  1,
	}

	if node.Op() != nil {
		t.Error("Expected nil op")
	}

	if len(node.Inputs()) != 0 {
		t.Error("Expected 0 inputs")
	}

	if node.LuaFile() != "test.lua" {
		t.Error("Expected Lua file to be test.lua")
	}

	if node.LuaLine() != 1 {
		t.Error("Expected Lua line to be 1")
	}
}

func TestAddInputMultipleTimes(t *testing.T) {
	op1 := NewOpNode(&pb.Op{}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{}, "test.lua", 2)
	op3 := NewOpNode(&pb.Op{}, "test.lua", 3)

	for range 5 {
		op1.AddInput(NewEdge(op2, 0))
	}

	if len(op1.Inputs()) != 5 {
		t.Errorf("Expected 5 inputs, got %d", len(op1.Inputs()))
	}

	op1.AddInput(NewEdge(op3, 1))
	if len(op1.Inputs()) != 6 {
		t.Errorf("Expected 6 inputs, got %d", len(op1.Inputs()))
	}
}

func TestStateOutputIndexConsistency(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	states := make([]*State, 5)
	for i := range 5 {
		states[i] = NewStateWithOutput(op, i)
	}

	for i, state := range states {
		if state.OutputIndex() != i {
			t.Errorf("Expected output index %d, got %d", i, state.OutputIndex())
		}

		if state.Op() != op {
			t.Error("Expected all states to share same Op")
		}
	}
}

func TestPlatformOverrideChain(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	platform1 := &pb.Platform{OS: "linux", Architecture: "amd64"}
	platform2 := &pb.Platform{OS: "linux", Architecture: "arm64"}
	platform3 := &pb.Platform{OS: "linux", Architecture: "arm64", Variant: "v8"}

	state0 := NewState(op)
	state1 := state0.WithPlatform(platform1)
	state2 := state1.WithPlatform(platform2)
	state3 := state2.WithPlatform(platform3)

	if state0.Platform() != nil {
		t.Error("Expected state0 platform to be nil")
	}

	if state1.Platform().Architecture != "amd64" {
		t.Error("Expected state1 platform to be amd64")
	}

	if state2.Platform().Architecture != "arm64" {
		t.Error("Expected state2 platform to be arm64")
	}

	if state3.Platform().Variant != "v8" {
		t.Error("Expected state3 platform variant to be v8")
	}
}

func TestDigestUniquenessAcrossOpTypes(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{Identifier: "docker-image://alpine:3.19"},
		},
	}

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "echo"}},
			},
		},
	}

	mergeOp := &pb.Op{
		Inputs: []*pb.Input{{}, {}},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}

	fileOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_File{
			File: &pb.FileOp{
				Actions: []*pb.FileAction{
					{Action: &pb.FileAction_Mkfile{Mkfile: &pb.FileActionMkFile{Path: "/test"}}},
				},
			},
		},
	}

	diffOp := &pb.Op{
		Inputs: []*pb.Input{{}, {}},
		Op: &pb.Op_Diff{
			Diff: &pb.DiffOp{},
		},
	}

	sourceNode := NewOpNode(sourceOp, "test.lua", 1)
	execNode := NewOpNode(execOp, "test.lua", 2)
	mergeNode := NewOpNode(mergeOp, "test.lua", 3)
	fileNode := NewOpNode(fileOp, "test.lua", 4)
	diffNode := NewOpNode(diffOp, "test.lua", 5)

	digests := []string{
		string(sourceNode.Digest()),
		string(execNode.Digest()),
		string(mergeNode.Digest()),
		string(fileNode.Digest()),
		string(diffNode.Digest()),
	}

	for i, d1 := range digests {
		if d1 == "" {
			t.Errorf("Expected non-empty digest for op type %d", i)
		}
		for j, d2 := range digests {
			if i != j && d1 == d2 {
				t.Errorf("Expected different digests for different op types (%d and %d)", i, j)
			}
		}
	}
}

func TestDigestWithDifferentInputs(t *testing.T) {
	sourceNode := NewOpNode(&pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://alpine:3.19"}},
	}, "test.lua", 1)

	sourceNode2 := NewOpNode(&pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://ubuntu:24.04"}},
	}, "test.lua", 2)

	execOp1 := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(sourceNode.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "echo1"}},
			},
		},
	}

	execOp2 := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(sourceNode2.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "echo2"}},
			},
		},
	}

	execNode1 := NewOpNode(execOp1, "test.lua", 3)
	execNode2 := NewOpNode(execOp2, "test.lua", 4)

	digest1 := execNode1.Digest()
	digest2 := execNode2.Digest()

	if digest1 == "" || digest2 == "" {
		t.Error("Expected non-empty digests")
	}

	if digest1 == digest2 {
		t.Error("Expected different digests for different inputs")
	}
}

func TestMetadataMutationIndependence(t *testing.T) {
	node1 := NewOpNode(&pb.Op{}, "test.lua", 1)
	node2 := NewOpNode(&pb.Op{}, "test.lua", 2)

	metadata1 := &pb.OpMetadata{
		Description: map[string]string{"key1": "value1"},
	}

	metadata2 := &pb.OpMetadata{
		Description: map[string]string{"key2": "value2"},
	}

	node1.SetMetadata(metadata1)
	node2.SetMetadata(metadata2)

	if node1.Metadata().Description["key1"] != "value1" {
		t.Error("Expected node1 metadata to have key1")
	}

	if node2.Metadata().Description["key2"] != "value2" {
		t.Error("Expected node2 metadata to have key2")
	}

	node1.Metadata().Description["key1"] = "modified"
	if metadata1.Description["key1"] != "modified" {
		t.Error("Expected metadata to be mutable")
	}

	if node1.Metadata().Description["key1"] != "modified" {
		t.Error("Expected node1 metadata to reflect modification")
	}
}

func TestEdgeOutputIndexVariations(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	testCases := []int{0, 1, 5, 100, -1}

	for _, index := range testCases {
		edge := NewEdge(op, index)
		if edge.OutputIndex() != index {
			t.Errorf("Expected output index %d, got %d", index, edge.OutputIndex())
		}

		if edge.Node() != op {
			t.Error("Expected edge to point to op")
		}
	}
}

func TestLuaLocationTracking(t *testing.T) {
	testCases := []struct {
		file string
		line int
	}{
		{"build.lua", 1},
		{"path/to/script.lua", 100},
		{"", 0},
		{"test.lua", -1},
	}

	for _, tc := range testCases {
		node := NewOpNode(&pb.Op{}, tc.file, tc.line)

		if node.LuaFile() != tc.file {
			t.Errorf("Expected Lua file '%s', got '%s'", tc.file, node.LuaFile())
		}

		if node.LuaLine() != tc.line {
			t.Errorf("Expected Lua line %d, got %d", tc.line, node.LuaLine())
		}
	}
}

func TestStateCreationVariations(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	state1 := NewState(op)
	state2 := NewStateWithOutput(op, 5)
	state3 := NewStateWithOutput(op, 0)

	if state1.OutputIndex() != 0 {
		t.Error("Expected state1 output index to be 0")
	}

	if state2.OutputIndex() != 5 {
		t.Error("Expected state2 output index to be 5")
	}

	if state3.OutputIndex() != 0 {
		t.Error("Expected state3 output index to be 0")
	}

	if state1 == state2 {
		t.Error("Expected different State objects")
	}

	if state1.Op() != state2.Op() {
		t.Error("Expected same Op for both states")
	}
}

func TestEmptyInputsSlice(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	if len(op.Inputs()) != 0 {
		t.Error("Expected 0 inputs for new node")
	}

	for i := range 3 {
		op.AddInput(NewEdge(NewOpNode(&pb.Op{}, "test.lua", i+2), 0))
	}

	if len(op.Inputs()) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(op.Inputs()))
	}
}

func TestMetadataSetAndRetrieve(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "test.lua", 1)

	desc := map[string]string{
		"llb.custom": "custom description",
		"test.key":   "test value",
	}

	progress := &pb.ProgressGroup{
		Id:   "group-id",
		Name: "Test Group",
	}

	metadata := &pb.OpMetadata{
		Description:   desc,
		ProgressGroup: progress,
	}

	node.SetMetadata(metadata)

	retrieved := node.Metadata()
	if retrieved == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if len(retrieved.Description) != 2 {
		t.Errorf("Expected 2 description entries, got %d", len(retrieved.Description))
	}

	if retrieved.Description["llb.custom"] != "custom description" {
		t.Error("Expected custom description")
	}

	if retrieved.ProgressGroup == nil {
		t.Error("Expected non-nil progress group")
	}

	if retrieved.ProgressGroup.Id != "group-id" {
		t.Error("Expected progress group id")
	}
}

func TestDigestConsistency(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{Identifier: "docker-image://alpine:3.19"},
		},
	}

	node := NewOpNode(op, "test.lua", 1)

	digest1 := node.Digest()
	digest2 := node.Digest()
	digest3 := node.Digest()

	if digest1 == "" {
		t.Error("Expected non-empty digest")
	}

	if digest1 != digest2 {
		t.Error("Expected digests to be consistent")
	}

	if digest2 != digest3 {
		t.Error("Expected digests to be consistent")
	}
}

func TestEdgeNilHandling(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	var nilEdge *Edge = nil

	op.AddInput(nilEdge)

	if len(op.Inputs()) != 1 {
		t.Errorf("Expected 1 input after adding nil edge, got %d", len(op.Inputs()))
	}

	if op.Inputs()[0] != nil {
		t.Error("Expected nil edge to be added")
	}
}

func TestStatePlatformNil(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	if state.Platform() != nil {
		t.Error("Expected nil platform for new state")
	}

	newState := state.WithPlatform(nil)

	if newState.Platform() != nil {
		t.Error("Expected nil platform after WithPlatform(nil)")
	}

	if state.Platform() != nil {
		t.Error("Expected original state platform to remain nil")
	}
}

func TestOpNodeWithZeroLuaLine(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "test.lua", 0)

	if node.LuaLine() != 0 {
		t.Errorf("Expected Lua line 0, got %d", node.LuaLine())
	}

	if node.LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", node.LuaFile())
	}
}

func TestNodeEquality(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://alpine:3.19"}},
	}

	node1 := NewOpNode(op, "test.lua", 1)
	node2 := NewOpNode(op, "test.lua", 1)

	if node1 == node2 {
		t.Error("Expected different Node objects")
	}

	if node1.Digest() != node2.Digest() {
		t.Error("Expected same digest for identical nodes")
	}
}

func TestInputDigestPopulation(t *testing.T) {
	base := NewOpNode(&pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://alpine:3.19"}},
	}, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{{}, {}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "test"}},
			},
		},
	}

	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(base, 0))

	secondBase := NewOpNode(&pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Source{Source: &pb.SourceOp{Identifier: "docker-image://ubuntu:24.04"}},
	}, "test.lua", 3)
	execNode.AddInput(NewEdge(secondBase, 1))

	if len(execNode.Inputs()) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(execNode.Inputs()))
	}

	if execNode.Inputs()[0].Node() != base {
		t.Error("Expected first input to be base")
	}

	if execNode.Inputs()[0].OutputIndex() != 0 {
		t.Errorf("Expected first input output index to be 0, got %d", execNode.Inputs()[0].OutputIndex())
	}

	if execNode.Inputs()[1].Node() != secondBase {
		t.Error("Expected second input to be secondBase")
	}

	if execNode.Inputs()[1].OutputIndex() != 1 {
		t.Errorf("Expected second input output index to be 1, got %d", execNode.Inputs()[1].OutputIndex())
	}
}

func TestStateWithDifferentOutputIndicesSameOp(t *testing.T) {
	op := NewOpNode(&pb.Op{
		Inputs: []*pb.Input{},
		Op:     &pb.Op_Exec{Exec: &pb.ExecOp{Meta: &pb.Meta{}}},
	}, "test.lua", 1)

	state0 := NewStateWithOutput(op, 0)
	state1 := NewStateWithOutput(op, 1)
	state2 := NewStateWithOutput(op, 2)

	if state0 == state1 || state0 == state2 || state1 == state2 {
		t.Error("Expected different State objects")
	}

	if state0.OutputIndex() != 0 {
		t.Errorf("Expected state0 output index to be 0, got %d", state0.OutputIndex())
	}

	if state1.OutputIndex() != 1 {
		t.Errorf("Expected state1 output index to be 1, got %d", state1.OutputIndex())
	}

	if state2.OutputIndex() != 2 {
		t.Errorf("Expected state2 output index to be 2, got %d", state2.OutputIndex())
	}

	if state0.Op() != state1.Op() || state1.Op() != state2.Op() {
		t.Error("Expected all states to share same Op")
	}
}

func TestMetadataEmptyMap(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "test.lua", 1)

	metadata := &pb.OpMetadata{
		Description:   map[string]string{},
		ProgressGroup: nil,
	}

	node.SetMetadata(metadata)

	retrieved := node.Metadata()
	if retrieved == nil {
		t.Fatal("Expected non-nil metadata")
	}

	if len(retrieved.Description) != 0 {
		t.Errorf("Expected empty description map, got %d entries", len(retrieved.Description))
	}

	if retrieved.ProgressGroup != nil {
		t.Error("Expected nil progress group")
	}
}
