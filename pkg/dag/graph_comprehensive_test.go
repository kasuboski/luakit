package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func TestStateImmutability(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	if state.Op() != op {
		t.Error("Expected op to be set")
	}

	if state.OutputIndex() != 0 {
		t.Error("Expected output index to be 0")
	}

	if state.Platform() != nil {
		t.Error("Expected platform to be nil")
	}
}

func TestStateWithPlatformImmutability(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	platform := &pb.Platform{
		OS:           "linux",
		Architecture: "arm64",
	}

	newState := state.WithPlatform(platform)

	if newState.Op() != op {
		t.Error("Expected op to be preserved")
	}

	if newState.Platform() != platform {
		t.Error("Expected platform to be set")
	}

	if state.Platform() != nil {
		t.Error("Expected original state platform to remain nil")
	}
}

func TestOpNodeWithNilMetadata(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	if op.Metadata() == nil {
		t.Error("Expected metadata to be initialized")
	}

	if len(op.Metadata().Description) != 0 {
		t.Error("Expected empty description")
	}
}

func TestOpNodeInputs(t *testing.T) {
	op1 := NewOpNode(&pb.Op{}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{}, "test.lua", 2)
	op3 := NewOpNode(&pb.Op{}, "test.lua", 3)

	op2.AddInput(NewEdge(op1, 0))
	op2.AddInput(NewEdge(op3, 1))

	if len(op2.Inputs()) != 2 {
		t.Errorf("Expected 2 inputs, got %d", len(op2.Inputs()))
	}

	if op2.Inputs()[0].Node() != op1 {
		t.Error("Expected first input to be op1")
	}

	if op2.Inputs()[0].OutputIndex() != 0 {
		t.Error("Expected first output index to be 0")
	}

	if op2.Inputs()[1].Node() != op3 {
		t.Error("Expected second input to be op3")
	}

	if op2.Inputs()[1].OutputIndex() != 1 {
		t.Error("Expected second output index to be 1")
	}
}

func TestEdgeMethods(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	edge := NewEdge(op, 5)

	if edge.Node() != op {
		t.Error("Expected edge to return op")
	}

	if edge.OutputIndex() != 5 {
		t.Error("Expected output index to be 5")
	}
}

func TestDigestUniqueness(t *testing.T) {
	op1 := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}

	op2 := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://ubuntu:24.04",
			},
		},
	}

	node1 := NewOpNode(op1, "test.lua", 1)
	node2 := NewOpNode(op2, "test.lua", 1)

	digest1 := node1.Digest()
	digest2 := node2.Digest()

	if digest1 == "" {
		t.Error("Expected digest1 to be non-empty")
	}

	if digest2 == "" {
		t.Error("Expected digest2 to be non-empty")
	}

	if digest1 == digest2 {
		t.Error("Expected digests to be different")
	}
}

func TestDigestDeterminism(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}

	node1 := NewOpNode(op, "test.lua", 1)
	node2 := NewOpNode(op, "test.lua", 1)

	digest1 := node1.Digest()
	digest2 := node2.Digest()

	if digest1 != digest2 {
		t.Error("Expected digests to be identical")
	}
}

func TestOpNodeLuaLocation(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "build.lua", 42)

	if node.LuaFile() != "build.lua" {
		t.Errorf("Expected Lua file 'build.lua', got '%s'", node.LuaFile())
	}

	if node.LuaLine() != 42 {
		t.Errorf("Expected Lua line 42, got %d", node.LuaLine())
	}
}

func TestStateWithDifferentOutputIndices(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state1 := NewState(op)
	state2 := NewStateWithOutput(op, 5)

	if state1.OutputIndex() != 0 {
		t.Error("Expected default output index to be 0")
	}

	if state2.OutputIndex() != 5 {
		t.Error("Expected output index to be 5")
	}

	if state1.Op() != state2.Op() {
		t.Error("Expected both states to share the same OpNode")
	}
}

func TestCircularDependencyDetection(t *testing.T) {
	op1 := NewOpNode(&pb.Op{}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{}, "test.lua", 2)
	op3 := NewOpNode(&pb.Op{}, "test.lua", 3)

	op1.AddInput(NewEdge(op2, 0))
	op2.AddInput(NewEdge(op3, 0))
	op3.AddInput(NewEdge(op1, 0))

	state := NewState(op1)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Circular dependencies should be handled gracefully by only visiting each node once
	if len(def.Def) != 1 {
		t.Errorf("Expected 1 op in definition (visited set prevents infinite loop), got %d", len(def.Def))
	}
}

func TestDeepDAGSerialization(t *testing.T) {
	nodes := make([]*OpNode, 10)

	for i := range 10 {
		op := &pb.Op{
			Inputs: []*pb.Input{},
			Op: &pb.Op_Source{
				Source: &pb.SourceOp{
					Identifier: "docker-image://alpine:3.19",
				},
			},
		}
		nodes[i] = NewOpNode(op, "test.lua", i+1)
	}

	for i := 1; i < 10; i++ {
		// Make each op unique by setting different identifiers
		op := &pb.Op{
			Inputs: []*pb.Input{},
			Op: &pb.Op_Exec{
				Exec: &pb.ExecOp{
					Meta: &pb.Meta{
						Args: []string{"/bin/sh", "-c", "echo step", string(rune('0' + i))},
					},
				},
			},
		}
		nodes[i] = NewOpNode(op, "test.lua", i+1)
		nodes[i].AddInput(NewEdge(nodes[i-1], 0))
	}

	state := NewState(nodes[9])

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 10 {
		t.Errorf("Expected 10 ops in definition, got %d", len(def.Def))
	}
}

