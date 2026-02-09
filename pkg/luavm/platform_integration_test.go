package luavm

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	lua "github.com/yuin/gopher-lua"
)

func TestPlatformAPIIntegration(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	t.Run("comprehensive platform test", func(t *testing.T) {
		resetExportedState()

		script := `
			-- Create platform using bk.platform function
			local arm64 = bk.platform("linux", "arm64", "v8")
			local amd64 = bk.platform("linux", "amd64")
			local darwin_arm = bk.platform("darwin", "arm64")

			-- Use platforms with images
			local base_arm64 = bk.image("ubuntu:24.04", { platform = arm64 })
			local base_amd64 = bk.image("ubuntu:24.04", { platform = amd64 })
			local base_darwin = bk.image("ubuntu:24.04", { platform = darwin_arm })

			-- Use platform string
			local base_arm64_str = bk.image("alpine:3.19", { platform = "linux/arm64" })

			-- Use platform table
			local base_amd64_tbl = bk.image("alpine:3.19", { platform = { os = "linux", arch = "amd64" } })

			-- Export one state
			bk.export(base_arm64)
		`

		if err := L.DoString(script); err != nil {
			t.Fatalf("Failed to execute script: %v", err)
		}

		state := GetExportedState()
		if state == nil {
			t.Fatal("Expected exported state")
		}

		platform := state.Platform()
		if platform == nil {
			t.Fatal("Expected platform to be set")
		}

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

	t.Run("platform immutability", func(t *testing.T) {
		resetExportedState()

		script := `
			local p1 = bk.platform("linux", "amd64")
			local base1 = bk.image("ubuntu:24.04", { platform = p1 })
			
			-- Changing platform shouldn't affect existing state
			local p2 = bk.platform("linux", "arm64")
			local base2 = bk.image("ubuntu:24.04", { platform = p2 })
			
			bk.export(base1)
		`

		if err := L.DoString(script); err != nil {
			t.Fatalf("Failed to execute script: %v", err)
		}

		state := GetExportedState()
		platform := state.Platform()

		if platform.Architecture != "amd64" {
			t.Errorf("Expected Architecture 'amd64', got %q", platform.Architecture)
		}
	})

	t.Run("different platforms for different states", func(t *testing.T) {
		resetExportedState()

		script := `
			local linux_amd64 = bk.image("ubuntu:24.04", { platform = "linux/amd64" })
			local linux_arm64 = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
			
			-- Both should have different platforms
			bk.export(linux_amd64)
		`

		if err := L.DoString(script); err != nil {
			t.Fatalf("Failed to execute script: %v", err)
		}

		state := GetExportedState()
		platform := state.Platform()

		if platform.Architecture != "amd64" {
			t.Errorf("Expected Architecture 'amd64', got %q", platform.Architecture)
		}
	})
}

func TestPlatformEdgeCases(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	t.Run("platform with os and arch", func(t *testing.T) {
		if err := L.DoString(`p = bk.platform("linux", "arm64")`); err != nil {
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
	})

	t.Run("platform with only arch", func(t *testing.T) {
		if err := L.DoString(`p = bk.platform("amd64")`); err != nil {
			t.Fatalf("Failed to call bk.platform: %v", err)
		}

		result := L.GetGlobal("p")
		ud := result.(*lua.LUserData)
		platform := ud.Value.(*pb.Platform)

		if platform.Architecture != "amd64" {
			t.Errorf("Expected Architecture 'amd64', got %q", platform.Architecture)
		}

		if platform.OS != "" {
			t.Errorf("Expected empty OS, got %q", platform.OS)
		}
	})

	t.Run("image without platform", func(t *testing.T) {
		resetExportedState()

		if err := L.DoString(`
			local base = bk.image("ubuntu:24.04")
			bk.export(base)
		`); err != nil {
			t.Fatalf("Failed to execute script: %v", err)
		}

		state := GetExportedState()
		if state.Platform() != nil {
			t.Errorf("Expected nil platform when not specified, got %+v", state.Platform())
		}
	})
}
