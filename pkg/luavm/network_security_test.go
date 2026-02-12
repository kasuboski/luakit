package luavm

import (
	"fmt"
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNetworkSecurityOptionsSerialization(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			network = "none",
			security = "insecure",
			user = "builder",
			cwd = "/app"
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if execOp.Network != pb.NetMode_NONE {
		t.Errorf("Expected network mode NONE, got %v", execOp.Network)
	}

	if execOp.Security != pb.SecurityMode_INSECURE {
		t.Errorf("Expected security mode INSECURE, got %v", execOp.Security)
	}

	if execOp.Meta.User != "builder" {
		t.Errorf("Expected user to be 'builder', got '%s'", execOp.Meta.User)
	}

	if execOp.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd to be '/app', got '%s'", execOp.Meta.Cwd)
	}
}

func TestNetworkModes(t *testing.T) {
	resetExportedState()

	tests := []struct {
		name     string
		network  string
		expected pb.NetMode
	}{
		{"sandbox", "sandbox", pb.NetMode_UNSET},
		{"host", "host", pb.NetMode_HOST},
		{"none", "none", pb.NetMode_NONE},
		{"default", "", pb.NetMode_UNSET},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportedState()
			L := NewVM(nil)
			testVM = L
			t.Cleanup(func() { L.Close() })
			t.Cleanup(func() { testVM = nil })

			script := `local base = bk.image("alpine:3.19"); local result = base:run("echo test", { network = "` + tt.network + `" }); bk.export(result)`

			if err := L.DoString(script); err != nil {
				t.Fatalf("Failed to execute Lua script: %v", err)
			}

			state := GetExportedState()
			execOp := state.Op().Op().GetExec()

			if execOp.Network != tt.expected {
				t.Errorf("Expected network mode %v, got %v", tt.expected, execOp.Network)
			}
		})
	}
}

func TestSecurityModes(t *testing.T) {
	resetExportedState()

	tests := []struct {
		name     string
		security string
		expected pb.SecurityMode
	}{
		{"sandbox", "sandbox", pb.SecurityMode_SANDBOX},
		{"insecure", "insecure", pb.SecurityMode_INSECURE},
		{"default", "", pb.SecurityMode_SANDBOX},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportedState()
			L := NewVM(nil)
			testVM = L
			t.Cleanup(func() { L.Close() })
			t.Cleanup(func() { testVM = nil })

			script := `local base = bk.image("alpine:3.19"); local result = base:run("echo test", { security = "` + tt.security + `" }); bk.export(result)`

			if err := L.DoString(script); err != nil {
				t.Fatalf("Failed to execute Lua script: %v", err)
			}

			state := GetExportedState()
			execOp := state.Op().Op().GetExec()

			if execOp.Security != tt.expected {
				t.Errorf("Expected security mode %v, got %v", tt.expected, execOp.Security)
			}
		})
	}
}

func TestHostnameOption(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			hostname = "builder"
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if execOp.Meta.Hostname != "builder" {
		t.Errorf("Expected hostname 'builder', got '%s'", execOp.Meta.Hostname)
	}
}

func TestValidExitCodesOption(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			valid_exit_codes = {0, 1}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Meta.ValidExitCodes) != 2 {
		t.Errorf("Expected 2 valid exit codes, got %d", len(execOp.Meta.ValidExitCodes))
	}

	if execOp.Meta.ValidExitCodes[0] != 0 {
		t.Errorf("Expected first exit code 0, got %d", execOp.Meta.ValidExitCodes[0])
	}

	if execOp.Meta.ValidExitCodes[1] != 1 {
		t.Errorf("Expected second exit code 1, got %d", execOp.Meta.ValidExitCodes[1])
	}
}

func TestAllExecOptionsTogether(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			network = "none",
			security = "sandbox",
			user = "builder",
			cwd = "/app",
			hostname = "builder",
			valid_exit_codes = {0, 1},
			env = {
				FOO = "bar",
				BAZ = "qux"
			}
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if execOp.Network != pb.NetMode_NONE {
		t.Errorf("Expected network mode NONE, got %v", execOp.Network)
	}

	if execOp.Security != pb.SecurityMode_SANDBOX {
		t.Errorf("Expected security mode SANDBOX, got %v", execOp.Security)
	}

	if execOp.Meta.User != "builder" {
		t.Errorf("Expected user 'builder', got '%s'", execOp.Meta.User)
	}

	if execOp.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd '/app', got '%s'", execOp.Meta.Cwd)
	}

	if execOp.Meta.Hostname != "builder" {
		t.Errorf("Expected hostname 'builder', got '%s'", execOp.Meta.Hostname)
	}

	if len(execOp.Meta.ValidExitCodes) != 2 {
		t.Errorf("Expected 2 valid exit codes, got %d", len(execOp.Meta.ValidExitCodes))
	}

	if len(execOp.Meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(execOp.Meta.Env))
	}
}

func TestSingleExitCode(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			valid_exit_codes = 1
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Meta.ValidExitCodes) != 1 {
		t.Errorf("Expected 1 valid exit code, got %d", len(execOp.Meta.ValidExitCodes))
	}

	if execOp.Meta.ValidExitCodes[0] != 1 {
		t.Errorf("Expected exit code 1, got %d", execOp.Meta.ValidExitCodes[0])
	}
}

func TestExitCodeRange(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			valid_exit_codes = "0..5"
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	expected := []int32{0, 1, 2, 3, 4, 5}
	if len(execOp.Meta.ValidExitCodes) != len(expected) {
		t.Errorf("Expected %d valid exit codes, got %d", len(expected), len(execOp.Meta.ValidExitCodes))
	}

	for i, code := range expected {
		if execOp.Meta.ValidExitCodes[i] != code {
			t.Errorf("Expected exit code %d at index %d, got %d", code, i, execOp.Meta.ValidExitCodes[i])
		}
	}
}

func TestExitCodeRangeLarge(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local result = base:run("echo test", {
			valid_exit_codes = "200..255"
		})
		bk.export(result)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if len(execOp.Meta.ValidExitCodes) != 56 {
		t.Errorf("Expected 56 valid exit codes, got %d", len(execOp.Meta.ValidExitCodes))
	}

	if execOp.Meta.ValidExitCodes[0] != 200 {
		t.Errorf("Expected first exit code 200, got %d", execOp.Meta.ValidExitCodes[0])
	}

	if execOp.Meta.ValidExitCodes[55] != 255 {
		t.Errorf("Expected last exit code 255, got %d", execOp.Meta.ValidExitCodes[55])
	}
}

func TestExitCodeRangeInvalid(t *testing.T) {
	resetExportedState()

	tests := []struct {
		name  string
		input string
	}{
		{"Invalid format", "0-5"},
		{"Start greater than end", "5..0"},
		{"Negative start", "-1..5"},
		{"End too large", "0..256"},
		{"Invalid characters", "a..b"},
		{"Missing end", "0.."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetExportedState()
			L := NewVM(nil)
			testVM = L
			t.Cleanup(func() { L.Close() })
			t.Cleanup(func() { testVM = nil })

			script := fmt.Sprintf(`
				local base = bk.image("alpine:3.19")
				local result = base:run("echo test", {
					valid_exit_codes = "%s"
				})
				bk.export(result)
			`, tt.input)

			err := L.DoString(script)
			if err == nil {
				t.Errorf("Expected error for input '%s', got nil", tt.input)
			}
		})
	}
}
