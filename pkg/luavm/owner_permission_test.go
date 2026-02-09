package luavm

import (
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
)

func TestCopyWithOwnerAndMode(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name            string
		script          string
		shouldHaveOwner bool
		expectedMode    int32
	}{
		{
			name: "copy with owner by name and mode as string",
			script: `
				local src = bk.image("alpine:3.19")
				local dst = bk.image("ubuntu:24.04")
				local result = dst:copy(src, "/src", "/dst", {
					owner = { user = "appuser", group = "appgroup" },
					mode = "0755"
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    0755,
		},
		{
			name: "copy with owner by ID and mode as number",
			script: `
				local src = bk.image("alpine:3.19")
				local dst = bk.image("ubuntu:24.04")
				local result = dst:copy(src, "/src", "/dst", {
					owner = { user = 1000, group = 1000 },
					mode = 420
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    420, // 0644 octal = 420 decimal
		},
		{
			name: "copy with mixed owner types",
			script: `
				local src = bk.image("alpine:3.19")
				local dst = bk.image("ubuntu:24.04")
				local result = dst:copy(src, "/src", "/dst", {
					owner = { user = "appuser", group = 1000 },
					mode = "0755"
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    0755,
		},
		{
			name: "copy with mode only",
			script: `
				local src = bk.image("alpine:3.19")
				local dst = bk.image("ubuntu:24.04")
				local result = dst:copy(src, "/src", "/dst", {
					mode = 493
				})
				bk.export(result)
			`,
			shouldHaveOwner: false,
			expectedMode:    493, // 0755 octal = 493 decimal
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			fileOp := state.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			copyAction := fileOp.Actions[0].GetCopy()
			if copyAction == nil {
				t.Fatal("Expected Copy action")
			}

			if copyAction.Mode != tc.expectedMode {
				t.Errorf("Expected mode %o, got %o", tc.expectedMode, copyAction.Mode)
			}

			if tc.shouldHaveOwner {
				if copyAction.Owner == nil {
					t.Error("Expected Owner to be set")
				}
			} else {
				if copyAction.Owner != nil {
					t.Error("Expected Owner to be nil")
				}
			}
		})
	}
}

func TestMkdirWithOwnerAndMode(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name            string
		script          string
		shouldHaveOwner bool
		expectedMode    int32
	}{
		{
			name: "mkdir with owner by name and mode as string",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = "appuser", group = "appgroup" },
					mode = "0755",
					make_parents = true
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    0755,
		},
		{
			name: "mkdir with owner by ID and mode as number",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = 1000, group = 1000 },
					mode = 493,
					make_parents = true
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    493, // 0755 octal = 493 decimal
		},
		{
			name: "mkdir with mode only",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					mode = 420,
					make_parents = true
				})
				bk.export(result)
			`,
			shouldHaveOwner: false,
			expectedMode:    420, // 0644 octal = 420 decimal
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			fileOp := state.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			mkdirAction := fileOp.Actions[0].GetMkdir()
			if mkdirAction == nil {
				t.Fatal("Expected Mkdir action")
			}

			if mkdirAction.Mode != tc.expectedMode {
				t.Errorf("Expected mode %o, got %o", tc.expectedMode, mkdirAction.Mode)
			}

			if tc.shouldHaveOwner {
				if mkdirAction.Owner == nil {
					t.Error("Expected Owner to be set")
				}
			} else {
				if mkdirAction.Owner != nil {
					t.Error("Expected Owner to be nil")
				}
			}
		})
	}
}

