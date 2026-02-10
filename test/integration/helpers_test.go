package integration

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	pb "github.com/moby/buildkit/solver/pb"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	luakitPath := filepath.Join(wd, "..", "..", "dist", "luakit")
	if _, err := os.Stat(luakitPath); err != nil {
		log.Fatalf("luakit binary not found at %s. Build it first with: go build -o dist/luakit ./cmd/luakit", luakitPath)
	}
	os.Exit(m.Run())
}

func runLuakitBuild(t *testing.T, scriptPath, contextDir string) ([]byte, error) {
	t.Helper()

	wd, _ := os.Getwd()
	luakitPath := filepath.Join(wd, "..", "..", "dist", "luakit")

	cmd := exec.Command(luakitPath, "build", "-o", "-", scriptPath)
	cmd.Dir = contextDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("luakit output: %s", string(output))
		return nil, err
	}

	return output, nil
}

func requireValidDefinition(t *testing.T, def []byte) *pb.Definition {
	t.Helper()

	var pbDef pb.Definition
	require.NoError(t, pbDef.UnmarshalVT(def), "definition should unmarshal successfully")
	require.NotEmpty(t, pbDef.Def, "definition should have ops")

	return &pbDef
}

func requireSourceMapping(t *testing.T, pbDef *pb.Definition) {
	t.Helper()

	require.NotNil(t, pbDef.Source, "definition should have source info")
	require.NotNil(t, pbDef.Source.Infos, "source should have infos")
	require.NotEmpty(t, pbDef.Source.Infos, "source should have at least one source info")

	for digest, locations := range pbDef.Source.Locations {
		require.NotEmpty(t, locations.Locations, "digest %s should have locations", digest)
		for _, loc := range locations.Locations {
			require.GreaterOrEqual(t, loc.SourceIndex, int32(0), "source index should be non-negative")
			require.NotEmpty(t, loc.Ranges, "location should have ranges")
		}
	}
}

func requireDeterministic(t *testing.T, script string) {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))

	def1, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "first build should succeed")

	def2, err := runLuakitBuild(t, scriptPath, ".")
	require.NoError(t, err, "second build should succeed")

	require.Equal(t, def1, def2, "output should be deterministic")
}

func requireNoSourceOp(t *testing.T, pbDef *pb.Definition) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		require.Nil(t, op.GetSource(), "should not have any source ops")
	}
}

func requireExecOpCount(t *testing.T, pbDef *pb.Definition, count int) {
	t.Helper()

	execCount := 0
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if op.GetExec() != nil {
			execCount++
		}
	}
	require.Equal(t, count, execCount, "should have expected number of exec ops")
}

func requireSourceOpCount(t *testing.T, pbDef *pb.Definition, count int) {
	t.Helper()

	sourceCount := 0
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if op.GetSource() != nil {
			sourceCount++
		}
	}
	require.Equal(t, count, sourceCount, "should have expected number of source ops")
}

func requireFileOpCount(t *testing.T, pbDef *pb.Definition, count int) {
	t.Helper()

	fileCount := 0
	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if op.GetFile() != nil {
			fileCount++
		}
	}
	require.Equal(t, count, fileCount, "should have expected number of file ops")
}

func requireMergeOp(t *testing.T, pbDef *pb.Definition, inputCount int) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if merge := op.GetMerge(); merge != nil {
			require.Equal(t, inputCount, len(merge.Inputs), "merge op should have expected number of inputs")
			return
		}
	}
	t.Fatal("definition should have a merge op")
}

func requireDiffOp(t *testing.T, pbDef *pb.Definition) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if op.GetDiff() != nil {
			require.Equal(t, 2, len(op.Inputs), "diff op should have exactly 2 inputs")
			return
		}
	}
	t.Fatal("definition should have a diff op")
}

func requireMountOfType(t *testing.T, pbDef *pb.Definition, mountType pb.MountType, dest string) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			for _, mount := range exec.Mounts {
				if mount.MountType == mountType && mount.Dest == dest {
					return
				}
			}
		}
	}
	t.Fatalf("definition should contain %v mount at %s", mountType, dest)
}

func requireExecMeta(t *testing.T, pbDef *pb.Definition, cwd, user string, env []string) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			require.Equal(t, cwd, exec.Meta.Cwd)
			require.Equal(t, user, exec.Meta.User)
			for _, e := range env {
				found := false
				for _, execEnv := range exec.Meta.Env {
					if strings.Contains(execEnv, "="+e) || execEnv == e {
						found = true
						break
					}
				}
				require.True(t, found, "env should contain %s", e)
			}
			return
		}
	}
	t.Fatal("definition should have exec op with expected meta")
}

func requirePlatform(t *testing.T, pbDef *pb.Definition, os, arch, variant string) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if op.Platform != nil {
			require.Equal(t, os, op.Platform.OS)
			require.Equal(t, arch, op.Platform.Architecture)
			require.Equal(t, variant, op.Platform.Variant)
			return
		}
	}
	t.Fatal("definition should have op with expected platform")
}

func requireNetworkMode(t *testing.T, pbDef *pb.Definition, mode pb.NetMode) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			require.Equal(t, mode, exec.Network)
			return
		}
	}
	t.Fatal("definition should have exec op with expected network mode")
}

func requireSecurityMode(t *testing.T, pbDef *pb.Definition, mode pb.SecurityMode) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if exec := op.GetExec(); exec != nil {
			require.Equal(t, mode, exec.Security)
			return
		}
	}
	t.Fatal("definition should have exec op with expected security mode")
}

func requireImageConfig(t *testing.T, pbDef *pb.Definition, entrypoint, cmd []string, env map[string]string) {
	t.Helper()

	for _, meta := range pbDef.Metadata {
		if desc := meta.GetDescription(); desc != nil {
			if configStr, ok := desc["moby.buildkit.image.config"]; ok {
				require.NotEmpty(t, configStr)
				return
			}
		}
	}
	t.Fatal("definition should have image config in metadata")
}

func requireSourceIdentifier(t *testing.T, pbDef *pb.Definition, identifier string) {
	t.Helper()

	for _, opBytes := range pbDef.Def {
		var op pb.Op
		require.NoError(t, op.UnmarshalVT(opBytes))
		if source := op.GetSource(); source != nil {
			require.Equal(t, identifier, source.GetIdentifier())
			return
		}
	}
	t.Fatalf("definition should have source op with identifier %s", identifier)
}

func createTestScript(t *testing.T, script string) string {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "build.lua")
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))
	return scriptPath
}

func createTestScriptForGolden(t *testing.T, script, scriptName string) string {
	t.Helper()

	wd, _ := os.Getwd()
	scriptDir := filepath.Join(wd, "testdata", "golden_scripts")
	require.NoError(t, os.MkdirAll(scriptDir, 0755))

	scriptPath := filepath.Join(scriptDir, scriptName)
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0644))
	return scriptPath
}

func createLuaVMAndRun(t *testing.T, script string) (*dag.State, interface{}) {
	t.Helper()

	scriptPath := createTestScript(t, script)
	scriptData, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	luavm.RegisterSourceFile(scriptPath, scriptData)

	result, err := luavm.EvaluateFile(scriptPath, nil)
	require.NoError(t, err)
	require.NotNil(t, result.State, "should have exported state")

	return result.State, result.ImageConfig
}
