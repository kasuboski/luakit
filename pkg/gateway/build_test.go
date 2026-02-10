package gateway

import (
	"testing"

	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	"github.com/stretchr/testify/require"
)

func TestEvaluateLua(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{
			name: "simple build",
			source: `local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /hello.txt")
bk.export(result)`,
			wantErr: false,
		},
		{
			name: "no export",
			source: `local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /hello.txt")`,
			wantErr: true,
		},
		{
			name:    "empty source",
			source:  ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluateLua([]byte(tt.source), nil)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluateLuaWithConfig(t *testing.T) {
	source := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /hello.txt")
bk.export(result, {
    entrypoint = {"/bin/sh"},
    env = {PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
    user = "root",
    workdir = "/app",
})`

	result, err := evaluateLua([]byte(source), nil)
	require.NoError(t, err)
	require.NotNil(t, result.ImageConfig)
	require.Equal(t, []string{"/bin/sh"}, result.ImageConfig.Config.Entrypoint)
	require.Contains(t, result.ImageConfig.Config.Env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	require.Equal(t, "root", result.ImageConfig.Config.User)
	require.Equal(t, "/app", result.ImageConfig.Config.WorkingDir)
}

func TestValidateCaps(t *testing.T) {
	tests := []struct {
		name    string
		caps    apicaps.CapSet
		wantErr bool
	}{
		{
			name:    "valid caps",
			caps:    pb.Caps.CapSet(pb.Caps.All()),
			wantErr: false,
		},
		{
			name:    "empty caps",
			caps:    pb.Caps.CapSet(nil),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCaps(tt.caps)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