func TestWidelyBranchingDAG(t *testing.T) {
	baseOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}
	base := NewOpNode(baseOp, "test.lua", 1)

	branches := make([]*OpNode, 5)
	for i := range 5 {
		branchOp := &pb.Op{
			Inputs: []*pb.Input{},
			Op: &pb.Op_Exec{
				Exec: &pb.ExecOp{
					Meta: &pb.Meta{
						Args: []string{"/bin/sh", "-c", "echo branch", string(rune('0' + i))},
					},
				},
			},
		}
		branches[i] = NewOpNode(branchOp, "test.lua", i+2)
		branches[i].AddInput(NewEdge(base, 0))
	}

	mergeOp := &pb.Op{
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	merge := NewOpNode(mergeOp, "test.lua", 10)
	for _, branch := range branches {
		merge.AddInput(NewEdge(branch, 0))
	}

	state := NewState(merge)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 7 {
		t.Errorf("Expected 7 ops in definition, got %d", len(def.Def))
	}
}

func TestSharedNodeDAG(t *testing.T) {
	baseOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}
	base := NewOpNode(baseOp, "test.lua", 1)

	sharedOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo shared"},
				},
			},
		},
	}
	shared := NewOpNode(sharedOp, "test.lua", 2)

	branch1Op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo branch1"},
				},
			},
		},
	}
	branch1 := NewOpNode(branch1Op, "test.lua", 3)

	branch2Op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo branch2"},
				},
			},
		},
	}
	branch2 := NewOpNode(branch2Op, "test.lua", 4)

	branch1.AddInput(NewEdge(base, 0))
	branch1.AddInput(NewEdge(shared, 0))

	branch2.AddInput(NewEdge(base, 0))
	branch2.AddInput(NewEdge(shared, 0))

	state := NewState(branch1)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Should serialize: base, shared, branch1 (branch2 is not reached)
	if len(def.Def) != 3 {
		t.Errorf("Expected 3 ops in definition, got %d", len(def.Def))
	}
}

func TestSerializeWithNilSourceMapBuilder(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Error("Expected Source to be initialized")
	}
}

func TestSerializeWithEmptyMetadata(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:3.19",
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)
	metadata := &pb.OpMetadata{}
	node.SetMetadata(metadata)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) != 0 {
		t.Errorf("Expected 0 metadata entries, got %d", len(def.Metadata))
	}
}

func TestSerializeWithNilOp(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
	}
	node := NewOpNode(op, "test.lua", 1)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 1 {
		t.Errorf("Expected 1 op in definition, got %d", len(def.Def))
	}
}

func TestMultipleEdgesToSameNode(t *testing.T) {
	source := NewOpNode(&pb.Op{}, "test.lua", 1)
	other := NewOpNode(&pb.Op{}, "test.lua", 2)
	merge := NewOpNode(&pb.Op{}, "test.lua", 3)

	merge.AddInput(NewEdge(source, 0))
	merge.AddInput(NewEdge(other, 0))
	merge.AddInput(NewEdge(source, 1))

	if len(merge.Inputs()) != 3 {
		t.Errorf("Expected 3 inputs, got %d", len(merge.Inputs()))
	}

	if merge.Inputs()[0].Node() != source {
		t.Error("Expected first input to be source")
	}

	if merge.Inputs()[2].Node() != source {
		t.Error("Expected third input to be source")
	}
}

func TestStateEquality(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state1 := NewState(op)
	state2 := NewState(op)

	if state1 == state2 {
		t.Error("Expected different State objects even with same Op")
	}

	if state1.Op() != state2.Op() {
		t.Error("Expected same Op")
	}
}

func TestOpNodeMetadataMutation(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	metadata1 := op.Metadata()
	metadata1.Description = map[string]string{"key": "value"}

	metadata2 := op.Metadata()

	if metadata2.Description["key"] != "value" {
		t.Error("Expected metadata to be mutable")
	}

	metadata2.Description["key2"] = "value2"

	if metadata1.Description["key2"] != "value2" {
		t.Error("Expected metadata references to be same")
	}
}

func TestSerializeEmptyDAG(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op:     nil,
	}
	node := NewOpNode(op, "test.lua", 1)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize empty DAG: %v", err)
	}

	if len(def.Def) != 1 {
		t.Errorf("Expected 1 op in definition, got %d", len(def.Def))
	}
}

