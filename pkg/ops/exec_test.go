package ops

import (
	"strings"
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	"github.com/stretchr/testify/require"
)

func TestNewExecOp(t *testing.T) {
	cmd := []string{"/bin/sh", "-c", "echo hello"}
	opts := &ExecOptions{
		Env:  []string{"PATH=/usr/bin", "FOO=bar"},
		Cwd:  "/app",
		User: "nobody",
	}

	op := NewExecOp(cmd, opts)

	if op.Meta.Args[0] != "/bin/sh" {
		t.Errorf("Expected args[0] to be '/bin/sh', got '%s'", op.Meta.Args[0])
	}

	if op.Meta.Args[1] != "-c" {
		t.Errorf("Expected args[1] to be '-c', got '%s'", op.Meta.Args[1])
	}

	if op.Meta.Args[2] != "echo hello" {
		t.Errorf("Expected args[2] to be 'echo hello', got '%s'", op.Meta.Args[2])
	}

	if len(op.Meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(op.Meta.Env))
	}

	if op.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd to be '/app', got '%s'", op.Meta.Cwd)
	}

	if op.Meta.User != "nobody" {
		t.Errorf("Expected user to be 'nobody', got '%s'", op.Meta.User)
	}
}

func TestNewExecOpWithNoOptions(t *testing.T) {
	cmd := []string{"ls", "-la"}

	op := NewExecOp(cmd, nil)

	if len(op.Meta.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(op.Meta.Args))
	}

	if op.Meta.Args[0] != "ls" {
		t.Errorf("Expected args[0] to be 'ls', got '%s'", op.Meta.Args[0])
	}

	if op.Meta.Cwd != "" {
		t.Errorf("Expected cwd '', got '%s'", op.Meta.Cwd)
	}
}

func TestRun(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	cmd := []string{"/bin/sh", "-c", "echo test"}
	opts := &ExecOptions{
		Env: []string{"TEST=1"},
	}

	execState := Run(sourceState, cmd, opts, "test.lua", 20)

	if execState == nil {
		t.Fatal("Expected non-nil exec state")
	}

	execOp := execState.Op().Op().GetExec()
	if execOp == nil {
		t.Fatal("Expected ExecOp")
	}

	if execState.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", execState.Op().LuaFile())
	}

	if execState.Op().LuaLine() != 20 {
		t.Errorf("Expected Lua line 20, got %d", execState.Op().LuaLine())
	}

	if len(execState.Op().Inputs()) != 1 {
		t.Errorf("Expected 1 input, got %d", len(execState.Op().Inputs()))
	}

	if len(execOp.Meta.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(execOp.Meta.Env))
	}
}

func TestRunWithEmptyCommand(t *testing.T) {
	sourceOp := NewSourceOp("scratch", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	cmd := []string{}
	result := Run(sourceState, cmd, nil, "test.lua", 20)

	if result != nil {
		t.Error("Expected nil state for empty command")
	}
}

func TestNewExecState(t *testing.T) {
	sourceOp := NewSourceOp("docker-image://docker.io/library/alpine:3.19", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	execOp := NewExecOp([]string{"echo", "hello"}, nil)
	execState := NewExecState(sourceState, execOp, "test.lua", 20)

	if execState == nil {
		t.Fatal("Expected non-nil exec state")
	}

	pbOp := execState.Op().Op()
	if len(pbOp.Inputs) != 1 {
		t.Errorf("Expected 1 input, got %d", len(pbOp.Inputs))
	}

	if pbOp.Inputs[0].Index != 0 {
		t.Errorf("Expected input index 0, got %d", pbOp.Inputs[0].Index)
	}

	if len(execState.Op().Inputs()) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(execState.Op().Inputs()))
	}
}

func TestNewExecOpWithNetworkMode(t *testing.T) {
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
			network := tt.network
			opts := &ExecOptions{
				Network: &network,
			}

			op := NewExecOp([]string{"echo", "test"}, opts)

			if op.Network != tt.expected {
				t.Errorf("Expected network mode %v, got %v", tt.expected, op.Network)
			}
		})
	}
}

func TestNewExecOpWithSecurityMode(t *testing.T) {
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
			security := tt.security
			opts := &ExecOptions{
				Security: &security,
			}

			op := NewExecOp([]string{"echo", "test"}, opts)

			if op.Security != tt.expected {
				t.Errorf("Expected security mode %v, got %v", tt.expected, op.Security)
			}
		})
	}
}

