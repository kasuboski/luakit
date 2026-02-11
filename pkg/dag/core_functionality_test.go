package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func TestSerializeWithNullMetadata(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)
	node.SetMetadata(&pb.OpMetadata{
		Description: nil,
	})
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) != 0 {
		t.Errorf("Expected 0 metadata entries, got %d", len(def.Metadata))
	}
}

func TestSerializeMergeWithMultipleInputs(t *testing.T) {
	inputs := make([]*OpNode, 5)
	for i := range 5 {
		op := &pb.Op{
			Inputs: []*pb.Input{},
			Op: &pb.Op_Source{
				Source: &pb.SourceOp{
					Identifier: "docker-image://base" + string(rune('0'+i)),
				},
			},
		}
		inputs[i] = NewOpNode(op, "test.lua", i+1)
	}

	mergeOp := &pb.Op{
		Inputs: make([]*pb.Input, 5),
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	mergeNode := NewOpNode(mergeOp, "test.lua", 10)

	for i, input := range inputs {
		mergeOp.Inputs[i] = &pb.Input{Digest: string(input.Digest()), Index: 0}
		mergeNode.AddInput(NewEdge(input, 0))
	}

	state := NewState(mergeNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 7 {
		t.Errorf("Expected 7 ops (5 sources + 1 merge), got %d", len(def.Def))
	}
}

func TestSerializeDiffOperation(t *testing.T) {
	lowerOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	lowerNode := NewOpNode(lowerOp, "test.lua", 1)

	upperOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo test > /file"},
				},
			},
		},
	}
	upperNode := NewOpNode(upperOp, "test.lua", 2)

	diffOp := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(lowerNode.Digest()), Index: 0},
			{Digest: string(upperNode.Digest()), Index: 0},
		},
		Op: &pb.Op_Diff{
			Diff: &pb.DiffOp{},
		},
	}
	diffNode := NewOpNode(diffOp, "test.lua", 3)
	diffNode.AddInput(NewEdge(lowerNode, 0))
	diffNode.AddInput(NewEdge(upperNode, 0))

	state := NewState(diffNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 4 {
		t.Errorf("Expected 4 ops, got %d", len(def.Def))
	}

	var foundDiffOp bool
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			continue
		}
		if diff := op.GetDiff(); diff != nil {
			foundDiffOp = true
			if len(op.Inputs) != 2 {
				t.Errorf("Expected 2 inputs in diff op, got %d", len(op.Inputs))
			}
			break
		}
	}

	if !foundDiffOp {
		t.Error("Expected to find DiffOp in definition")
	}
}

func TestSerializeWithLargeImageConfig(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	config := &dockerspec.DockerOCIImage{}
	config.OS = "linux"
	config.Architecture = "amd64"
	config.Config.Env = make([]string, 100)
	for i := range 100 {
		config.Config.Env[i] = "VAR" + string(rune('0'+i%10)) + "=value"
	}
	config.Config.Labels = make(map[string]string)
	for i := range 50 {
		config.Config.Labels["label"+string(rune('0'+i%10))+string(rune('0'+(i/10)%10))] = "value"
	}

	opts := &SerializeOptions{
		ImageConfig: config,
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) == 0 {
		t.Error("Expected metadata entries")
	}
}

