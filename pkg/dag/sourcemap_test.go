package dag

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestSourceMapBuilder_AddFile(t *testing.T) {
	smb := NewSourceMapBuilder()

	luaSource := []byte("local s = bk.image(\"alpine:latest\")\n")

	idx1 := smb.AddFile("test.lua", luaSource)
	if idx1 != 0 {
		t.Errorf("first file index should be 0, got %d", idx1)
	}

	idx2 := smb.AddFile("test.lua", luaSource)
	if idx2 != 0 {
		t.Errorf("adding same file again should return same index, got %d", idx2)
	}

	idx3 := smb.AddFile("other.lua", []byte("print('hello')"))
	if idx3 != 1 {
		t.Errorf("second unique file index should be 1, got %d", idx3)
	}

	source := smb.Build()
	if len(source.Infos) != 2 {
		t.Errorf("expected 2 source infos, got %d", len(source.Infos))
	}

	if source.Infos[0].Filename != "test.lua" {
		t.Errorf("expected filename 'test.lua', got '%s'", source.Infos[0].Filename)
	}

	if source.Infos[0].Language != "Lua" {
		t.Errorf("expected language 'Lua', got '%s'", source.Infos[0].Language)
	}
}

func TestSourceMapBuilder_AddLocation(t *testing.T) {
	smb := NewSourceMapBuilder()

	luaSource := []byte("local s = bk.image(\"alpine:latest\")\n")
	smb.AddFile("test.lua", luaSource)

	digest := "sha256:abc123"
	smb.AddLocation(digest, "test.lua", 10)

	source := smb.Build()
	if len(source.Locations) != 1 {
		t.Errorf("expected 1 location, got %d", len(source.Locations))
	}

	locations, ok := source.Locations[digest]
	if !ok {
		t.Errorf("digest %s not found in locations", digest)
	}

	if len(locations.Locations) != 1 {
		t.Errorf("expected 1 location entry, got %d", len(locations.Locations))
	}

	loc := locations.Locations[0]
	if loc.SourceIndex != 0 {
		t.Errorf("expected source index 0, got %d", loc.SourceIndex)
	}

	if len(loc.Ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(loc.Ranges))
	}

	if loc.Ranges[0].Start.Line != 10 {
		t.Errorf("expected start line 10, got %d", loc.Ranges[0].Start.Line)
	}

	if loc.Ranges[0].End.Line != 10 {
		t.Errorf("expected end line 10, got %d", loc.Ranges[0].End.Line)
	}
}

func TestSourceMapBuilder_Build(t *testing.T) {
	smb := NewSourceMapBuilder()

	luaSource := []byte("local s = bk.image(\"alpine:latest\")\n")
	smb.AddFile("test.lua", luaSource)

	digest1 := "sha256:abc123"
	digest2 := "sha256:def456"

	smb.AddLocation(digest1, "test.lua", 10)
	smb.AddLocation(digest2, "test.lua", 15)

	source := smb.Build()

	if len(source.Infos) != 1 {
		t.Errorf("expected 1 source info, got %d", len(source.Infos))
	}

	if len(source.Locations) != 2 {
		t.Errorf("expected 2 locations, got %d", len(source.Locations))
	}

	if _, ok := source.Locations[digest1]; !ok {
		t.Errorf("digest %s not found", digest1)
	}

	if _, ok := source.Locations[digest2]; !ok {
		t.Errorf("digest %s not found", digest2)
	}
}

func TestSourceMapBuilder_MultipleLocations(t *testing.T) {
	smb := NewSourceMapBuilder()

	luaSource := []byte("local s = bk.image(\"alpine:latest\")\n")
	smb.AddFile("test.lua", luaSource)

	digest := "sha256:abc123"

	smb.AddLocation(digest, "test.lua", 10)
	smb.AddLocation(digest, "test.lua", 15)

	source := smb.Build()

	locations, ok := source.Locations[digest]
	if !ok {
		t.Errorf("digest %s not found", digest)
	}

	if len(locations.Locations) != 2 {
		t.Errorf("expected 2 location entries, got %d", len(locations.Locations))
	}
}

func TestSerialize_WithSourceMapping(t *testing.T) {
	sourceFiles := map[string][]byte{
		"test.lua": []byte("local s = bk.image(\"alpine:latest\")\n"),
	}

	op := &pb.Op{
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}

	node := NewOpNode(op, "test.lua", 5)
	state := NewState(node)

	t.Logf("Node LuaFile: %s, LuaLine: %d", node.LuaFile(), node.LuaLine())

	def, err := Serialize(state, &SerializeOptions{
		SourceFiles: sourceFiles,
	})
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	if def.Source == nil {
		t.Errorf("source map should not be nil")
	}

	t.Logf("SourceInfos count: %d", len(def.Source.Infos))
	t.Logf("Locations count: %d", len(def.Source.Locations))

	if len(def.Source.Infos) != 1 {
		t.Errorf("expected 1 source info, got %d", len(def.Source.Infos))
	}

	if len(def.Source.Locations) != 1 {
		t.Errorf("expected 1 location, got %d", len(def.Source.Locations))
	}

	// Verify source content is preserved
	info := def.Source.Infos[0]
	if info.Filename != "test.lua" {
		t.Errorf("expected filename 'test.lua', got '%s'", info.Filename)
	}

	if info.Language != "Lua" {
		t.Errorf("expected language 'Lua', got '%s'", info.Language)
	}

	if string(info.Data) != "local s = bk.image(\"alpine:latest\")\n" {
		t.Errorf("source data mismatch")
	}

	// Verify location points to correct line
	digest := string(node.Digest())
	locations, ok := def.Source.Locations[digest]
	if !ok {
		t.Errorf("digest %s not found in locations", digest)
		return
	}

	if len(locations.Locations) != 1 {
		t.Errorf("expected 1 location entry, got %d", len(locations.Locations))
		return
	}

	loc := locations.Locations[0]
	if loc.SourceIndex != 0 {
		t.Errorf("expected source index 0, got %d", loc.SourceIndex)
	}

	if len(loc.Ranges) != 1 {
		t.Errorf("expected 1 range, got %d", len(loc.Ranges))
		return
	}

	if loc.Ranges[0].Start.Line != 5 {
		t.Errorf("expected start line 5, got %d", loc.Ranges[0].Start.Line)
	}

	if loc.Ranges[0].End.Line != 5 {
		t.Errorf("expected end line 5, got %d", loc.Ranges[0].End.Line)
	}
}
