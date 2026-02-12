package resolver

import (
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestDefaultPlatform(t *testing.T) {
	platform := DefaultPlatform()

	if platform.OS == "" {
		t.Error("Expected non-empty OS")
	}

	if platform.Architecture == "" {
		t.Error("Expected non-empty Architecture")
	}
}

func TestImageConfig(t *testing.T) {
	config := &ImageConfig{
		Ref:    "alpine:3.19",
		Digest: "sha256:abc123",
		Config: &ocispec.Image{
			Config: ocispec.ImageConfig{
				Env: []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			},
		},
	}

	if config.Ref != "alpine:3.19" {
		t.Errorf("Expected ref 'alpine:3.19', got '%s'", config.Ref)
	}

	if config.Digest != "sha256:abc123" {
		t.Errorf("Expected digest 'sha256:abc123', got '%s'", config.Digest)
	}

	if len(config.Config.Config.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(config.Config.Config.Env))
	}

	if config.Config.Config.Env[0] != "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" {
		t.Errorf("Unexpected env var: %s", config.Config.Config.Env[0])
	}
}