func TestSerializeWithNilImageConfig(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	opts := &SerializeOptions{
		ImageConfig: nil,
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if def == nil {
		t.Error("Expected non-nil definition")
	}
}

func TestSerializeDeepChain(t *testing.T) {
	nodes := make([]*OpNode, 20)

	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	nodes[0] = NewOpNode(sourceOp, "test.lua", 1)

	for i := 1; i < 20; i++ {
		op := &pb.Op{
			Inputs: []*pb.Input{{Digest: string(nodes[i-1].Digest()), Index: 0}},
			Op: &pb.Op_Exec{
				Exec: &pb.ExecOp{
					Meta: &pb.Meta{
						Args: []string{"/bin/sh", "-c", "echo step" + string(rune('0'+i))},
					},
				},
			},
		}
		nodes[i] = NewOpNode(op, "test.lua", i+1)
		nodes[i].AddInput(NewEdge(nodes[i-1], 0))
	}

	state := NewState(nodes[19])

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 21 {
		t.Errorf("Expected 21 ops in definition, got %d", len(def.Def))
	}
}

func TestSerializeDAGWithMultipleOutputs(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	source := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(source.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/echo", "test"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/root",
						Output: 0,
					},
					{
						Input:  0,
						Dest:   "/tmp",
						Output: 1,
					},
				},
			},
		},
	}
	exec := NewOpNode(execOp, "test.lua", 2)
	exec.AddInput(NewEdge(source, 0))

	state0 := NewStateWithOutput(exec, 0)
	state1 := NewStateWithOutput(exec, 1)

	def0, err := Serialize(state0, nil)
	if err != nil {
		t.Fatalf("Failed to serialize state0: %v", err)
	}

	def1, err := Serialize(state1, nil)
	if err != nil {
		t.Fatalf("Failed to serialize state1: %v", err)
	}

	if len(def0.Def) != 3 {
		t.Errorf("Expected 3 ops in def0, got %d", len(def0.Def))
	}

	if len(def1.Def) != 3 {
		t.Errorf("Expected 3 ops in def1, got %d", len(def1.Def))
	}
}

func TestSerializeDiamondDAG(t *testing.T) {
	rootOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	root := NewOpNode(rootOp, "test.lua", 1)

	leftOp := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(root.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo left"},
				},
			},
		},
	}
	left := NewOpNode(leftOp, "test.lua", 2)
	left.AddInput(NewEdge(root, 0))

	rightOp := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(root.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo right"},
				},
			},
		},
	}
	right := NewOpNode(rightOp, "test.lua", 3)
	right.AddInput(NewEdge(root, 0))

	mergeOp := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(left.Digest()), Index: 0},
			{Digest: string(right.Digest()), Index: 0},
		},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	merge := NewOpNode(mergeOp, "test.lua", 4)
	merge.AddInput(NewEdge(left, 0))
	merge.AddInput(NewEdge(right, 0))

	state := NewState(merge)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 5 {
		t.Errorf("Expected 5 ops in diamond DAG, got %d", len(def.Def))
	}
}

func TestSerializeWithAllMountOptions(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	source := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(source.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "test"},
					Cwd:  "/workspace",
					User: "builder",
					Env: []string{
						"PATH=/usr/bin",
						"HOME=/home/builder",
					},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/root",
						Output: 0,
					},
					{
						Dest:      "/cache",
						MountType: pb.MountType_CACHE,
						CacheOpt: &pb.CacheOpt{
							ID:      "cache-id",
							Sharing: pb.CacheSharingOpt_SHARED,
						},
					},
					{
						Dest:      "/run/secrets/secret",
						MountType: pb.MountType_SECRET,
						SecretOpt: &pb.SecretOpt{
							ID:       "secret-id",
							Uid:      1000,
							Gid:      1000,
							Mode:     0600,
							Optional: false,
						},
					},
					{
						Dest:      "/run/ssh",
						MountType: pb.MountType_SSH,
						SSHOpt: &pb.SSHOpt{
							ID:       "ssh-id",
							Uid:      0,
							Gid:      0,
							Mode:     0600,
							Optional: false,
						},
					},
					{
						Dest:      "/tmp",
						MountType: pb.MountType_TMPFS,
						TmpfsOpt:  &pb.TmpfsOpt{Size: 1024 * 1024 * 1024},
					},
				},
			},
		},
	}
	exec := NewOpNode(execOp, "test.lua", 2)
	exec.AddInput(NewEdge(source, 0))

	state := NewState(exec)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 3 {
		t.Errorf("Expected 3 ops, got %d", len(def.Def))
	}

	var foundExecOp bool
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			continue
		}
		if exec := op.GetExec(); exec != nil {
			foundExecOp = true
			if len(exec.Mounts) != 5 {
				t.Errorf("Expected 5 mounts, got %d", len(exec.Mounts))
			}

			if exec.Meta.Cwd != "/workspace" {
				t.Errorf("Expected cwd '/workspace', got '%s'", exec.Meta.Cwd)
			}

			if exec.Meta.User != "builder" {
				t.Errorf("Expected user 'builder', got '%s'", exec.Meta.User)
			}

			if len(exec.Meta.Env) != 2 {
				t.Errorf("Expected 2 env vars, got %d", len(exec.Meta.Env))
			}
			break
		}
	}

	if !foundExecOp {
		t.Error("Expected to find ExecOp")
	}
}

