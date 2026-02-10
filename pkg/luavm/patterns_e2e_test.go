package luavm

import (
	"testing"
)

func TestCopyWithPatternsEndToEnd(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")

		local result = dst:copy(src, "/app", "/app", {
			include = {
				"*.go",
				"*.mod",
				"*.sum",
				"**/*.yaml",
				"**/*.yml"
			},
			exclude = {
				"*_test.go",
				"*_gen.go",
				"vendor/",
				".git/",
				"**/testdata/"
			},
			mode = "0644",
			follow_symlink = true,
			create_dest_path = true,
			allow_wildcard = true
		})

		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute end-to-end script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state")
	}

	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	copyAction := fileOp.Actions[0].GetCopy()
	if copyAction == nil {
		t.Fatal("Expected Copy action")
	}

	if len(copyAction.IncludePatterns) != 5 {
		t.Errorf("Expected 5 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 5 {
		t.Errorf("Expected 5 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}

	if copyAction.Mode != 0644 {
		t.Errorf("Expected mode 0644, got %d", copyAction.Mode)
	}

	if !copyAction.FollowSymlink {
		t.Error("Expected FollowSymlink to be true")
	}

	if !copyAction.CreateDestPath {
		t.Error("Expected CreateDestPath to be true")
	}

	if !copyAction.AllowWildcard {
		t.Error("Expected AllowWildcard to be true")
	}
}

func TestLocalWithPatternsEndToEnd(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local ctx = bk.local_("context", {
			include = {
				"*.go",
				"*.mod",
				"*.sum",
				"cmd/**/*"
			},
			exclude = {
				"*_test.go",
				"vendor/",
				"**/testdata/"
			},
			shared_key_hint = "go-build-cache"
		})

		bk.export(ctx)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute end-to-end script: %v", err)
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

	if sourceOp.Attrs["sharedkeyhint"] != "go-build-cache" {
		t.Errorf("Expected sharedkeyhint 'go-build-cache', got '%s'", sourceOp.Attrs["sharedkeyhint"])
	}

	if sourceOp.Attrs["includepattern0"] != "*.go" {
		t.Errorf("Expected includepattern0 '*.go', got '%s'", sourceOp.Attrs["includepattern0"])
	}

	if sourceOp.Attrs["excludepattern0"] != "*_test.go" {
		t.Errorf("Expected excludepattern0 '*_test.go', got '%s'", sourceOp.Attrs["excludepattern0"])
	}
}

func TestComplexPatternCombination(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local src = bk.local_("sources", {
			include = {
				"src/**/*.go",
				"pkg/**/*.go",
				"internal/**/*.go",
				"go.mod",
				"go.sum",
				"configs/**/*.yaml",
				"configs/**/*.yml"
			},
			exclude = {
				"**/*_test.go",
				"**/*_gen.go",
				"**/*_mock.go",
				"**/testdata/**/*",
				"vendor/**/*",
				".git/**/*",
				"**/.DS_Store"
			}
		})

		bk.export(src)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute complex pattern script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "local://sources" {
		t.Errorf("Expected identifier 'local://sources', got '%s'", sourceOp.Identifier)
	}

	// 7 include patterns + 7 exclude patterns + 1 shared_key_hint = 15 attrs (but shared_key_hint is only set if provided)
	// Actually shared_key_hint is not set in this test, so 7 + 7 = 14 attrs
	if len(sourceOp.Attrs) != 14 {
		t.Errorf("Expected 14 attrs, got %d", len(sourceOp.Attrs))
	}

	// Check first few include patterns
	expectedIncludes := []string{"src/**/*.go", "pkg/**/*.go", "internal/**/*.go", "go.mod", "go.sum", "configs/**/*.yaml", "configs/**/*.yml"}
	for i, pattern := range expectedIncludes {
		key := "includepattern" + string(rune('0'+i))
		if sourceOp.Attrs[key] != pattern {
			t.Errorf("Expected %s '%s', got '%s'", key, pattern, sourceOp.Attrs[key])
		}
	}

	// Check first few exclude patterns
	expectedExcludes := []string{"**/*_test.go", "**/*_gen.go", "**/*_mock.go", "**/testdata/**/*", "vendor/**/*", ".git/**/*", "**/.DS_Store"}
	for i, pattern := range expectedExcludes {
		key := "excludepattern" + string(rune('0'+i))
		if sourceOp.Attrs[key] != pattern {
			t.Errorf("Expected %s '%s', got '%s'", key, pattern, sourceOp.Attrs[key])
		}
	}
}

func TestPatternNegation(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")

		local result = dst:copy(src, "/config", "/etc/app", {
			include = {
				"*.yaml",
				"*.yml",
				"!test*.yaml",
				"!dev*.yml"
			}
		})

		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute negation pattern script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if len(copyAction.IncludePatterns) != 4 {
		t.Errorf("Expected 4 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	expectedPatterns := []string{"*.yaml", "*.yml", "!test*.yaml", "!dev*.yml"}
	for i, expected := range expectedPatterns {
		if copyAction.IncludePatterns[i] != expected {
			t.Errorf("Include pattern %d: expected '%s', got '%s'", i, expected, copyAction.IncludePatterns[i])
		}
	}
}