func TestMkfileWithOwnerAndMode(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name            string
		script          string
		shouldHaveOwner bool
		expectedMode    int32
	}{
		{
			name: "mkfile with owner by name and mode as string",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkfile("/app/config.json", '{"key":"value"}', {
					owner = { user = "root", group = "root" },
					mode = "0644"
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    0644,
		},
		{
			name: "mkfile with owner by ID and mode as number",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkfile("/app/config.json", '{"key":"value"}', {
					owner = { user = 0, group = 0 },
					mode = 384
				})
				bk.export(result)
			`,
			shouldHaveOwner: true,
			expectedMode:    384, // 0600 octal = 384 decimal
		},
		{
			name: "mkfile with mode only",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkfile("/app/config.json", '{"key":"value"}', {
					mode = 420
				})
				bk.export(result)
			`,
			shouldHaveOwner: false,
			expectedMode:    420, // 0644 octal = 420 decimal
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			fileOp := state.Op().Op().GetFile()
			if fileOp == nil {
				t.Fatal("Expected FileOp")
			}

			mkfileAction := fileOp.Actions[0].GetMkfile()
			if mkfileAction == nil {
				t.Fatal("Expected Mkfile action")
			}

			if mkfileAction.Mode != tc.expectedMode {
				t.Errorf("Expected mode %o, got %o", tc.expectedMode, mkfileAction.Mode)
			}

			if tc.shouldHaveOwner {
				if mkfileAction.Owner == nil {
					t.Error("Expected Owner to be set")
				}
			} else {
				if mkfileAction.Owner != nil {
					t.Error("Expected Owner to be nil")
				}
			}
		})
	}
}

