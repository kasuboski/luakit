package luavm

import (
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
)

func TestPlatformSerialization(t *testing.T) {
	t.Run("platform serializes correctly", func(t *testing.T) {
		resetExportedState()
		t.Cleanup(resetExportedState)
		L := NewVM(nil)
		testVM = L
		t.Cleanup(func() { L.Close(); testVM = nil })

		script := `
			local p = bk.platform("linux", "arm64", "v8")
			local base = bk.image("ubuntu:24.04", { platform = p })
			bk.export(base)
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

	t.Run("platform string serializes correctly", func(t *testing.T) {
		resetExportedState()
		t.Cleanup(resetExportedState)
		L := NewVM(nil)
		testVM = L
		t.Cleanup(func() { L.Close(); testVM = nil })

		script := `
			local base = bk.image("alpine:3.19", { platform = "linux/amd64" })
			bk.export(base)
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

		if platform.Architecture != "amd64" {
			t.Errorf("Expected Architecture 'amd64', got %q", platform.Architecture)
		}
	})

	t.Run("platform table serializes correctly", func(t *testing.T) {
		resetExportedState()
		t.Cleanup(resetExportedState)
		L := NewVM(nil)
		testVM = L
		t.Cleanup(func() { L.Close(); testVM = nil })

		script := `
			local base = bk.image("ubuntu:24.04", { platform = { os = "darwin", arch = "arm64" } })
			bk.export(base)
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

		if platform.OS != "darwin" {
			t.Errorf("Expected OS 'darwin', got %q", platform.OS)
		}

		if platform.Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", platform.Architecture)
		}
	})
}

func TestStateWithPlatform(t *testing.T) {
	t.Run("State.WithPlatform creates new state", func(t *testing.T) {
		platform := &pb.Platform{
			OS:           "linux",
			Architecture: "arm64",
			Variant:      "v8",
		}

		opNode := dag.NewOpNode(&pb.Op{
			Op: &pb.Op_Source{
				Source: &pb.SourceOp{
					Identifier: "docker-image://ubuntu:24.04",
				},
			},
		}, "test.lua", 10)

		state := dag.NewState(opNode)
		stateWithPlatform := state.WithPlatform(platform)

		if stateWithPlatform == nil {
			t.Fatal("Expected non-nil state")
		}

		if stateWithPlatform.Platform() == nil {
			t.Fatal("Expected platform to be set")
		}

		if stateWithPlatform.Platform().OS != "linux" {
			t.Errorf("Expected OS 'linux', got %q", stateWithPlatform.Platform().OS)
		}

		if stateWithPlatform.Platform().Architecture != "arm64" {
			t.Errorf("Expected Architecture 'arm64', got %q", stateWithPlatform.Platform().Architecture)
		}

		if stateWithPlatform.Platform().Variant != "v8" {
			t.Errorf("Expected Variant 'v8', got %q", stateWithPlatform.Platform().Variant)
		}
	})

	t.Run("State.WithPlatform does not modify original state", func(t *testing.T) {
		platform := &pb.Platform{
			OS:           "linux",
			Architecture: "arm64",
		}

		opNode := dag.NewOpNode(&pb.Op{
			Op: &pb.Op_Source{
				Source: &pb.SourceOp{
					Identifier: "docker-image://ubuntu:24.04",
				},
			},
		}, "test.lua", 10)

		state := dag.NewState(opNode)
		stateWithPlatform := state.WithPlatform(platform)

		if state.Platform() != nil {
			t.Errorf("Expected original state platform to be nil, got %+v", state.Platform())
		}

		if stateWithPlatform.Platform() == nil {
			t.Fatal("Expected new state platform to be set")
		}
	})
}
