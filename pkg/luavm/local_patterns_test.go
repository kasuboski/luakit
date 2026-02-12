package luavm

import (
	"testing"
)

func TestBkLocalWithPatterns(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			include = {"*.go", "*.mod", "*.sum"},
			exclude = {"*_test.go", "vendor/"},
			shared_key_hint = "go-sources"
		})
		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "local://context" {
		t.Errorf("Expected identifier 'local://context', got '%s'", sourceOp.Identifier)
	}

	if sourceOp.Attrs["includepattern0"] != "*.go" {
		t.Errorf("Expected includepattern0 '*.go', got '%s'", sourceOp.Attrs["includepattern0"])
	}

	if sourceOp.Attrs["includepattern1"] != "*.mod" {
		t.Errorf("Expected includepattern1 '*.mod', got '%s'", sourceOp.Attrs["includepattern1"])
	}

	if sourceOp.Attrs["includepattern2"] != "*.sum" {
		t.Errorf("Expected includepattern2 '*.sum', got '%s'", sourceOp.Attrs["includepattern2"])
	}

	if sourceOp.Attrs["excludepattern0"] != "*_test.go" {
		t.Errorf("Expected excludepattern0 '*_test.go', got '%s'", sourceOp.Attrs["excludepattern0"])
	}

	if sourceOp.Attrs["excludepattern1"] != "vendor/" {
		t.Errorf("Expected excludepattern1 'vendor/', got '%s'", sourceOp.Attrs["excludepattern1"])
	}

	if sourceOp.Attrs["sharedkeyhint"] != "go-sources" {
		t.Errorf("Expected sharedkeyhint 'go-sources', got '%s'", sourceOp.Attrs["sharedkeyhint"])
	}
}

func TestBkLocalWithIncludeOnly(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			include = {"*.txt"}
		})
		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	sourceOp := state.Op().Op().GetSource()

	if len(sourceOp.Attrs) != 1 {
		t.Errorf("Expected 1 attr, got %d", len(sourceOp.Attrs))
	}

	if sourceOp.Attrs["includepattern0"] != "*.txt" {
		t.Errorf("Expected includepattern0 '*.txt', got '%s'", sourceOp.Attrs["includepattern0"])
	}
}

func TestBkLocalWithExcludeOnly(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			exclude = {".git/", "node_modules/"}
		})
		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	sourceOp := state.Op().Op().GetSource()

	if len(sourceOp.Attrs) != 2 {
		t.Errorf("Expected 2 attrs, got %d", len(sourceOp.Attrs))
	}

	if sourceOp.Attrs["excludepattern0"] != ".git/" {
		t.Errorf("Expected excludepattern0 '.git/', got '%s'", sourceOp.Attrs["excludepattern0"])
	}

	if sourceOp.Attrs["excludepattern1"] != "node_modules/" {
		t.Errorf("Expected excludepattern1 'node_modules/', got '%s'", sourceOp.Attrs["excludepattern1"])
	}
}

func TestBkLocalWithEmptyPatterns(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			include = {},
			exclude = {}
		})
		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Identifier != "local://context" {
		t.Errorf("Expected identifier 'local://context', got '%s'", sourceOp.Identifier)
	}

	if len(sourceOp.Attrs) != 0 {
		t.Errorf("Expected 0 attrs, got %d", len(sourceOp.Attrs))
	}
}

func TestBkLocalWithSharedKeyHint(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			shared_key_hint = "my-cache-key"
		})
		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Attrs["sharedkeyhint"] != "my-cache-key" {
		t.Errorf("Expected sharedkeyhint 'my-cache-key', got '%s'", sourceOp.Attrs["sharedkeyhint"])
	}
}
