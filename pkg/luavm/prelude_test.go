package luavm

import (
	"path/filepath"
	"testing"
)

func getStdlibPath(t *testing.T) string {
	absPath, err := filepath.Abs("../../lua/stdlib")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	return absPath
}

func TestPreludeRequire(t *testing.T) {
	resetExportedState()
	defer resetExportedState()

	L := NewVM(&VMConfig{
		StdlibDir: getStdlibPath(t),
	})
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
local prelude = require("prelude")
local base = prelude.from_alpine()
local result = base:run("echo test")
bk.export(result)
`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}
}

func TestPreludeBaseImages(t *testing.T) {
	defer resetExportedState()
	stdlibPath := getStdlibPath(t)

	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "from_alpine",
			script: `
local prelude = require("prelude")
local base = prelude.from_alpine()
bk.export(base)
`,
			expected: "alpine:3.19",
		},
		{
			name: "from_ubuntu",
			script: `
local prelude = require("prelude")
local base = prelude.from_ubuntu()
bk.export(base)
`,
			expected: "ubuntu:24.04",
		},
		{
			name: "from_debian",
			script: `
local prelude = require("prelude")
local base = prelude.from_debian()
bk.export(base)
`,
			expected: "debian:bookworm-slim",
		},
		{
			name: "from_fedora",
			script: `
local prelude = require("prelude")
local base = prelude.from_fedora()
bk.export(base)
`,
			expected: "fedora:39",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()

			L := NewVM(&VMConfig{
				StdlibDir: stdlibPath,
			})
			testVM = L
			defer L.Close()
			defer func() { testVM = nil }()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			sourceOp := state.Op().Op().GetSource()
			if sourceOp == nil {
				t.Fatal("Expected source operation")
			}

			expectedIdentifier := "docker-image://" + tc.expected
			if sourceOp.Identifier != expectedIdentifier {
				t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
			}
		})
	}
}

func TestPreludeGoBuilders(t *testing.T) {
	defer resetExportedState()
	stdlibPath := getStdlibPath(t)

	testCases := []struct {
		name   string
		script string
	}{
		{
			name: "go_base",
			script: `
local prelude = require("prelude")
local base = prelude.go_base()
bk.export(base)
`,
		},
		{
			name: "go_base_custom_version",
			script: `
local prelude = require("prelude")
local base = prelude.go_base("1.21-alpine")
bk.export(base)
`,
		},
		{
			name: "go_runtime",
			script: `
local prelude = require("prelude")
local runtime = prelude.go_runtime()
bk.export(runtime)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()

			L := NewVM(&VMConfig{
				StdlibDir: stdlibPath,
			})
			testVM = L
			defer L.Close()
			defer func() { testVM = nil }()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}
		})
	}
}

func TestPreludePackageInstallers(t *testing.T) {
	defer resetExportedState()
	stdlibPath := getStdlibPath(t)

	testCases := []struct {
		name   string
		script string
	}{
		{
			name: "apk_package_table",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local result = prelude.apk_package(base, { "git", "curl" })
bk.export(result)
`,
		},
		{
			name: "apk_package_string",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local result = prelude.apk_package(base, "git curl")
bk.export(result)
`,
		},
		{
			name: "deb_package_table",
			script: `
local prelude = require("prelude")
local base = bk.image("debian:bookworm-slim")
local result = prelude.deb_package(base, { "git", "curl" })
bk.export(result)
`,
		},
		{
			name: "deb_package_string",
			script: `
local prelude = require("prelude")
local base = bk.image("debian:bookworm-slim")
local result = prelude.deb_package(base, "git curl")
bk.export(result)
`,
		},
		{
			name: "install_git",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local result = prelude.install_git(base)
bk.export(result)
`,
		},
		{
			name: "install_curl",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local result = prelude.install_curl(base)
bk.export(result)
`,
		},
		{
			name: "install_ca_certs",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local result = prelude.install_ca_certs(base)
bk.export(result)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()

			L := NewVM(&VMConfig{
				StdlibDir: stdlibPath,
			})
			testVM = L
			defer L.Close()
			defer func() { testVM = nil }()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}
		})
	}
}

func TestPreludeHelpers(t *testing.T) {
	defer resetExportedState()
	stdlibPath := getStdlibPath(t)

	testCases := []struct {
		name   string
		script string
	}{
		{
			name: "standard_base_alpine",
			script: `
local prelude = require("prelude")
local base = prelude.standard_base("alpine")
bk.export(base)
`,
		},
		{
			name: "standard_base_ubuntu",
			script: `
local prelude = require("prelude")
local base = prelude.standard_base("ubuntu")
bk.export(base)
`,
		},
		{
			name: "standard_base_debian",
			script: `
local prelude = require("prelude")
local base = prelude.standard_base("debian")
bk.export(base)
`,
		},
		{
			name: "parallel_build",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local state1 = base:run("echo '1' > /1.txt")
local state2 = base:run("echo '2' > /2.txt")
local state3 = base:run("echo '3' > /3.txt")
local merged = prelude.parallel_build(state1, state2, state3)
bk.export(merged)
`,
		},
		{
			name: "merge_multiple",
			script: `
local prelude = require("prelude")
local base = bk.image("alpine:3.19")
local state1 = base:run("echo '1' > /1.txt")
local state2 = base:run("echo '2' > /2.txt")
local merged = prelude.merge_multiple({ state1, state2 })
bk.export(merged)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()

			L := NewVM(&VMConfig{
				StdlibDir: stdlibPath,
			})
			testVM = L
			defer L.Close()
			defer func() { testVM = nil }()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}
		})
	}
}

func TestPreludeAppBuilders(t *testing.T) {
	defer resetExportedState()
	stdlibPath := getStdlibPath(t)

	testCases := []struct {
		name   string
		script string
	}{
		{
			name: "go_binary_app",
			script: `
local prelude = require("prelude")
local src = bk.local_("context")
local final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = ".",
	user = "app",
})
bk.export(final)
`,
		},
		{
			name: "node_app",
			script: `
local prelude = require("prelude")
local src = bk.local_("context")
local final = prelude.node_app("20-alpine", src, {
	cwd = "/app",
	user = "nodejs",
})
bk.export(final)
`,
		},
		{
			name: "python_app",
			script: `
local prelude = require("prelude")
local src = bk.local_("context")
local final = prelude.python_app("3.11", src, {
	cwd = "/workspace",
	user = "appuser",
})
bk.export(final)
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()

			L := NewVM(&VMConfig{
				StdlibDir: stdlibPath,
			})
			testVM = L
			defer L.Close()
			defer func() { testVM = nil }()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}
		})
	}
}

func TestPludeRealWorldPattern(t *testing.T) {
	stdlibPath := getStdlibPath(t)

	script := `
local prelude = require("prelude")

local src = bk.local_("context")

local final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = "./cmd/server",
	output = "/out/server",
	user = "app",
	uid = 1000,
	gid = 1000,
})

bk.export(final, {
	entrypoint = {"/app/server"},
	user = "app",
	workdir = "/app",
	expose = {"8080/tcp"},
	env = {
		GIN_MODE = "release",
		PORT = "8080",
	},
})
`

	defer resetExportedState()

	L := NewVM(&VMConfig{
		StdlibDir: stdlibPath,
	})
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	config := GetExportedImageConfig()
	if config == nil {
		t.Fatal("Expected image config to be set")
	}

	if config.Config.User != "app" {
		t.Errorf("Expected user 'app', got '%s'", config.Config.User)
	}

	if config.Config.WorkingDir != "/app" {
		t.Errorf("Expected workdir '/app', got '%s'", config.Config.WorkingDir)
	}
}