func TestNewExecOpWithAllOptions(t *testing.T) {
	network := "none"
	security := "sandbox"
	cmd := []string{"/bin/sh", "-c", "echo test"}
	opts := &ExecOptions{
		Env:      []string{"PATH=/usr/bin", "FOO=bar"},
		Cwd:      "/app",
		User:     "nobody",
		Mounts:   []*Mount{},
		Network:  &network,
		Security: &security,
	}

	op := NewExecOp(cmd, opts)

	if op.Network != pb.NetMode_NONE {
		t.Errorf("Expected network mode NONE, got %v", op.Network)
	}

	if op.Security != pb.SecurityMode_SANDBOX {
		t.Errorf("Expected security mode SANDBOX, got %v", op.Security)
	}

	if op.Meta.Cwd != "/app" {
		t.Errorf("Expected cwd to be '/app', got '%s'", op.Meta.Cwd)
	}

	if op.Meta.User != "nobody" {
		t.Errorf("Expected user to be 'nobody', got '%s'", op.Meta.User)
	}

	if len(op.Meta.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(op.Meta.Env))
	}
}

func TestNewExecOpWithHostname(t *testing.T) {
	opts := &ExecOptions{
		Hostname: "builder",
	}

	op := NewExecOp([]string{"echo", "test"}, opts)

	if op.Meta.Hostname != "builder" {
		t.Errorf("Expected hostname 'builder', got '%s'", op.Meta.Hostname)
	}
}

func TestNewExecOpWithValidExitCodes(t *testing.T) {
	opts := &ExecOptions{
		ValidExitCodes: []int32{0, 1},
	}

	op := NewExecOp([]string{"echo", "test"}, opts)

	if len(op.Meta.ValidExitCodes) != 2 {
		t.Errorf("Expected 2 valid exit codes, got %d", len(op.Meta.ValidExitCodes))
	}

	if op.Meta.ValidExitCodes[0] != 0 {
		t.Errorf("Expected first exit code 0, got %d", op.Meta.ValidExitCodes[0])
	}

	if op.Meta.ValidExitCodes[1] != 1 {
		t.Errorf("Expected second exit code 1, got %d", op.Meta.ValidExitCodes[1])
	}
}

