package resolver

import (
	"testing"
)

func TestGatewayResolverPrefixStripping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docker-image://alpine:3.19", "alpine:3.19"},
		{"oci-layout://myimage:latest", "myimage:latest"},
		{"alpine:3.19", "alpine:3.19"},
		{"docker.io/library/alpine:3.19", "docker.io/library/alpine:3.19"},
	}

	for _, tt := range tests {
		result := tt.input
		result = stripPrefix(result)
		if result != tt.expected {
			t.Errorf("stripPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func stripPrefix(ref string) string {
	ref = stripDockerImagePrefix(ref)
	ref = stripOCILayoutPrefix(ref)
	return ref
}

func stripDockerImagePrefix(ref string) string {
	const prefix = "docker-image://"
	if len(ref) >= len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ref
}

func stripOCILayoutPrefix(ref string) string {
	const prefix = "oci-layout://"
	if len(ref) >= len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ref
}
