package luavm

import (
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
	lua "github.com/yuin/gopher-lua"
)

func TestParsePlatformString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *pb.Platform
	}{
		{
			name:  "simple os/arch",
			input: "linux/amd64",
			expected: &pb.Platform{
				OS:           "linux",
				Architecture: "amd64",
			},
		},
		{
			name:  "os/arch with variant",
			input: "linux/arm64/v8",
			expected: &pb.Platform{
				OS:           "linux",
				Architecture: "arm64",
				Variant:      "v8",
			},
		},
		{
			name:  "arm64 only",
			input: "arm64",
			expected: &pb.Platform{
				Architecture: "arm64",
			},
		},
		{
			name:  "amd64 only",
			input: "amd64",
			expected: &pb.Platform{
				Architecture: "amd64",
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePlatformString(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected non-nil platform")
			}

			if result.OS != tt.expected.OS {
				t.Errorf("OS: expected %q, got %q", tt.expected.OS, result.OS)
			}

			if result.Architecture != tt.expected.Architecture {
				t.Errorf("Architecture: expected %q, got %q", tt.expected.Architecture, result.Architecture)
			}

			if result.Variant != tt.expected.Variant {
				t.Errorf("Variant: expected %q, got %q", tt.expected.Variant, result.Variant)
			}
		})
	}
}

func TestBkPlatform(t *testing.T) {
	t.Run("single argument platform string", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("linux/amd64")`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("p")
		if result.Type() != lua.LTUserData {
			t.Fatalf("Expected userdata, got %v", result.Type())
		}

		ud := result.(*lua.LUserData)
		platform, ok := ud.Value.(*pb.Platform)
		if !ok {
			t.Fatal("Expected Platform value")
		}

		if platform.OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", platform.OS)
		}

		if platform.Architecture != "amd64" {
			t.Errorf("Expected Architecture 'amd64', got %q", platform.Architecture)
		}
	})

	t.Run("three arguments os, arch, variant", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("linux", "arm64", "v8")`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("p")
		ud := result.(*lua.LUserData)
		platform := ud.Value.(*pb.Platform)

		if platform.OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", platform.OS)
		}

		if platform.Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", platform.Architecture)
		}

		if platform.Variant != "v8" {
			t.Errorf("Expected Variant 'v8', got %q", platform.Variant)
		}
	})

	t.Run("two arguments os, arch", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("darwin", "arm64")`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("p")
		ud := result.(*lua.LUserData)
		platform := ud.Value.(*pb.Platform)

		if platform.OS != "darwin" {
			t.Errorf("Expected OS 'darwin', got %q", platform.OS)
		}

		if platform.Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", platform.Architecture)
		}

		if platform.Variant != "" {
			t.Errorf("Expected empty Variant, got %q", platform.Variant)
		}
	})

	t.Run("no arguments error", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		err := L.DoString(`p = bk.platform()`)
		if err == nil {
			t.Fatal("Expected error for no arguments")
		}

		if err.Error() == "" {
			t.Fatal("Expected non-empty error message")
		}
	})

	t.Run("platform to string", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("linux/arm64/v8"); s = tostring(p)`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("s")
		if result.Type() != lua.LTString {
			t.Fatalf("Expected string, got %v", result.Type())
		}

		str := result.String()
		if str != "linux/arm64/v8" {
			t.Errorf("Expected 'linux/arm64/v8', got %q", str)
		}
	})

	t.Run("platform to string without variant", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("linux/amd64"); s = tostring(p)`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("s")
		if result.Type() != lua.LTString {
			t.Fatalf("Expected string, got %v", result.Type())
		}

		str := result.String()
		if str != "linux/amd64" {
			t.Errorf("Expected 'linux/amd64', got %q", str)
		}
	})
}

func TestImageWithPlatformString(t *testing.T) {
	t.Run("platform string in image", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`base = bk.image("ubuntu:24.04", { platform = "linux/arm64" })`); err != nil {
			t.Fatalf("Failed to call bk.image with platform: %v", err)
		}

		result := L.GetGlobal("base")
		ud := result.(*lua.LUserData)
		state := ud.Value.(*dag.State)

		if state.Platform() == nil {
			t.Fatal("Expected non-nil platform")
		}

		if state.Platform().OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", state.Platform().OS)
		}

		if state.Platform().Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", state.Platform().Architecture)
		}
	})

	t.Run("platform object in image", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`p = bk.platform("linux", "arm64", "v8"); base = bk.image("ubuntu:24.04", { platform = p })`); err != nil {
			t.Fatalf("Failed to call bk.image with platform: %v", err)
		}

		result := L.GetGlobal("base")
		ud := result.(*lua.LUserData)
		state := ud.Value.(*dag.State)

		if state.Platform() == nil {
			t.Fatal("Expected non-nil platform")
		}

		if state.Platform().OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", state.Platform().OS)
		}

		if state.Platform().Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", state.Platform().Architecture)
		}

		if state.Platform().Variant != "v8" {
			t.Errorf("Expected Variant 'v8', got %q", state.Platform().Variant)
		}
	})

	t.Run("platform table in image inline", func(t *testing.T) {
		L := NewVM(nil)
		t.Cleanup(func() { L.Close() })

		if err := L.DoString(`base = bk.image("ubuntu:24.04", { platform = { os = "linux", arch = "arm64", variant = "v8" } })`); err != nil {
			t.Fatalf("Failed to call bk.image with platform: %v", err)
		}

		result := L.GetGlobal("base")
		ud := result.(*lua.LUserData)
		state := ud.Value.(*dag.State)

		if state.Platform() == nil {
			t.Fatal("Expected non-nil platform")
		}

		if state.Platform().OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", state.Platform().OS)
		}

		if state.Platform().Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", state.Platform().Architecture)
		}

		if state.Platform().Variant != "v8" {
			t.Errorf("Expected Variant 'v8', got %q", state.Platform().Variant)
		}
	})
}
