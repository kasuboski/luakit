package luavm

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestExecSecurityOptionsExample(t *testing.T) {
	resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")

		local all_options = base:run("echo test", {
			network = "none",
			security = "sandbox",
			user = "builder",
			cwd = "/app",
			hostname = "builder",
			valid_exit_codes = {0, 1, 2}
		})

		bk.export(all_options)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	state := GetExportedState()
	execOp := state.Op().Op().GetExec()

	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	// Test network mode
	if execOp.Network != pb.NetMode_NONE {
		t.Errorf("Expected network mode NONE, got %v", execOp.Network)
	}

	// Test security mode
	if execOp.Security != pb.SecurityMode_SANDBOX {
		t.Errorf("Expected security mode SANDBOX, got %v", execOp.Security)
	}

	// Test user (running as non-root)
	if execOp.Meta.User != "builder" {
		t.Errorf("Expected user 'builder', got '%s'", execOp.Meta.User)
	}

	// Test cwd
	if execOp.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd '/app', got '%s'", execOp.Meta.Cwd)
	}

	// Test hostname
	if execOp.Meta.Hostname != "builder" {
		t.Errorf("Expected hostname 'builder', got '%s'", execOp.Meta.Hostname)
	}

	// Test valid_exit_codes
	if len(execOp.Meta.ValidExitCodes) != 3 {
		t.Errorf("Expected 3 valid exit codes, got %d", len(execOp.Meta.ValidExitCodes))
	}

	if execOp.Meta.ValidExitCodes[0] != 0 {
		t.Errorf("Expected exit code 0, got %d", execOp.Meta.ValidExitCodes[0])
	}

	if execOp.Meta.ValidExitCodes[1] != 1 {
		t.Errorf("Expected exit code 1, got %d", execOp.Meta.ValidExitCodes[1])
	}

	if execOp.Meta.ValidExitCodes[2] != 2 {
		t.Errorf("Expected exit code 2, got %d", execOp.Meta.ValidExitCodes[2])
	}
}
