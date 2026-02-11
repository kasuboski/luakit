package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestSerializeSingleNode(t *testing.T) {
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

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 2 {
		t.Errorf("Expected 2 ops in definition, got %d", len(def.Def))
	}

	var unmarshaledOp pb.Op
	if err := unmarshaledOp.UnmarshalVT(def.Def[0]); err != nil {
		t.Fatalf("Failed to unmarshal op: %v", err)
	}

	sourceOp := unmarshaledOp.GetSource()
	if sourceOp == nil {
		t.Error("Expected source operation")
	}

	if sourceOp.Identifier != "docker-image://alpine:latest" {
		t.Errorf("Expected identifier 'docker-image://alpine:latest', got '%s'", sourceOp.Identifier)
	}
}

func TestSerializeWithMetadata(t *testing.T) {
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
			"llb.custom": "test",
		},
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
		if meta.Description["llb.custom"] != "test" {
			t.Errorf("Expected metadata description 'test', got '%s'", meta.Description["llb.custom"])
		}
	}
}

func TestSerializeWithDependencies(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo hello"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/",
						Output: 0,
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	edge := &Edge{
		node:        sourceNode,
		outputIndex: 0,
	}
	execNode.AddInput(edge)

	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 3 {
		t.Errorf("Expected 3 ops in definition, got %d", len(def.Def))
	}

	var unmarshaledExecOp pb.Op
	if err := unmarshaledExecOp.UnmarshalVT(def.Def[1]); err != nil {
		t.Fatalf("Failed to unmarshal exec op: %v", err)
	}

	if len(unmarshaledExecOp.Inputs) != 1 {
		t.Errorf("Expected 1 input in exec op, got %d", len(unmarshaledExecOp.Inputs))
	}

	if unmarshaledExecOp.Inputs[0].Digest == "" {
		t.Error("Expected input digest to be set")
	}

	if unmarshaledExecOp.Inputs[0].Index != 0 {
		t.Errorf("Expected input index 0, got %d", unmarshaledExecOp.Inputs[0].Index)
	}
}

func TestSerializeMultipleOutputs(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	copyOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/true"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/src",
						Output: 1,
					},
				},
			},
		},
	}
	copyNode := NewOpNode(copyOp, "test.lua", 2)
	edge := &Edge{
		node:        sourceNode,
		outputIndex: 1,
	}
	copyNode.AddInput(edge)

	state := NewStateWithOutput(copyNode, 1)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 3 {
		t.Errorf("Expected 3 ops in definition, got %d", len(def.Def))
	}

	var unmarshaledCopyOp pb.Op
	if err := unmarshaledCopyOp.UnmarshalVT(def.Def[1]); err != nil {
		t.Fatalf("Failed to unmarshal copy op: %v", err)
	}

	if unmarshaledCopyOp.Inputs[0].Index != 1 {
		t.Errorf("Expected input index 1, got %d", unmarshaledCopyOp.Inputs[0].Index)
	}
}

func TestSerializeWithProgressGroup(t *testing.T) {
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
		ProgressGroup: &pb.ProgressGroup{
			Id:   "test-group",
			Name: "Test Group",
		},
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
		if meta.ProgressGroup.Id != "test-group" {
			t.Errorf("Expected progress group id 'test-group', got '%s'", meta.ProgressGroup.Id)
		}
	}
}

func TestSerializeDAG(t *testing.T) {
	baseOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	baseNode := NewOpNode(baseOp, "test.lua", 1)

	copyOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/cp", "-r", "/src", "/dest"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/src",
						Output: 0,
					},
				},
			},
		},
	}
	copyNode := NewOpNode(copyOp, "test.lua", 2)
	copyNode.AddInput(NewEdge(baseNode, 0))

	rmOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/rm", "-rf", "/dest"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/src",
						Output: 0,
					},
				},
			},
		},
	}
	rmNode := NewOpNode(rmOp, "test.lua", 3)
	rmNode.AddInput(NewEdge(copyNode, 0))

	state := NewState(rmNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 4 {
		t.Errorf("Expected 4 ops in definition, got %d", len(def.Def))
	}

	digests := make(map[string]bool)
	for _, dt := range def.Def {
		var unmarshaledOp pb.Op
		if err := unmarshaledOp.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		for _, input := range unmarshaledOp.Inputs {
			if input.Digest != "" {
				digests[input.Digest] = true
			}
		}
	}

	if len(digests) != 3 {
		t.Errorf("Expected 3 unique input digests, got %d", len(digests))
	}
}

