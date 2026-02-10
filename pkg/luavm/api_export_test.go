package luavm

import (
	"encoding/json"
	"testing"

	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	lua "github.com/yuin/gopher-lua"
)

func TestParseExportOptions(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	opts := L.NewTable()

	entrypoint := L.NewTable()
	entrypoint.RawSetInt(1, lua.LString("/bin/sh"))
	entrypoint.RawSetInt(2, lua.LString("-c"))
	entrypoint.RawSetInt(3, lua.LString("echo hello"))
	L.SetField(opts, "entrypoint", entrypoint)

	cmd := L.NewTable()
	cmd.RawSetInt(1, lua.LString("server"))
	cmd.RawSetInt(2, lua.LString("--port"))
	cmd.RawSetInt(3, lua.LString("8080"))
	L.SetField(opts, "cmd", cmd)

	env := L.NewTable()
	env.RawSetString("PATH", lua.LString("/usr/local/bin:/usr/bin:/bin"))
	env.RawSetString("NODE_ENV", lua.LString("production"))
	L.SetField(opts, "env", env)

	L.SetField(opts, "workdir", lua.LString("/app"))

	L.SetField(opts, "user", lua.LString("appuser"))

	labels := L.NewTable()
	labels.RawSetString("org.opencontainers.image.title", lua.LString("Test App"))
	labels.RawSetString("org.opencontainers.image.description", lua.LString("A test application"))
	L.SetField(opts, "labels", labels)

	expose := L.NewTable()
	expose.RawSetInt(1, lua.LString("8080/tcp"))
	expose.RawSetInt(2, lua.LString("9090/udp"))
	L.SetField(opts, "expose", expose)

	L.SetField(opts, "os", lua.LString("linux"))
	L.SetField(opts, "arch", lua.LString("arm64"))
	L.SetField(opts, "variant", lua.LString("v8"))

	config := parseExportOptions(L, opts)

	if config == nil {
		t.Fatal("parseExportOptions returned nil")
	}

	if len(config.Config.Entrypoint) != 3 {
		t.Errorf("Expected entrypoint length 3, got %d", len(config.Config.Entrypoint))
	}

	if config.Config.Entrypoint[0] != "/bin/sh" {
		t.Errorf("Expected entrypoint[0] '/bin/sh', got '%s'", config.Config.Entrypoint[0])
	}

	if len(config.Config.Cmd) != 3 {
		t.Errorf("Expected cmd length 3, got %d", len(config.Config.Cmd))
	}

	if config.Config.Cmd[0] != "server" {
		t.Errorf("Expected cmd[0] 'server', got '%s'", config.Config.Cmd[0])
	}

	expectedEnv := []string{"PATH=/usr/local/bin:/usr/bin:/bin", "NODE_ENV=production"}
	if len(config.Config.Env) != len(expectedEnv) {
		t.Errorf("Expected env length %d, got %d", len(expectedEnv), len(config.Config.Env))
	}

	if config.Config.WorkingDir != "/app" {
		t.Errorf("Expected working dir '/app', got '%s'", config.Config.WorkingDir)
	}

	if config.Config.User != "appuser" {
		t.Errorf("Expected user 'appuser', got '%s'", config.Config.User)
	}

	if len(config.Config.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(config.Config.Labels))
	}

	if config.Config.Labels["org.opencontainers.image.title"] != "Test App" {
		t.Errorf("Expected label 'Test App', got '%s'", config.Config.Labels["org.opencontainers.image.title"])
	}

	if len(config.Config.ExposedPorts) != 2 {
		t.Errorf("Expected 2 exposed ports, got %d", len(config.Config.ExposedPorts))
	}

	if _, ok := config.Config.ExposedPorts["8080/tcp"]; !ok {
		t.Error("Expected exposed port '8080/tcp'")
	}

	if config.OS != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", config.OS)
	}

	if config.Architecture != "arm64" {
		t.Errorf("Expected architecture 'arm64', got '%s'", config.Architecture)
	}

	if config.Variant != "v8" {
		t.Errorf("Expected variant 'v8', got '%s'", config.Variant)
	}
}

func TestParseExportOptionsMinimal(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	opts := L.NewTable()

	config := parseExportOptions(L, opts)

	if config == nil {
		t.Fatal("parseExportOptions returned nil")
	}

	if config.OS != "linux" {
		t.Errorf("Expected default OS 'linux', got '%s'", config.OS)
	}

	if config.Architecture != "amd64" {
		t.Errorf("Expected default architecture 'amd64', got '%s'", config.Architecture)
	}

	if len(config.Config.Env) != 0 {
		t.Errorf("Expected empty env, got %v", config.Config.Env)
	}
}