func TestSerializeWithNilOptions(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize with nil options: %v", err)
	}

	if len(def.Def) != 2 {
		t.Errorf("Expected 2 ops, got %d", len(def.Def))
	}
}

func TestMultipleSerializationsSameState(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)
	state := NewState(node)

	def1, err1 := Serialize(state, nil)
	def2, err2 := Serialize(state, nil)
	def3, err3 := Serialize(state, nil)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatalf("Errors serializing: err1=%v, err2=%v, err3=%v", err1, err2, err3)
	}

	if len(def1.Def) != len(def2.Def) || len(def2.Def) != len(def3.Def) {
		t.Error("All serializations should have same number of ops")
	}

	for i := range def1.Def {
		if string(def1.Def[i]) != string(def2.Def[i]) || string(def2.Def[i]) != string(def3.Def[i]) {
			t.Errorf("Op %d differs between serializations", i)
		}
	}
}

func TestSerializeWithVeryLongIdentifiers(t *testing.T) {
	longIdentifier := "docker-image://registry.example.com:5000/namespace/very-long-repository-name-with-many-words-and-numbers-12345:v2.5.0-beta.1.build.12345.sha256.abc123def456"

	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: longIdentifier,
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize with long identifier: %v", err)
	}

	if len(def.Def) != 2 {
		t.Errorf("Expected 2 ops, got %d", len(def.Def))
	}

	var unmarshaledOp pb.Op
	if err := unmarshaledOp.UnmarshalVT(def.Def[0]); err != nil {
		t.Fatalf("Failed to unmarshal op: %v", err)
	}

	if source := unmarshaledOp.GetSource(); source != nil {
		if source.Identifier != longIdentifier {
			t.Errorf("Identifier not preserved: got '%s'", source.Identifier)
		}
	} else {
		t.Error("Expected SourceOp")
	}
}

func TestSerializeComplexMetadata(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)

	metadata := &pb.OpMetadata{
		Description: map[string]string{
			"llb.custom":            "custom description",
			"cachekey":              "cache-key-value",
			"moby.buildkit.buildid": "build-123",
			"com.example.label":     "example value",
		},
		ProgressGroup: &pb.ProgressGroup{
			Id:   "test-group-id",
			Name: "Test Progress Group",
		},
		Caps: map[string]bool{
			"exec":         true,
			"file.base":    true,
			"source.local": true,
		},
		IgnoreCache: true,
	}
	node.SetMetadata(metadata)
	state := NewState(node)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) != 1 {
		t.Errorf("Expected 1 metadata entry, got %d", len(def.Metadata))
	}

	for _, meta := range def.Metadata {
		if meta.Description == nil {
			t.Error("Expected description to be set")
		}

		if len(meta.Description) != 4 {
			t.Errorf("Expected 4 description entries, got %d", len(meta.Description))
		}

		if meta.ProgressGroup == nil {
			t.Error("Expected progress group to be set")
		}

		if meta.ProgressGroup.Id != "test-group-id" {
			t.Errorf("Expected progress group id 'test-group-id', got '%s'", meta.ProgressGroup.Id)
		}

		if !meta.IgnoreCache {
			t.Error("Expected IgnoreCache to be true")
		}

		if meta.Caps == nil {
			t.Error("Expected Caps to be set")
		}

		if len(meta.Caps) != 3 {
			t.Errorf("Expected 3 caps, got %d", len(meta.Caps))
		}
	}
}
