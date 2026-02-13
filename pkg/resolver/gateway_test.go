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