func TestExportImageConfigSerialization(t *testing.T) {
	config := &dockerspec.DockerOCIImage{}
	config.OS = "linux"
	config.Architecture = "amd64"
	config.Config.Env = []string{"PATH=/usr/bin"}
	config.Config.WorkingDir = "/app"
	config.Config.User = "appuser"
	config.Config.Entrypoint = []string{"/bin/sh"}
	config.Config.Cmd = []string{"-c", "echo hello"}
	config.Config.ExposedPorts = map[string]struct{}{"8080/tcp": {}}
	config.Config.Labels = map[string]string{"key": "value"}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var unmarshaled dockerspec.DockerOCIImage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if unmarshaled.OS != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", unmarshaled.OS)
	}

	if unmarshaled.Architecture != "amd64" {
		t.Errorf("Expected architecture 'amd64', got '%s'", unmarshaled.Architecture)
	}

	if len(unmarshaled.Config.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(unmarshaled.Config.Env))
	}

	if unmarshaled.Config.WorkingDir != "/app" {
		t.Errorf("Expected working dir '/app', got '%s'", unmarshaled.Config.WorkingDir)
	}
}

func TestWithMetadata(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local s = base:run("echo hello", {
			cwd = "/app",
			user = "builder"
		}):with_metadata({
			description = "Building application",
			progress_group = "build"
		})
		bk.export(s, {
			workdir = "/workspace",
			user = "appuser",
			expose = {"8080/tcp"}
		})
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	exportedState := GetExportedState()
	if exportedState == nil {
		t.Fatal("Expected exported state to be set")
	}

	op := exportedState.Op()
	if op == nil {
		t.Fatal("Expected op to be set")
	}

	metadata := op.Metadata()
	if metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if metadata.Description == nil {
		t.Fatal("Expected description map to be set")
	}

	if metadata.Description["llb.custom"] != "Building application" {
		t.Errorf("Expected description 'Building application', got '%s'", metadata.Description["llb.custom"])
	}

	if metadata.ProgressGroup == nil {
		t.Fatal("Expected progress group to be set")
	}

	if metadata.ProgressGroup.Id != "build" {
		t.Errorf("Expected progress group id 'build', got '%s'", metadata.ProgressGroup.Id)
	}
}

func TestWithMetadataOnlyDescription(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local s = base:run("echo hello"):with_metadata({
			description = "Simple build"
		})
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	exportedState := GetExportedState()
	if exportedState == nil {
		t.Fatal("Expected exported state to be set")
	}

	metadata := exportedState.Op().Metadata()
	if metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if metadata.Description == nil {
		t.Fatal("Expected description map to be set")
	}

	if metadata.Description["llb.custom"] != "Simple build" {
		t.Errorf("Expected description 'Simple build', got '%s'", metadata.Description["llb.custom"])
	}

	if metadata.ProgressGroup != nil {
		t.Error("Expected progress group to be nil")
	}
}

func TestWithMetadataOnlyProgressGroup(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local s = base:run("echo hello"):with_metadata({
			progress_group = "test-group"
		})
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	exportedState := GetExportedState()
	if exportedState == nil {
		t.Fatal("Expected exported state to be set")
	}

	metadata := exportedState.Op().Metadata()
	if metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if metadata.Description != nil {
		if len(metadata.Description) > 0 {
			t.Error("Expected description map to be empty or nil")
		}
	}

	if metadata.ProgressGroup == nil {
		t.Fatal("Expected progress group to be set")
	}

	if metadata.ProgressGroup.Id != "test-group" {
		t.Errorf("Expected progress group id 'test-group', got '%s'", metadata.ProgressGroup.Id)
	}
}

func TestWithMetadataNilOpts(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		local s = base:run("echo hello"):with_metadata({})
		bk.export(s)
	`

	if err := L.DoString(script); err != nil {
		t.Fatalf("Failed to execute script: %v", err)
	}

	exportedState := GetExportedState()
	if exportedState == nil {
		t.Fatal("Expected exported state to be set")
	}

	metadata := exportedState.Op().Metadata()
	if metadata == nil {
		t.Fatal("Expected metadata to be set")
	}
}

func TestWithMetadataInvalidType(t *testing.T) {
	defer resetExportedState()

	L := NewVM(nil)
	testVM = L
	defer L.Close()
	defer func() { testVM = nil }()

	script := `
		local base = bk.image("alpine:3.19")
		base:run("echo hello"):with_metadata(123)
	`

	err := L.DoString(script)
	if err == nil {
		t.Fatal("Expected error when with_metadata receives non-table argument")
	}

	errStr := err.Error()
	if len(errStr) < 5 {
		t.Fatalf("Expected error message, got: %v", err)
	}
}