func TestParseNetworkMode(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.NetMode
	}{
		{"sandbox", pb.NetMode_UNSET},
		{"host", pb.NetMode_HOST},
		{"none", pb.NetMode_NONE},
		{"", pb.NetMode_UNSET},
		{"invalid", pb.NetMode_UNSET},
	}

	for _, tt := range tests {
		result := parseNetworkMode(tt.input)
		if result != tt.expected {
			t.Errorf("parseNetworkMode(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseSecurityMode(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.SecurityMode
	}{
		{"sandbox", pb.SecurityMode_SANDBOX},
		{"insecure", pb.SecurityMode_INSECURE},
		{"", pb.SecurityMode_SANDBOX},
		{"invalid", pb.SecurityMode_SANDBOX},
	}

	for _, tt := range tests {
		result := parseSecurityMode(tt.input)
		if result != tt.expected {
			t.Errorf("parseSecurityMode(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		name     string
		imageEnv []string
		userEnv  []string
		expected []string
	}{
		{
			name:     "empty image env",
			imageEnv: []string{},
			userEnv:  []string{"FOO=bar"},
			expected: []string{"FOO=bar"},
		},
		{
			name:     "empty user env",
			imageEnv: []string{"PATH=/usr/bin", "HOME=/root"},
			userEnv:  []string{},
			expected: []string{"PATH=/usr/bin", "HOME=/root"},
		},
		{
			name:     "merge env vars",
			imageEnv: []string{"PATH=/usr/bin", "HOME=/root"},
			userEnv:  []string{"FOO=bar"},
			expected: []string{"PATH=/usr/bin", "HOME=/root", "FOO=bar"},
		},
		{
			name:     "user env overrides image env",
			imageEnv: []string{"PATH=/usr/bin", "FOO=old"},
			userEnv:  []string{"FOO=new"},
			expected: []string{"PATH=/usr/bin", "FOO=new"},
		},
		{
			name:     "user env unsets variable",
			imageEnv: []string{"PATH=/usr/bin", "FOO=bar"},
			userEnv:  []string{"FOO"},
			expected: []string{"PATH=/usr/bin"},
		},
		{
			name:     "multiple overrides",
			imageEnv: []string{"PATH=/usr/bin", "HOME=/root", "USER=root"},
			userEnv:  []string{"PATH=/custom/path", "USER=nobody"},
			expected: []string{"PATH=/custom/path", "HOME=/root", "USER=nobody"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeEnv(tt.imageEnv, tt.userEnv)

			// Convert both to sets for easier comparison (order doesn't matter)
			resultSet := make(map[string]string)
			for _, env := range result {
				parts := []string{env, ""}
				if idx := strings.Index(env, "="); idx > 0 {
					parts[0] = env[:idx]
					parts[1] = env[idx+1:]
				}
				resultSet[parts[0]] = parts[1]
			}

			expectedSet := make(map[string]string)
			for _, env := range tt.expected {
				parts := []string{env, ""}
				if idx := strings.Index(env, "="); idx > 0 {
					parts[0] = env[:idx]
					parts[1] = env[idx+1:]
				}
				expectedSet[parts[0]] = parts[1]
			}

			if len(resultSet) != len(expectedSet) {
				t.Errorf("Expected %d env vars, got %d", len(expectedSet), len(resultSet))
			}

			for k, v := range expectedSet {
				if resultSet[k] != v {
					t.Errorf("Expected %s=%s, got %s=%s", k, v, k, resultSet[k])
				}
			}
		})
	}
}

// TestNewExecStateWithOpts_AddsRootfsMount verifies that all exec operations have a rootfs mount.
// This is REQUIRED by BuildKit - the validation in BuildKit explicitly checks that every
// exec operation has a mount with Dest="/" and will reject operations without it with
// the error "invalid exec op with no rootfs". Additionally, BuildKit assumes the rootfs
// mount is at index 0. The automatic rootfs mount logic is implemented in NewExecStateWithOpts.
func TestNewExecStateWithOpts_AddsRootfsMount(t *testing.T) {
	sourceOp := NewSourceOp("docker-image://alpine:3.19", nil)
	sourceState := NewSourceState(sourceOp, "test.lua", 10)

	t.Run("exec with no mounts adds rootfs", func(t *testing.T) {
		execOp := NewExecOp([]string{"echo", "hello"}, nil)
		execState := NewExecState(sourceState, execOp, "test.lua", 20)

		require.NotNil(t, execState)
		pbOp := execState.Op().Op().GetExec()
		require.NotNil(t, pbOp)
		require.Len(t, pbOp.Mounts, 1, "exec op should have exactly one mount (rootfs)")

		rootfsMount := pbOp.Mounts[0]
		require.Equal(t, "/", rootfsMount.GetDest(), "first mount should be rootfs at /")
		require.Equal(t, pb.MountType_BIND, rootfsMount.GetMountType(), "rootfs should be BIND type")
		require.Equal(t, int64(0), rootfsMount.GetInput(), "rootfs should reference input index 0")
		require.False(t, rootfsMount.GetReadonly(), "rootfs should not be readonly")
	})

	t.Run("exec with existing mounts prepends rootfs", func(t *testing.T) {
		cacheMount := CacheMount("/cache", nil)
		opts := &ExecOptions{Mounts: []*Mount{cacheMount}}
		execState := Run(sourceState, []string{"echo", "test"}, opts, "test.lua", 30)

		require.NotNil(t, execState)
		pbOp := execState.Op().Op().GetExec()
		require.NotNil(t, pbOp)
		require.Len(t, pbOp.Mounts, 2, "exec op should have 2 mounts (rootfs + cache)")

		rootfsMount := pbOp.Mounts[0]
		require.Equal(t, "/", rootfsMount.GetDest(), "first mount should be rootfs at /")
		require.Equal(t, pb.MountType_BIND, rootfsMount.GetMountType(), "rootfs should be BIND type")
		require.Equal(t, int64(0), rootfsMount.GetInput(), "rootfs should reference input index 0")

		cachePbMount := pbOp.Mounts[1]
		require.Equal(t, "/cache", cachePbMount.GetDest(), "second mount should be cache mount")
		require.Equal(t, pb.MountType_CACHE, cachePbMount.GetMountType(), "second mount should be CACHE type")
	})
}
