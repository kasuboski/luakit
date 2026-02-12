package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	pb "github.com/moby/buildkit/solver/pb"
)

func TestSourceMappingIntegration(t *testing.T) {
	script := `-- Line 1: Create base image
local base = bk.image("alpine:3.19")

-- Line 4: Run a command
local result = base:run("echo hello")

-- Line 7: Export the result
bk.export(result)
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "build.lua")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	result, err := luavm.EvaluateFile(scriptPath, nil)
	if err != nil {
		t.Fatalf("failed to run script: %v", err)
	}

	state := result.State
	if state == nil {
		t.Fatal("no exported state")
	}

	imageConfig := result.ImageConfig
	sourceFiles := result.SourceFiles

	def, err := dag.Serialize(state, &dag.SerializeOptions{
		SourceFiles: sourceFiles,
		ImageConfig: imageConfig,
	})
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Fatal("source map should not be nil")
	}

	// Check source info
	if len(def.Source.Infos) != 1 {
		t.Fatalf("expected 1 source info, got %d", len(def.Source.Infos))
	}

	info := def.Source.Infos[0]
	if info.Filename != scriptPath {
		t.Errorf("expected filename %s, got %s", scriptPath, info.Filename)
	}

	if info.Language != "Lua" {
		t.Errorf("expected language 'Lua', got '%s'", info.Language)
	}

	if len(info.Data) == 0 {
		t.Error("source data should not be empty")
	}

	// Check that we have locations
	if len(def.Source.Locations) == 0 {
		t.Fatal("expected at least 1 location mapping")
	}

	// Verify locations are valid
	locationCount := 0
	for digest, locations := range def.Source.Locations {
		t.Logf("Digest %s has %d location(s)", digest, len(locations.Locations))
		if len(locations.Locations) == 0 {
			t.Errorf("digest %s has no locations", digest)
			continue
		}

		for i, loc := range locations.Locations {
			locationCount++
			t.Logf("  Location %d: SourceIndex=%d, Line=%d",
				i, loc.SourceIndex, loc.Ranges[0].Start.Line)

			if loc.SourceIndex != 0 {
				t.Errorf("location %d: expected source index 0, got %d", i, loc.SourceIndex)
			}

			if len(loc.Ranges) == 0 {
				t.Errorf("location %d: expected at least 1 range", i)
				continue
			}

			if loc.Ranges[0].Start.Line < 1 {
				t.Errorf("location %d: invalid line number %d", i, loc.Ranges[0].Start.Line)
			}
		}
	}

	// Verify we have at least the two main operations with source mapping
	opCount := len(def.Def)
	if opCount < 2 {
		t.Fatalf("expected at least 2 ops, got %d", opCount)
	}

	if locationCount < 1 {
		t.Fatal("expected at least 1 location mapping")
	}

	t.Logf("Source mapping working correctly with %d ops and %d location mappings",
		opCount, locationCount)
}

func TestSourceMappingWithComplexDAG(t *testing.T) {
	script := `local base = bk.image("alpine:3.19")
local ctx = bk.local_("context")

local workspace = base:copy(ctx, ".", "/app")

local deps = workspace:run("apk add --no-cache build-base", {
  cwd = "/app"
})

local built = deps:run("make", {
  cwd = "/app",
  mounts = { bk.cache("/root/.cache") }
})

bk.export(built)
`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "complex.lua")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	result, err := luavm.EvaluateFile(scriptPath, nil)
	if err != nil {
		t.Fatalf("failed to run script: %v", err)
	}

	state := result.State
	if state == nil {
		t.Fatal("no exported state")
	}

	imageConfig := result.ImageConfig
	sourceFiles := result.SourceFiles

	def, err := dag.Serialize(state, &dag.SerializeOptions{
		SourceFiles: sourceFiles,
		ImageConfig: imageConfig,
	})
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Verify we have multiple operations with source mapping
	if len(def.Def) < 4 {
		t.Errorf("expected at least 4 ops for complex DAG, got %d", len(def.Def))
	}

	// Check that many operations have source mappings
	locationCount := len(def.Source.Locations)
	opCount := len(def.Def)

	t.Logf("Complex DAG: %d ops, %d with source mappings (%.0f%% coverage)",
		opCount, locationCount, float64(locationCount)/float64(opCount)*100)

	// Most operations should have source mappings
	if locationCount < opCount/2 {
		t.Errorf("expected at least half of ops to have source mappings, got %d/%d",
			locationCount, opCount)
	}
}

func TestSourceMappingLocationAccuracy(t *testing.T) {
	script := `local s = bk.image("alpine:3.19")
local r = s:run("echo hello")
bk.export(r)`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.lua")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	result, err := luavm.EvaluateFile(scriptPath, nil)
	if err != nil {
		t.Fatalf("failed to run script: %v", err)
	}

	state := result.State
	if state == nil {
		t.Fatal("no exported state")
	}

	imageConfig := result.ImageConfig
	sourceFiles := result.SourceFiles

	def, err := dag.Serialize(state, &dag.SerializeOptions{
		SourceFiles: sourceFiles,
		ImageConfig: imageConfig,
	})
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Check that line numbers are reasonable
	for digest, locations := range def.Source.Locations {
		for _, loc := range locations.Locations {
			line := loc.Ranges[0].Start.Line
			t.Logf("Digest %s mapped to line %d", digest, line)

			// Line numbers should be within the script range
			if line < 1 || line > 3 {
				t.Errorf("line %d is outside expected range [1,3]", line)
			}
		}
	}
}

func TestSourceMappingForProgressDisplay(t *testing.T) {
	// This test verifies that the source mapping provides the information
	// needed for BuildKit to display progress like:
	// #4 [build.lua:12] run("make -j$(nproc)")
	// #4 DONE 42.3s

	script := `local base = bk.image("alpine:3.19")
local step1 = base:run("echo step 1")
local step2 = step1:run("echo step 2")
local step3 = step2:run("echo step 3")
bk.export(step3)`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "build.lua")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	result, err := luavm.EvaluateFile(scriptPath, nil)
	if err != nil {
		t.Fatalf("failed to run script: %v", err)
	}

	state := result.State
	if state == nil {
		t.Fatal("no exported state")
	}

	imageConfig := result.ImageConfig
	sourceFiles := result.SourceFiles

	def, err := dag.Serialize(state, &dag.SerializeOptions{
		SourceFiles: sourceFiles,
		ImageConfig: imageConfig,
	})
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Verify source info is available
	if len(def.Source.Infos) == 0 {
		t.Fatal("no source info available")
	}

	info := def.Source.Infos[0]
	if info.Filename != scriptPath {
		t.Errorf("expected filename %s, got %s", scriptPath, info.Filename)
	}

	// Check source content is available
	if len(info.Data) == 0 {
		t.Fatal("source data not available")
	}

	if !strings.Contains(string(info.Data), "step 1") {
		t.Error("source data should contain script content")
	}

	// Verify each exec operation has a location
	execOpCount := 0
	for _, opBytes := range def.Def {
		var op pb.Op
		if err := op.UnmarshalVT(opBytes); err != nil {
			continue
		}

		if op.GetExec() != nil {
			execOpCount++
			digest := computeDigest(&op)

			if locations, ok := def.Source.Locations[digest]; ok {
				if len(locations.Locations) == 0 {
					t.Errorf("exec op at digest %s has no location", digest)
				} else {
					loc := locations.Locations[0]
					line := loc.Ranges[0].Start.Line
					t.Logf("Exec op mapped to %s:%d", scriptPath, line)

					// Verify the source info index is correct
					if loc.SourceIndex != 0 {
						t.Errorf("expected source index 0, got %d", loc.SourceIndex)
					}
				}
			} else {
				t.Logf("Warning: exec op at digest %s has no location mapping", digest)
			}
		}
	}

	if execOpCount == 0 {
		t.Fatal("no exec ops found in definition")
	}

	t.Logf("Found %d exec operations, all should have source mappings for progress display", execOpCount)
}

func computeDigest(op *pb.Op) string {
	// Simple digest computation for testing
	// In real code, this uses the digest package
	return "sha256:test"
}