func TestMultipleStatesFromSameOp(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	state0 := NewState(op)
	state1 := NewStateWithOutput(op, 1)
	state2 := NewStateWithOutput(op, 2)

	if state0.OutputIndex() != 0 {
		t.Error("Expected state0 output index to be 0")
	}

	if state1.OutputIndex() != 1 {
		t.Error("Expected state1 output index to be 1")
	}

	if state2.OutputIndex() != 2 {
		t.Error("Expected state2 output index to be 2")
	}

	if state0.Op() != state1.Op() {
		t.Error("Expected states to share same Op")
	}
}

func TestSerializePreservesInputOrder(t *testing.T) {
	op1 := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "op1"}}}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "op2"}}}, "test.lua", 2)
	op3 := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "op3"}}}, "test.lua", 3)

	mergeInputs := []*pb.MergeInput{
		{Input: 0},
		{Input: 1},
		{Input: 2},
	}

	inputs := []*pb.Input{
		{Digest: string(op1.Digest()), Index: 0},
		{Digest: string(op2.Digest()), Index: 0},
		{Digest: string(op3.Digest()), Index: 0},
	}

	mergeOp := &pb.Op{
		Inputs: inputs,
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{
				Inputs: mergeInputs,
			},
		},
	}

	merge := NewOpNode(mergeOp, "test.lua", 4)
	merge.AddInput(NewEdge(op1, 0))
	merge.AddInput(NewEdge(op2, 0))
	merge.AddInput(NewEdge(op3, 0))

	state := NewState(merge)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var foundMergeOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			continue
		}
		if op.GetMerge() != nil {
			foundMergeOp = op
			break
		}
	}

	if foundMergeOp == nil {
		t.Fatal("Expected to find merge op")
	}

	if len(foundMergeOp.Inputs) != 3 {
		t.Errorf("Expected 3 inputs in merge op, got %d", len(foundMergeOp.Inputs))
	}
}

func TestEdgeNilNode(t *testing.T) {
	edge := &Edge{
		node:        nil,
		outputIndex: 0,
	}

	if edge.Node() != nil {
		t.Error("Expected nil node")
	}

	if edge.OutputIndex() != 0 {
		t.Error("Expected output index to be 0")
	}
}

func TestNewEdgeWithNilNode(t *testing.T) {
	edge := NewEdge(nil, 5)

	if edge.Node() != nil {
		t.Error("Expected nil node from NewEdge")
	}

	if edge.OutputIndex() != 5 {
		t.Error("Expected output index to be 5")
	}
}

func TestOpNodeWithNilLuaFile(t *testing.T) {
	node := NewOpNode(&pb.Op{}, "", 0)

	if node.LuaFile() != "" {
		t.Error("Expected empty Lua file")
	}

	if node.LuaLine() != 0 {
		t.Error("Expected Lua line to be 0")
	}
}

func TestSerializeWithMultipleMetadataEntries(t *testing.T) {
	// Create ops with different content so they have different digests
	op1 := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "op1"}}}, "test.lua", 1)
	op2 := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "op2"}}}, "test.lua", 2)

	metadata1 := &pb.OpMetadata{
		Description: map[string]string{"op1": "value"},
	}
	op1.SetMetadata(metadata1)

	metadata2 := &pb.OpMetadata{
		Description: map[string]string{"op2": "value"},
	}
	op2.SetMetadata(metadata2)

	op2.AddInput(NewEdge(op1, 0))

	state := NewState(op2)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(def.Metadata))
	}
}

func TestStateWithPlatformNil(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	stateWithNil := state.WithPlatform(nil)

	if stateWithNil.Platform() != nil {
		t.Error("Expected platform to be nil")
	}
}

func TestComplexDAGWithMergeAndDiff(t *testing.T) {
	// Create ops with different content to ensure unique digests
	base := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "base"}}}, "test.lua", 1)
	modified := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "modified"}}}, "test.lua", 2)
	diffed := NewOpNode(&pb.Op{Op: &pb.Op_Diff{Diff: &pb.DiffOp{}}}, "test.lua", 3)

	diffed.AddInput(NewEdge(base, 0))
	diffed.AddInput(NewEdge(modified, 0))

	another := NewOpNode(&pb.Op{Op: &pb.Op_Source{Source: &pb.SourceOp{Identifier: "another"}}}, "test.lua", 4)
	merged := NewOpNode(&pb.Op{Op: &pb.Op_Merge{Merge: &pb.MergeOp{}}}, "test.lua", 5)

	merged.AddInput(NewEdge(diffed, 0))
	merged.AddInput(NewEdge(another, 0))

	state := NewState(merged)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 5 {
		t.Errorf("Expected 5 ops in definition, got %d", len(def.Def))
	}
}

func TestOpNodeSetMetadataNil(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)

	op.SetMetadata(nil)

	if op.Metadata() != nil {
		t.Error("Expected metadata to be nil after setting to nil")
	}
}

func TestSerializeWithSourceFiles(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	opts := &SerializeOptions{
		SourceFiles: map[string][]byte{
			"test.lua": []byte("print('hello')"),
		},
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Error("Expected Source to be initialized")
	}
}

func TestSerializeWithImageConfigSimple(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	opts := &SerializeOptions{
		ImageConfig: &dockerspec.DockerOCIImage{},
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) == 0 {
		t.Error("Expected metadata with image config")
	}
}
