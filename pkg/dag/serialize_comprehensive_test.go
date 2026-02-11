package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func TestSerializeWithComplexImageConfig(t *testing.T) {
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

	imageConfig := &dockerspec.DockerOCIImage{}
	imageConfig.OS = "linux"
	imageConfig.Architecture = "arm64"
	imageConfig.Variant = "v8"
	imageConfig.Config.Env = []string{
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	}
	imageConfig.Config.Cmd = []string{"/bin/sh"}
	imageConfig.Config.Entrypoint = []string{"/entrypoint.sh"}
	imageConfig.Config.WorkingDir = "/app"
	imageConfig.Config.User = "appuser"
	imageConfig.Config.ExposedPorts = map[string]struct{}{
		"8080/tcp": {},
		"9090/udp": {},
	}
	imageConfig.Config.Labels = map[string]string{
		"org.opencontainers.image.title":       "Test App",
		"org.opencontainers.image.description": "A test application",
		"version":                              "1.0.0",
	}

	opts := &SerializeOptions{
		ImageConfig: imageConfig,
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Metadata) == 0 {
		t.Error("Expected metadata entries")
	}

	foundConfig := false
	for _, meta := range def.Metadata {
		if meta != nil && meta.Description != nil {
			if _, ok := meta.Description["containerimage.config"]; ok {
				foundConfig = true
				break
			}
		}
	}

	if !foundConfig {
		t.Error("Expected to find image config in metadata")
	}
}

func TestSerializeWithMultipleSourceFiles(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	opts := &SerializeOptions{
		SourceFiles: map[string][]byte{
			"build.lua":           []byte("print('hello')"),
			"scripts/setup.sh":    []byte("#!/bin/sh\necho setup"),
			"config/default.json": []byte(`{"key": "value"}`),
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

func TestSerializeWithEmptySourceFiles(t *testing.T) {
	op := NewOpNode(&pb.Op{}, "test.lua", 1)
	state := NewState(op)

	opts := &SerializeOptions{
		SourceFiles: map[string][]byte{},
	}

	def, err := Serialize(state, opts)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Error("Expected Source to be initialized")
	}
}

func TestSerializeExecWithAllMetaFields(t *testing.T) {
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
			{Digest: string(sourceNode.Digest()), Index: 0},
		},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo test"},
					Env:  []string{"PATH=/usr/bin", "HOME=/root"},
					Cwd:  "/workspace",
					User: "builder",
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
			continue
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	if unmarshaledExecOp == nil {
		t.Fatal("Expected to find ExecOp")
	}

	meta := unmarshaledExecOp.GetExec().Meta
	if meta.Cwd != "/workspace" {
		t.Errorf("Expected cwd '/workspace', got '%s'", meta.Cwd)
	}

	if meta.User != "builder" {
		t.Errorf("Expected user 'builder', got '%s'", meta.User)
	}

	if len(meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(meta.Env))
	}
}

func TestSerializeFileOpWithMultipleActions(t *testing.T) {
	sourceOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	sourceNode := NewOpNode(sourceOp, "test.lua", 1)

	fileOp := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(sourceNode.Digest()), Index: 0},
		},
		Op: &pb.Op_File{
			File: &pb.FileOp{
				Actions: []*pb.FileAction{
					{
						Action: &pb.FileAction_Mkdir{
							Mkdir: &pb.FileActionMkDir{
								Path:        "/app",
								Mode:        0755,
								MakeParents: true,
							},
						},
					},
					{
						Action: &pb.FileAction_Mkfile{
							Mkfile: &pb.FileActionMkFile{
								Path: "/app/config.json",
								Data: []byte(`{"key": "value"}`),
								Mode: 0644,
							},
						},
					},
					{
						Action: &pb.FileAction_Symlink{
							Symlink: &pb.FileActionSymlink{
								Oldpath: "/app/config.json",
								Newpath: "/app/config.link",
							},
						},
					},
				},
			},
		},
	}
	fileNode := NewOpNode(fileOp, "test.lua", 2)
	fileNode.AddInput(NewEdge(sourceNode, 0))
	state := NewState(fileNode)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var unmarshaledFileOp *pb.Op
	for _, dt := range def.Def {
		op := &pb.Op{}
		if err := op.UnmarshalVT(dt); err != nil {
			continue
		}
		if op.GetFile() != nil {
			unmarshaledFileOp = op
			break
		}
	}

	if unmarshaledFileOp == nil {
		t.Fatal("Expected to find FileOp")
	}

	actions := unmarshaledFileOp.GetFile().Actions
	if len(actions) != 3 {
		t.Errorf("Expected 3 file actions, got %d", len(actions))
	}

	if actions[0].GetMkdir() == nil {
		t.Error("Expected first action to be Mkdir")
	}

	if actions[1].GetMkfile() == nil {
		t.Error("Expected second action to be Mkfile")
	}

	if actions[2].GetSymlink() == nil {
		t.Error("Expected third action to be Symlink")
	}
}