func TestOwnerParsing(t *testing.T) {
	L := NewVM(nil)
	defer L.Close()

	testCases := []struct {
		name     string
		script   string
		validate func(t *testing.T, state *dag.State)
	}{
		{
			name: "owner with user and group names",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = "appuser", group = "appgroup" }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner == nil {
					t.Fatal("Expected Owner to be set")
				}

				if mkdirAction.Owner.User.GetByName() == nil {
					t.Error("Expected user to be set by name")
				}

				if mkdirAction.Owner.User.GetByName().Name != "appuser" {
					t.Errorf("Expected user name 'appuser', got '%s'", mkdirAction.Owner.User.GetByName().Name)
				}

				if mkdirAction.Owner.Group.GetByName() == nil {
					t.Error("Expected group to be set by name")
				}

				if mkdirAction.Owner.Group.GetByName().Name != "appgroup" {
					t.Errorf("Expected group name 'appgroup', got '%s'", mkdirAction.Owner.Group.GetByName().Name)
				}
			},
		},
		{
			name: "owner with user and group IDs",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = 1000, group = 1000 }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner == nil {
					t.Fatal("Expected Owner to be set")
				}

				if mkdirAction.Owner.User.GetByID() != 1000 {
					t.Errorf("Expected user ID 1000, got %d", mkdirAction.Owner.User.GetByID())
				}

				if mkdirAction.Owner.Group.GetByID() != 1000 {
					t.Errorf("Expected group ID 1000, got %d", mkdirAction.Owner.Group.GetByID())
				}
			},
		},
		{
			name: "owner with user name and group ID",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = "appuser", group = 1000 }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner.User.GetByName() == nil {
					t.Error("Expected user to be set by name")
				}

				if mkdirAction.Owner.User.GetByName().Name != "appuser" {
					t.Errorf("Expected user name 'appuser', got '%s'", mkdirAction.Owner.User.GetByName().Name)
				}

				if mkdirAction.Owner.Group.GetByID() != 1000 {
					t.Errorf("Expected group ID 1000, got %d", mkdirAction.Owner.Group.GetByID())
				}
			},
		},
		{
			name: "owner with user ID and group name",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = 1000, group = "appgroup" }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner.User.GetByID() != 1000 {
					t.Errorf("Expected user ID 1000, got %d", mkdirAction.Owner.User.GetByID())
				}

				if mkdirAction.Owner.Group.GetByName() == nil {
					t.Error("Expected group to be set by name")
				}

				if mkdirAction.Owner.Group.GetByName().Name != "appgroup" {
					t.Errorf("Expected group name 'appgroup', got '%s'", mkdirAction.Owner.Group.GetByName().Name)
				}
			},
		},
		{
			name: "owner with only user",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { user = "appuser" }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner.User.GetByName() == nil {
					t.Error("Expected user to be set by name")
				}

				if mkdirAction.Owner.Group != nil {
					t.Error("Expected group to be nil")
				}
			},
		},
		{
			name: "owner with only group",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", {
					owner = { group = "appgroup" }
				})
				bk.export(result)
			`,
			validate: func(t *testing.T, state *dag.State) {
				fileOp := state.Op().Op().GetFile()
				mkdirAction := fileOp.Actions[0].GetMkdir()

				if mkdirAction.Owner.User != nil {
					t.Error("Expected user to be nil")
				}

				if mkdirAction.Owner.Group.GetByName() == nil {
					t.Error("Expected group to be set by name")
				}

				if mkdirAction.Owner.Group.GetByName().Name != "appgroup" {
					t.Errorf("Expected group name 'appgroup', got '%s'", mkdirAction.Owner.Group.GetByName().Name)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			tc.validate(t, state)
		})
	}
}

func TestModeParsingStringAndNumber(t *testing.T) {
	defer resetExportedState()

	testCases := []struct {
		name   string
		mode   any
		script string
	}{
		{
			name: "mode as octal string",
			mode: "0755",
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", { mode = "0755" })
				bk.export(result)
			`,
		},
		{
			name: "mode as decimal number representing octal",
			mode: 493, // 0755 octal = 493 decimal
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", { mode = 493 })
				bk.export(result)
			`,
		},
		{
			name: "mode as decimal number",
			mode: 644,
			script: `
				local s = bk.image("alpine:3.19")
				local result = s:mkdir("/app", { mode = 644 })
				bk.export(result)
			`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer resetExportedState()
			L := NewVM(nil)
			defer L.Close()

			if err := L.DoString(tc.script); err != nil {
				t.Fatalf("Failed to execute script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			fileOp := state.Op().Op().GetFile()
			mkdirAction := fileOp.Actions[0].GetMkdir()

			expectedMode := int32(493) // 0755 octal = 493 decimal
			if tc.name == "mode as decimal number" {
				expectedMode = 644
			}

			if mkdirAction.Mode != expectedMode {
				t.Errorf("Expected mode %d (decimal), got %d (decimal)", expectedMode, mkdirAction.Mode)
			}
		})
	}
}

func TestComprehensiveFileOperationsWithOwnerAndMode(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	defer L.Close()

	script := `
		local base = bk.image("alpine:3.19")

		-- Mkdir with owner and mode
		local s1 = base:mkdir("/app", {
			mode = 0755,
			make_parents = true,
			owner = { user = 1000, group = 1000 }
		})

		-- Mkfile with owner and mode
		local s2 = s1:mkfile("/app/config.json", '{"key":"value"}', {
			mode = 0644,
			owner = { user = "appuser", group = "appgroup" }
		})

		-- Create another source
		local src = bk.image("ubuntu:24.04")

		-- Copy with owner and mode
		local s3 = s2:copy(src, "/etc/hosts", "/app/hosts", {
			mode = "0644",
			owner = { user = "root", group = "root" }
		})

		bk.export(s3)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	state := GetExportedState()
	if state == nil {
		t.Fatal("Expected exported state to be non-nil")
	}

	// The last operation is copy, so we should have a FileOp with Copy action
	fileOp := state.Op().Op().GetFile()
	if fileOp == nil {
		t.Fatal("Expected FileOp")
	}

	copyAction := fileOp.Actions[0].GetCopy()
	if copyAction == nil {
		t.Fatal("Expected Copy action")
	}

	if copyAction.Mode != 0644 {
		t.Errorf("Expected copy mode 0644, got %o", copyAction.Mode)
	}

	if copyAction.Owner == nil {
		t.Fatal("Expected Owner to be set in copy")
	}

	// Check the owner values
	if copyAction.Owner.User.GetByName() == nil {
		t.Error("Expected user to be set by name in copy")
	}

	if copyAction.Owner.User.GetByName().Name != "root" {
		t.Errorf("Expected user name 'root' in copy, got '%s'", copyAction.Owner.User.GetByName().Name)
	}

	if copyAction.Owner.Group.GetByName() == nil {
		t.Error("Expected group to be set by name in copy")
	}

	if copyAction.Owner.Group.GetByName().Name != "root" {
		t.Errorf("Expected group name 'root' in copy, got '%s'", copyAction.Owner.Group.GetByName().Name)
	}
}
