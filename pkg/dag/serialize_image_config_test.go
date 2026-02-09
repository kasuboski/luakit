package dag

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

func TestSerializeWithImageConfig(t *testing.T) {
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
	imageConfig.Architecture = "amd64"
	imageConfig.Config.Env = []string{"PATH=/usr/bin", "NODE_ENV=production"}
	imageConfig.Config.WorkingDir = "/app"
	imageConfig.Config.User = "appuser"
	imageConfig.Config.Entrypoint = []string{"/bin/sh"}
	imageConfig.Config.Cmd = []string{"-c", "echo hello"}
	imageConfig.Config.ExposedPorts = map[string]struct{}{"8080/tcp": {}}
	imageConfig.Config.Labels = map[string]string{
		"org.opencontainers.image.title":       "Test Image",
		"org.opencontainers.image.description": "A test image",
	}

	def, err := Serialize(state, &SerializeOptions{
		ImageConfig: imageConfig,
	})
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(def.Def) != 1 {
		t.Errorf("Expected 1 op in definition, got %d", len(def.Def))
	}

	digest := node.Digest()
	metadata, ok := def.Metadata[digest.String()]
	if !ok {
		t.Fatal("Expected metadata for the final op")
	}

	if metadata.Description == nil {
		t.Fatal("Expected description in metadata")
	}

	configStr, ok := metadata.Description[exptypes.ExporterImageConfigKey]
	if !ok {
		t.Fatal("Expected image config in description")
	}

	var unmarshaledConfig dockerspec.DockerOCIImage
	if err := json.Unmarshal([]byte(configStr), &unmarshaledConfig); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if unmarshaledConfig.OS != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", unmarshaledConfig.OS)
	}

	if unmarshaledConfig.Architecture != "amd64" {
		t.Errorf("Expected architecture 'amd64', got '%s'", unmarshaledConfig.Architecture)
	}

	if len(unmarshaledConfig.Config.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(unmarshaledConfig.Config.Env))
	}

	foundPath := false
	foundNodeEnv := false
	for _, env := range unmarshaledConfig.Config.Env {
		if strings.HasPrefix(env, "PATH=") {
			foundPath = true
		}
		if strings.HasPrefix(env, "NODE_ENV=") {
			foundNodeEnv = true
		}
	}

	if !foundPath {
		t.Error("Expected PATH env var")
	}

	if !foundNodeEnv {
		t.Error("Expected NODE_ENV env var")
	}

	if unmarshaledConfig.Config.WorkingDir != "/app" {
		t.Errorf("Expected working dir '/app', got '%s'", unmarshaledConfig.Config.WorkingDir)
	}

	if unmarshaledConfig.Config.User != "appuser" {
		t.Errorf("Expected user 'appuser', got '%s'", unmarshaledConfig.Config.User)
	}

	if len(unmarshaledConfig.Config.Entrypoint) != 1 {
		t.Errorf("Expected 1 entrypoint element, got %d", len(unmarshaledConfig.Config.Entrypoint))
	}

	if unmarshaledConfig.Config.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Expected entrypoint '/bin/sh', got '%s'", unmarshaledConfig.Config.Entrypoint[0])
	}

	if len(unmarshaledConfig.Config.Cmd) != 2 {
		t.Errorf("Expected 2 cmd elements, got %d", len(unmarshaledConfig.Config.Cmd))
	}

	if unmarshaledConfig.Config.Cmd[0] != "-c" {
		t.Errorf("Expected cmd[0] '-c', got '%s'", unmarshaledConfig.Config.Cmd[0])
	}

	if len(unmarshaledConfig.Config.ExposedPorts) != 1 {
		t.Errorf("Expected 1 exposed port, got %d", len(unmarshaledConfig.Config.ExposedPorts))
	}

	if _, ok := unmarshaledConfig.Config.ExposedPorts["8080/tcp"]; !ok {
		t.Error("Expected exposed port '8080/tcp'")
	}

	if len(unmarshaledConfig.Config.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(unmarshaledConfig.Config.Labels))
	}

	if unmarshaledConfig.Config.Labels["org.opencontainers.image.title"] != "Test Image" {
		t.Errorf("Expected label 'Test Image', got '%s'", unmarshaledConfig.Config.Labels["org.opencontainers.image.title"])
	}
}

func TestSerializeWithoutImageConfig(t *testing.T) {
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

	if len(def.Def) != 1 {
		t.Errorf("Expected 1 op in definition, got %d", len(def.Def))
	}

	digest := node.Digest()
	metadata, ok := def.Metadata[digest.String()]
	if ok && metadata.Description != nil {
		if _, ok := metadata.Description[exptypes.ExporterImageConfigKey]; ok {
			t.Error("Expected no image config in description when none was provided")
		}
	}
}
