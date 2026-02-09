package luavm

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestBkImageValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "empty string",
			input:       `bk.image("")`,
			shouldError: true,
			errorMsg:    "identifier must not be empty",
		},
		{
			name:        "whitespace only",
			input:       `bk.image("   ")`,
			shouldError: true,
			errorMsg:    "identifier must not be empty",
		},
		{
			name:        "nil",
			input:       `bk.image(nil)`,
			shouldError: true,
			errorMsg:    "string expected",
		},
		{
			name:        "number",
			input:       `bk.image(123)`,
			shouldError: true,
			errorMsg:    "string expected",
		},
		{
			name:        "table",
			input:       `bk.image({})`,
			shouldError: true,
			errorMsg:    "string expected",
		},
		{
			name:        "valid simple ref",
			input:       `bk.image("alpine:3.19")`,
			shouldError: false,
		},
		{
			name:        "valid full ref",
			input:       `bk.image("docker.io/library/alpine:3.19")`,
			shouldError: false,
		},
		{
			name:        "valid with digest",
			input:       `bk.image("alpine@sha256:abc123")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got no error", tc.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBkLocalValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "empty string",
			input:       `bk.local_("")`,
			shouldError: true,
			errorMsg:    "name must not be empty",
		},
		{
			name:        "whitespace only",
			input:       `bk.local_("   ")`,
			shouldError: true,
			errorMsg:    "name must not be empty",
		},
		{
			name:        "nil",
			input:       `bk.local_(nil)`,
			shouldError: true,
			errorMsg:    "string expected",
		},
		{
			name:        "number",
			input:       `bk.local_(123)`,
			shouldError: true,
			errorMsg:    "string expected",
		},
		{
			name:        "valid context name",
			input:       `bk.local_("context")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBkScratchValidation(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	err := L.DoString(`bk.scratch("extra")`)
	if err == nil {
		t.Errorf("bk.scratch should reject arguments, got no error")
	}
}

func TestStateRunValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "no command",
			input:       `bk.image("alpine:3.19"):run()`,
			shouldError: true,
			errorMsg:    "command argument required",
		},
		{
			name:        "nil command",
			input:       `bk.image("alpine:3.19"):run(nil)`,
			shouldError: true,
			errorMsg:    "string or table",
		},
		{
			name:        "number command",
			input:       `bk.image("alpine:3.19"):run(123)`,
			shouldError: true,
			errorMsg:    "string or table",
		},
		{
			name:        "table command",
			input:       `bk.image("alpine:3.19"):run({"echo", "hello"})`,
			shouldError: false,
		},
		{
			name:        "string command",
			input:       `bk.image("alpine:3.19"):run("echo hello")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestStateCopyValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "missing from state",
			input:       `bk.image("alpine:3.19"):copy(nil, "/src", "/dest")`,
			shouldError: true,
		},
		{
			name:        "non-state from",
			input:       `bk.image("alpine:3.19"):copy("not a state", "/src", "/dest")`,
			shouldError: true,
		},
		{
			name:        "missing src",
			input:       `local s = bk.image("alpine:3.19"); s:copy(s, nil, "/dest")`,
			shouldError: true,
		},
		{
			name:        "missing dest",
			input:       `local s = bk.image("alpine:3.19"); s:copy(s, "/src", nil)`,
			shouldError: true,
		},
		{
			name:        "valid copy",
			input:       `local s = bk.image("alpine:3.19"); s:copy(s, "/src", "/dest")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestStateMkdirValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "missing path",
			input:       `bk.image("alpine:3.19"):mkdir(nil)`,
			shouldError: true,
		},
		{
			name:        "empty path",
			input:       `bk.image("alpine:3.19"):mkdir("")`,
			shouldError: true,
		},
		{
			name:        "valid mkdir",
			input:       `bk.image("alpine:3.19"):mkdir("/app")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestStateMkfileValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "missing path",
			input:       `bk.image("alpine:3.19"):mkfile(nil, "data")`,
			shouldError: true,
		},
		{
			name:        "empty path",
			input:       `bk.image("alpine:3.19"):mkfile("", "data")`,
			shouldError: true,
		},
		{
			name:        "missing data",
			input:       `bk.image("alpine:3.19"):mkfile("/path", nil)`,
			shouldError: true,
		},
		{
			name:        "valid mkfile",
			input:       `bk.image("alpine:3.19"):mkfile("/path", "data")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestStateRmValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "missing path",
			input:       `bk.image("alpine:3.19"):rm(nil)`,
			shouldError: true,
		},
		{
			name:        "empty path",
			input:       `bk.image("alpine:3.19"):rm("")`,
			shouldError: true,
		},
		{
			name:        "valid rm",
			input:       `bk.image("alpine:3.19"):rm("/tmp")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestStateSymlinkValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "missing oldpath",
			input:       `bk.image("alpine:3.19"):symlink(nil, "/new")`,
			shouldError: true,
		},
		{
			name:        "empty oldpath",
			input:       `bk.image("alpine:3.19"):symlink("", "/new")`,
			shouldError: true,
		},
		{
			name:        "missing newpath",
			input:       `bk.image("alpine:3.19"):symlink("/old", nil)`,
			shouldError: true,
		},
		{
			name:        "empty newpath",
			input:       `bk.image("alpine:3.19"):symlink("/old", "")`,
			shouldError: true,
		},
		{
			name:        "valid symlink",
			input:       `bk.image("alpine:3.19"):symlink("/old", "/new")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBkExportValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "nil state",
			input:       `bk.export(nil)`,
			shouldError: true,
		},
		{
			name:        "non-state",
			input:       `bk.export("not a state")`,
			shouldError: true,
		},
		{
			name:        "double export",
			input:       `local s = bk.image("alpine:3.19"); bk.export(s); bk.export(s)`,
			shouldError: true,
			errorMsg:    "already called once",
		},
		{
			name:        "valid export",
			input:       `local s = bk.image("alpine:3.19"); bk.export(s)`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					if tc.errorMsg != "" {
						t.Errorf("Expected error containing '%s', got no error", tc.errorMsg)
					} else {
						t.Errorf("Expected error, got no error")
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBkMergeValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "zero states",
			input:       `bk.merge()`,
			shouldError: true,
			errorMsg:    "requires at least 2 states",
		},
		{
			name:        "one state",
			input:       `local s = bk.image("alpine:3.19"); bk.merge(s)`,
			shouldError: true,
			errorMsg:    "requires at least 2 states",
		},
		{
			name:        "non-state argument",
			input:       `bk.merge("not a state")`,
			shouldError: true,
		},
		{
			name:        "valid merge",
			input:       `local s1 = bk.image("alpine:3.19"); local s2 = bk.image("ubuntu:24.04"); bk.merge(s1, s2)`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					if tc.errorMsg != "" {
						t.Errorf("Expected error containing '%s', got no error", tc.errorMsg)
					} else {
						t.Errorf("Expected error, got no error")
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBkDiffValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "zero arguments",
			input:       `bk.diff()`,
			shouldError: true,
			errorMsg:    "requires lower and upper",
		},
		{
			name:        "one argument",
			input:       `local s = bk.image("alpine:3.19"); bk.diff(s)`,
			shouldError: true,
			errorMsg:    "requires lower and upper",
		},
		{
			name:        "non-state lower",
			input:       `bk.diff("not a state", bk.image("alpine:3.19"))`,
			shouldError: true,
		},
		{
			name:        "non-state upper",
			input:       `bk.diff(bk.image("alpine:3.19"), "not a state")`,
			shouldError: true,
		},
		{
			name:        "valid diff",
			input:       `local s1 = bk.image("alpine:3.19"); local s2 = s1:run("echo"); bk.diff(s1, s2)`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					if tc.errorMsg != "" {
						t.Errorf("Expected error containing '%s', got no error", tc.errorMsg)
					} else {
						t.Errorf("Expected error, got no error")
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestMountValidation(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{
			name:        "cache nil dest",
			input:       `bk.cache(nil)`,
			shouldError: true,
		},
		{
			name:        "cache number dest",
			input:       `bk.cache(123)`,
			shouldError: true,
		},
		{
			name:        "cache valid",
			input:       `bk.cache("/cache")`,
			shouldError: false,
		},
		{
			name:        "secret nil dest",
			input:       `bk.secret(nil)`,
			shouldError: true,
		},
		{
			name:        "secret number dest",
			input:       `bk.secret(123)`,
			shouldError: true,
		},
		{
			name:        "secret valid",
			input:       `bk.secret("/secret")`,
			shouldError: false,
		},
		{
			name:        "tmpfs nil dest",
			input:       `bk.tmpfs(nil)`,
			shouldError: true,
		},
		{
			name:        "tmpfs number dest",
			input:       `bk.tmpfs(123)`,
			shouldError: true,
		},
		{
			name:        "tmpfs valid",
			input:       `bk.tmpfs("/tmp")`,
			shouldError: false,
		},
		{
			name:        "bind nil state",
			input:       `bk.bind(nil, "/dest")`,
			shouldError: true,
		},
		{
			name:        "bind non-state",
			input:       `bk.bind("not a state", "/dest")`,
			shouldError: true,
		},
		{
			name:        "bind nil dest",
			input:       `local s = bk.image("alpine:3.19"); bk.bind(s, nil)`,
			shouldError: true,
		},
		{
			name:        "bind valid",
			input:       `local s = bk.image("alpine:3.19"); bk.bind(s, "/dest")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			err := L.DoString(tc.input)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error, got no error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestExportedImageConfigComplete(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		bk.export(s, {
			entrypoint = {"/bin/sh"},
			cmd = {"echo hello"},
			env = {PATH = "/usr/bin"},
			workdir = "/app",
			user = "appuser",
			labels = {version = "1.0"},
			expose = {"8080/tcp"},
			os = "linux",
			arch = "arm64",
			variant = "v8"
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	config := GetExportedImageConfig()
	if config == nil {
		t.Fatal("Expected non-nil image config")
	}

	if len(config.Config.Entrypoint) != 1 || config.Config.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Expected entrypoint ['/bin/sh'], got %v", config.Config.Entrypoint)
	}

	if len(config.Config.Cmd) != 1 || config.Config.Cmd[0] != "echo hello" {
		t.Errorf("Expected cmd ['echo hello'], got %v", config.Config.Cmd)
	}

	if config.Config.WorkingDir != "/app" {
		t.Errorf("Expected workdir '/app', got '%s'", config.Config.WorkingDir)
	}

	if config.Config.User != "appuser" {
		t.Errorf("Expected user 'appuser', got '%s'", config.Config.User)
	}

	if config.OS != "linux" {
		t.Errorf("Expected os 'linux', got '%s'", config.OS)
	}

	if config.Architecture != "arm64" {
		t.Errorf("Expected arch 'arm64', got '%s'", config.Architecture)
	}

	if config.Variant != "v8" {
		t.Errorf("Expected variant 'v8', got '%s'", config.Variant)
	}
}

func TestUnknownStateMethod(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s:unknown_method()
	`

	err := L.DoString(script)
	if err == nil {
		t.Error("Expected error for unknown method")
	}
}

func TestStateImmutabilityAcrossOperations(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")
		local s1 = base:run("echo 1")
		local s2 = base:run("echo 2")

		bk.export(s2)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()
	if execOp.Meta.Args[2] != "echo 2" {
		t.Errorf("Expected 'echo 2', got '%s'", execOp.Meta.Args[2])
	}
}

func TestComplexDAGConstruction(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")
		local deps = base:run("apk add curl", { cwd = "/root" })
		local src = bk.local_("context")
		local workspace = deps:copy(src, ".", "/app")
		local build1 = workspace:run("make build1", { cwd = "/app" })
		local build2 = workspace:run("make build2", { cwd = "/app" })
		local test = workspace:run("make test", { cwd = "/app" })
		local merged = bk.merge(build1, build2, test)
		local final = merged:mkdir("/output")
		bk.export(final)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute complex script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	if len(fileOp.Actions) != 1 {
		t.Errorf("Expected 1 file action, got %d", len(fileOp.Actions))
	}

	mkdirAction := fileOp.Actions[0].GetMkdir()
	if mkdirAction == nil {
		t.Fatal("Expected Mkdir action")
	}
}

func TestStateChaining(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s = s:mkdir("/app")
		s = s:mkdir("/app/data")
		s = s:mkfile("/app/config.json", "{}")
		s = s:mkfile("/app/data/data.json", "[]")
		s = s:run("chmod +x /app")
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute chained operations: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()
	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}
}

func TestFileOperationsWithOwner(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	script := `
		local s = bk.image("alpine:3.19")
		s = s:mkdir("/app", { owner = { user = 1000, group = 1000 } })
		s = s:mkfile("/app/file", "data", { owner = { user = "appuser", group = "appgroup" } })
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute file operations with owner: %v", err)
	}

	state := GetExportedState()
	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	for _, action := range fileOp.Actions {
		if mkdirAction := action.GetMkdir(); mkdirAction != nil {
			if mkdirAction.Owner == nil {
				t.Error("Expected mkdir owner to be set")
			}
		}
		if mkfileAction := action.GetMkfile(); mkfileAction != nil {
			if mkfileAction.Owner == nil {
				t.Error("Expected mkfile owner to be set")
			}
		}
	}
}

func TestRunOptionsValidation(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name        string
		options     string
		shouldError bool
	}{
		{
			name:        "valid env",
			options:     `{ env = { PATH = "/usr/bin" } }`,
			shouldError: false,
		},
		{
			name:        "valid cwd",
			options:     `{ cwd = "/app" }`,
			shouldError: false,
		},
		{
			name:        "valid user",
			options:     `{ user = "nobody" }`,
			shouldError: false,
		},
		{
			name:        "valid mounts",
			options:     `{ mounts = { bk.cache("/cache") } }`,
			shouldError: false,
		},
		{
			name:        "valid network",
			options:     `{ network = "none" }`,
			shouldError: false,
		},
		{
			name:        "valid security",
			options:     `{ security = "sandbox" }`,
			shouldError: false,
		},
		{
			name:        "valid network host",
			options:     `{ network = "host" }`,
			shouldError: false,
		},
		{
			name:        "valid security insecure",
			options:     `{ security = "insecure" }`,
			shouldError: false,
		},
		{
			name:        "invalid env type",
			options:     `{ env = "not a table" }`,
			shouldError: true,
		},
		{
			name:        "invalid cwd type",
			options:     `{ cwd = 123 }`,
			shouldError: true,
		},
		{
			name:        "invalid user type",
			options:     `{ user = 123 }`,
			shouldError: true,
		},
		{
			name:        "invalid mounts type",
			options:     `{ mounts = "not a table" }`,
			shouldError: true,
		},
		{
			name:        "invalid network type",
			options:     `{ network = 123 }`,
			shouldError: true,
		},
		{
			name:        "invalid security type",
			options:     `{ security = 123 }`,
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			script := `local s = bk.image("alpine:3.19"); s:run("echo", ` + tc.options + `)`

			err := L.DoString(script)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for options: %s", tc.options)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for options %s: %v", tc.options, err)
				}
			}
		})
	}
}

func TestStateTypeChecking(t *testing.T) {
	defer resetExportedState()
	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name        string
		script      string
		shouldError bool
	}{
		{
			name:        "image returns userdata",
			script:      `result = bk.image("alpine:3.19")`,
			shouldError: false,
		},
		{
			name:        "scratch returns userdata",
			script:      `result = bk.scratch()`,
			shouldError: false,
		},
		{
			name:        "local returns userdata",
			script:      `result = bk.local_("context")`,
			shouldError: false,
		},
		{
			name:        "cache returns userdata",
			script:      `result = bk.cache("/cache")`,
			shouldError: false,
		},
		{
			name:        "secret returns userdata",
			script:      `result = bk.secret("/secret")`,
			shouldError: false,
		},
		{
			name:        "ssh returns userdata",
			script:      `result = bk.ssh()`,
			shouldError: false,
		},
		{
			name:        "tmpfs returns userdata",
			script:      `result = bk.tmpfs("/tmp")`,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := L.DoString(tc.script)
			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for: %s", tc.script)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tc.name, err)
				}

				result := L.GetGlobal("result")
				if result.Type() != lua.LTUserData {
					t.Errorf("Expected userdata, got %v", result.Type())
				}
			}
		})
	}
}

func TestConcurrencySafety(t *testing.T) {
	defer resetExportedState()

	done := make(chan bool, 2)

	go func() {
		L := NewVM(nil)
		defer L.Close()
		L.DoString(`local s = bk.image("alpine:3.19"); bk.export(s)`)
		done <- true
	}()

	go func() {
		L := NewVM(nil)
		defer L.Close()
		L.DoString(`local s = bk.image("ubuntu:24.04"); bk.export(s)`)
		done <- true
	}()

	<-done
	<-done
}
