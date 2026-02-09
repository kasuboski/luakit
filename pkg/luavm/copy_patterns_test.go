package luavm

import (
	"strings"
	"testing"
)

func TestCopyPatternsGitIgnoreStyle(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	// Test various .gitignore-style patterns
	testCases := []struct {
		name        string
		include     []string
		exclude     []string
		description string
	}{
		{
			name:        "wildcard patterns",
			include:     []string{"*.go", "*.mod", "*.sum"},
			exclude:     []string{"*_test.go", "vendor/"},
			description: "Go files excluding tests and vendor",
		},
		{
			name:        "directory patterns",
			include:     []string{"src/**/*.go"},
			exclude:     []string{"src/**/*_test.go"},
			description: "Recursive directory patterns",
		},
		{
			name:        "negation patterns",
			include:     []string{"*.go"},
			exclude:     []string{"!main.go"},
			description: "Exclude specific files with negation",
		},
		{
			name:        "path anchored patterns",
			include:     []string{"/app/*.go"},
			exclude:     []string{"/app/test/*.go"},
			description: "Path-specific patterns",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset state for each test case
			exportedState = nil
			exportedImageConfig = nil

			script := buildCopyScript(tc.include, tc.exclude)

			if err := L.DoString(script); err != nil {
				t.Fatalf("Failed to execute script %s: %v", tc.name, err)
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

			if len(copyAction.IncludePatterns) != len(tc.include) {
				t.Errorf("Expected %d include patterns, got %d", len(tc.include), len(copyAction.IncludePatterns))
			}

			if len(copyAction.ExcludePatterns) != len(tc.exclude) {
				t.Errorf("Expected %d exclude patterns, got %d", len(tc.exclude), len(copyAction.ExcludePatterns))
			}

			for i, pattern := range tc.include {
				if copyAction.IncludePatterns[i] != pattern {
					t.Errorf("Include pattern %d: expected '%s', got '%s'", i, pattern, copyAction.IncludePatterns[i])
				}
			}

			for i, pattern := range tc.exclude {
				if copyAction.ExcludePatterns[i] != pattern {
					t.Errorf("Exclude pattern %d: expected '%s', got '%s'", i, pattern, copyAction.ExcludePatterns[i])
				}
			}
		})
	}
}

func TestCopyPatternsEmptyArrays(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			include = {},
			exclude = {}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if len(copyAction.IncludePatterns) != 0 {
		t.Errorf("Expected 0 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 0 {
		t.Errorf("Expected 0 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}
}

func TestCopyPatternsOnlyInclude(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			include = {"*.go"}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if len(copyAction.IncludePatterns) != 1 {
		t.Errorf("Expected 1 include pattern, got %d", len(copyAction.IncludePatterns))
	}

	if copyAction.IncludePatterns[0] != "*.go" {
		t.Errorf("Expected include pattern '*.go', got '%s'", copyAction.IncludePatterns[0])
	}

	if len(copyAction.ExcludePatterns) != 0 {
		t.Errorf("Expected 0 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}
}

func TestCopyPatternsOnlyExclude(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			exclude = {"*.test", "node_modules/"}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if len(copyAction.IncludePatterns) != 0 {
		t.Errorf("Expected 0 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}
}

func TestCopyPatternsCombinedWithOtherOptions(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			mode = "0755",
			follow_symlink = true,
			create_dest_path = true,
			allow_wildcard = true,
			include = {"*.so", "*.a"},
			exclude = {"*.debug"},
			owner = { user = "root", group = "root" }
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if copyAction.Mode != 0755 {
		t.Errorf("Expected mode 0755, got %d", copyAction.Mode)
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

	if len(copyAction.IncludePatterns) != 2 {
		t.Errorf("Expected 2 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 1 {
		t.Errorf("Expected 1 exclude pattern, got %d", len(copyAction.ExcludePatterns))
	}

	if copyAction.Owner == nil {
		t.Fatal("Expected owner to be set")
	}
}

func TestCopyPatternsWithWildcards(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	// Test common wildcard patterns used in .gitignore
	script := `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
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
				"**/testdata/",
				".git/"
			}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	copyAction := state.Op().Op().GetFile().Actions[0].GetCopy()

	if len(copyAction.IncludePatterns) != 5 {
		t.Errorf("Expected 5 include patterns, got %d", len(copyAction.IncludePatterns))
	}

	if len(copyAction.ExcludePatterns) != 5 {
		t.Errorf("Expected 5 exclude patterns, got %d", len(copyAction.ExcludePatterns))
	}

	// Verify specific patterns
	expectedIncludes := []string{"*.go", "*.mod", "*.sum", "**/*.yaml", "**/*.yml"}
	for i, expected := range expectedIncludes {
		if copyAction.IncludePatterns[i] != expected {
			t.Errorf("Include pattern %d: expected '%s', got '%s'", i, expected, copyAction.IncludePatterns[i])
		}
	}

	expectedExcludes := []string{"*_test.go", "*_gen.go", "vendor/", "**/testdata/", ".git/"}
	for i, expected := range expectedExcludes {
		if copyAction.ExcludePatterns[i] != expected {
			t.Errorf("Exclude pattern %d: expected '%s', got '%s'", i, expected, copyAction.ExcludePatterns[i])
		}
	}
}

func buildCopyScript(include, exclude []string) string {
	includeStr := formatLuaArray(include)
	excludeStr := formatLuaArray(exclude)

	return `
		local src = bk.image("alpine:3.19")
		local dst = bk.image("ubuntu:24.04")
		local result = dst:copy(src, "/src", "/dst", {
			include = ` + includeStr + `,
			exclude = ` + excludeStr + `
		})
		bk.export(result)
	`
}

func formatLuaArray(items []string) string {
	if len(items) == 0 {
		return "{}"
	}
	var result strings.Builder
	result.WriteString("{")
	for i, item := range items {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(`"` + item + `"`)
	}
	result.WriteString("}")
	return result.String()
}