func TestSerializeComplexDAGWithMultiplePaths(t *testing.T) {
	baseOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	base := NewOpNode(baseOp, "test.lua", 1)

	branch1Op := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(base.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "branch1"}},
			},
		},
	}
	branch1 := NewOpNode(branch1Op, "test.lua", 2)
	branch1.AddInput(NewEdge(base, 0))

	branch2Op := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(base.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "branch2"}},
			},
		},
	}
	branch2 := NewOpNode(branch2Op, "test.lua", 3)
	branch2.AddInput(NewEdge(base, 0))

	sharedOp := &pb.Op{
		Inputs: []*pb.Input{{Digest: string(base.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "shared"}},
			},
		},
	}
	shared := NewOpNode(sharedOp, "test.lua", 4)
	shared.AddInput(NewEdge(base, 0))

	merge1Op := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(branch1.Digest()), Index: 0},
			{Digest: string(shared.Digest()), Index: 0},
		},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	merge1 := NewOpNode(merge1Op, "test.lua", 5)
	merge1.AddInput(NewEdge(branch1, 0))
	merge1.AddInput(NewEdge(shared, 0))

	merge2Op := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(branch2.Digest()), Index: 0},
			{Digest: string(shared.Digest()), Index: 0},
		},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	merge2 := NewOpNode(merge2Op, "test.lua", 6)
	merge2.AddInput(NewEdge(branch2, 0))
	merge2.AddInput(NewEdge(shared, 0))

	finalMergeOp := &pb.Op{
		Inputs: []*pb.Input{
			{Digest: string(merge1.Digest()), Index: 0},
			{Digest: string(merge2.Digest()), Index: 0},
		},
		Op: &pb.Op_Merge{
			Merge: &pb.MergeOp{},
		},
	}
	finalMerge := NewOpNode(finalMergeOp, "test.lua", 7)
	finalMerge.AddInput(NewEdge(merge1, 0))
	finalMerge.AddInput(NewEdge(merge2, 0))

	state := NewState(finalMerge)

	def, err := Serialize(state, nil)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 8 {
		t.Errorf("Expected 8 ops in definition, got %d", len(def.Def))
	}
}

func TestSerializeWithAllMountTypes(t *testing.T) {
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
		Inputs: []*pb.Input{{Digest: string(sourceNode.Digest()), Index: 0}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{Args: []string{"/bin/sh", "-c", "test"}},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/root",
						Output: 0,
					},
					{
						Dest:      "/cache",
						MountType: pb.MountType_CACHE,
						CacheOpt:  &pb.CacheOpt{ID: "mycache"},
					},
					{
						Dest:      "/run/secrets/secret",
						MountType: pb.MountType_SECRET,
						SecretOpt: &pb.SecretOpt{ID: "mysecret"},
					},
					{
						Dest:      "/run/ssh",
						MountType: pb.MountType_SSH,
						SSHOpt:    &pb.SSHOpt{ID: "default"},
					},
					{
						Dest:      "/tmp",
						MountType: pb.MountType_TMPFS,
						TmpfsOpt:  &pb.TmpfsOpt{Size: 1024 * 1024},
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
			continue
		}
		if op.GetExec() != nil {
			unmarshaledExecOp = op
			break
		}
	}

	if unmarshaledExecOp == nil {
		t.Fatal("Expected to find ExecOp")
	}

	mounts := unmarshaledExecOp.GetExec().Mounts
	if len(mounts) != 5 {
		t.Errorf("Expected 5 mounts, got %d", len(mounts))
	}

	if mounts[1].MountType != pb.MountType_CACHE {
		t.Error("Expected second mount to be CACHE")
	}

	if mounts[2].MountType != pb.MountType_SECRET {
		t.Error("Expected third mount to be SECRET")
	}

	if mounts[3].MountType != pb.MountType_SSH {
		t.Error("Expected fourth mount to be SSH")
	}

	if mounts[4].MountType != pb.MountType_TMPFS {
		t.Error("Expected fifth mount to be TMPFS")
	}
}

func TestSerializeDeterminism(t *testing.T) {
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
	if err1 != nil {
		t.Fatalf("Failed to serialize first time: %v", err1)
	}

	def2, err2 := Serialize(state, nil)
	if err2 != nil {
		t.Fatalf("Failed to serialize second time: %v", err2)
	}

	if len(def1.Def) != len(def2.Def) {
		t.Error("Expected same number of ops in both serializations")
	}

	for i := range def1.Def {
		if string(def1.Def[i]) != string(def2.Def[i]) {
			t.Errorf("Expected op %d to be identical", i)
		}
	}
}