func TestSerializeDigestLinks(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)
	sourceDigest := sourceNode.Digest()

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/echo", "test"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/",
						Output: 0,
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))

	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	if len(unmarshaledExecOp.Inputs) != 1 {
		t.Errorf("Expected 1 input in exec op, got %d", len(unmarshaledExecOp.Inputs))
	}

	if unmarshaledExecOp.Inputs[0].Digest != string(sourceDigest) {
		t.Errorf("Expected input digest '%s', got '%s'", sourceDigest, unmarshaledExecOp.Inputs[0].Digest)
	}

	if unmarshaledExecOp.Inputs[0].Index != 0 {
		t.Errorf("Expected input index 0, got %d", unmarshaledExecOp.Inputs[0].Index)
	}
}

func TestSerializeSourceField(t *testing.T) {
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

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Error("Expected Source field to be initialized")
	}
}

func TestSerializeExecWithCacheMount(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "go build"},
				},
				Mounts: []*pb.Mount{
					{
						Dest:      "/cache",
						MountType: pb.MountType_CACHE,
						CacheOpt: &pb.CacheOpt{
							ID:      "mycache",
							Sharing: pb.CacheSharingOpt_SHARED,
						},
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 3 {
		t.Errorf("Expected 3 ops in definition, got %d", len(def.Def))
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	if len(unmarshaledExecOp.GetExec().Mounts) != 1 {
		t.Errorf("Expected 1 mount, got %d", len(unmarshaledExecOp.GetExec().Mounts))
	}

	mount := unmarshaledExecOp.GetExec().Mounts[0]
	if mount.Dest != "/cache" {
		t.Errorf("Expected mount dest '/cache', got '%s'", mount.Dest)
	}

	if mount.MountType != pb.MountType_CACHE {
		t.Errorf("Expected mount type CACHE, got %v", mount.MountType)
	}

	if mount.CacheOpt.ID != "mycache" {
		t.Errorf("Expected cache ID 'mycache', got '%s'", mount.CacheOpt.ID)
	}
}

func TestSerializeExecWithSecretMount(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "cat /run/secrets/mysecret"},
				},
				Mounts: []*pb.Mount{
					{
						Dest:      "/run/secrets/mysecret",
						MountType: pb.MountType_SECRET,
						SecretOpt: &pb.SecretOpt{
							ID:       "mysecret",
							Uid:      1000,
							Gid:      1000,
							Mode:     0600,
							Optional: false,
						},
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	mount := unmarshaledExecOp.GetExec().Mounts[0]
	if mount.Dest != "/run/secrets/mysecret" {
		t.Errorf("Expected mount dest '/run/secrets/mysecret', got '%s'", mount.Dest)
	}

	if mount.MountType != pb.MountType_SECRET {
		t.Errorf("Expected mount type SECRET, got %v", mount.MountType)
	}

	if mount.SecretOpt.ID != "mysecret" {
		t.Errorf("Expected secret ID 'mysecret', got '%s'", mount.SecretOpt.ID)
	}
}

func TestSerializeExecWithSSHMount(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "git clone"},
				},
				Mounts: []*pb.Mount{
					{
						Dest:      "/run/ssh",
						MountType: pb.MountType_SSH,
						SSHOpt: &pb.SSHOpt{
							ID:       "default",
							Uid:      0,
							Gid:      0,
							Mode:     0600,
							Optional: false,
						},
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	mount := unmarshaledExecOp.GetExec().Mounts[0]
	if mount.Dest != "/run/ssh" {
		t.Errorf("Expected mount dest '/run/ssh', got '%s'", mount.Dest)
	}

	if mount.MountType != pb.MountType_SSH {
		t.Errorf("Expected mount type SSH, got %v", mount.MountType)
	}
}

func TestSerializeExecWithTmpfsMount(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo test"},
				},
				Mounts: []*pb.Mount{
					{
						Dest:      "/tmp",
						MountType: pb.MountType_TMPFS,
						TmpfsOpt: &pb.TmpfsOpt{
							Size: 1024 * 1024 * 1024,
						},
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	mount := unmarshaledExecOp.GetExec().Mounts[0]
	if mount.Dest != "/tmp" {
		t.Errorf("Expected mount dest '/tmp', got '%s'", mount.Dest)
	}

	if mount.MountType != pb.MountType_TMPFS {
		t.Errorf("Expected mount type TMPFS, got %v", mount.MountType)
	}

	if mount.TmpfsOpt.Size != 1024*1024*1024 {
		t.Errorf("Expected tmpfs size 1073741824, got %d", mount.TmpfsOpt.Size)
	}
}

func TestSerializeExecWithMultipleMounts(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	execOp := &pb.Op{
		Inputs: []*pb.Input{
			{},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "build"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/",
						Output: 0,
					},
					{
						Dest:      "/cache",
						MountType: pb.MountType_CACHE,
						CacheOpt: &pb.CacheOpt{
							ID: "mycache",
						},
					},
					{
						Dest:      "/run/secrets/secret",
						MountType: pb.MountType_SECRET,
						SecretOpt: &pb.SecretOpt{
							ID: "mysecret",
						},
					},
					{
						Dest:      "/run/ssh",
						MountType: pb.MountType_SSH,
						SSHOpt: &pb.SSHOpt{
							ID: "default",
						},
					},
					{
						Dest:      "/tmp",
						MountType: pb.MountType_TMPFS,
						TmpfsOpt: &pb.TmpfsOpt{
							Size: 67108864,
						},
					},
				},
			},
		},
	}
	execNode := NewOpNode(execOp, "test.lua", 2)
	execNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(execNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledExecOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			t.Fatalf("Failed to unmarshal op: %v", err)
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	if len(unmarshaledExecOp.GetExec().Mounts) != 5 {
		t.Errorf("Expected 5 mounts, got %d", len(unmarshaledExecOp.GetExec().Mounts))
	}
}

func TestSerializeWithCustomDescription(t *testing.T) {
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

	meta := &pb.OpMetadata{
		Description: map[string]string{
			"llb.custom": "Building application",
		},
	}
	node.SetMetadata(meta)

	state := NewState(node)
	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 2 {
		t.Errorf("Expected 2 ops in definition, got %d", len(def.Def))
	}

	digest := node.Digest()
	metadata, ok := def.Metadata[digest.String()]
	if !ok {
		t.Fatal("Expected metadata for the op")
	}

	if metadata.Description == nil {
		t.Fatal("Expected description map in metadata")
	}

	if metadata.Description["llb.custom"] != "Building application" {
		t.Errorf("Expected custom description 'Building application', got '%s'", metadata.Description["llb.custom"])
	}
}

func TestSerializeWithBothMetadata(t *testing.T) {
	op := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "npm run build"},
				},
			},
		},
	}
	node := NewOpNode(op, "test.lua", 1)

	meta := &pb.OpMetadata{
		Description: map[string]string{
			"llb.custom": "Compiling TypeScript",
		},
		ProgressGroup: &pb.ProgressGroup{
			Id: "build",
		},
	}
	node.SetMetadata(meta)

	state := NewState(node)
	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 2 {
		t.Errorf("Expected 2 ops in definition, got %d", len(def.Def))
	}

	digest := node.Digest()
	metadata, ok := def.Metadata[digest.String()]
	if !ok {
		t.Fatal("Expected metadata for the op")
	}

	if metadata.Description == nil {
		t.Fatal("Expected description map in metadata")
	}

	if metadata.Description["llb.custom"] != "Compiling TypeScript" {
		t.Errorf("Expected custom description 'Compiling TypeScript', got '%s'", metadata.Description["llb.custom"])
	}

	if metadata.ProgressGroup == nil {
		t.Fatal("Expected progress group in metadata")
	}

	if metadata.ProgressGroup.Id != "build" {
		t.Errorf("Expected progress group id 'build', got '%s'", metadata.ProgressGroup.Id)
	}
}
