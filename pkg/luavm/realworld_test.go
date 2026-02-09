package luavm

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRealWorldExamples(t *testing.T) {
	examples := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "Node.js web application",
			script:   "../../examples/real-world/nodejs/build.lua",
			expected: "node:20-alpine",
		},
		{
			name:     "Go microservice",
			script:   "../../examples/real-world/go/build.lua",
			expected: "golang:1.21-alpine",
		},
		{
			name:     "Python data science",
			script:   "../../examples/real-world/python/build.lua",
			expected: "python:3.11-slim",
		},
	}

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.script)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				t.Fatalf("Script file does not exist: %s", absPath)
			}

			defer ResetExportedState()

			L := NewVM(nil)
			defer L.Close()

			if err := L.DoFile(absPath); err != nil {
				t.Fatalf("Failed to execute Lua script: %v", err)
			}

			state := GetExportedState()
			if state == nil {
				t.Fatal("Expected exported state to be non-nil")
			}

			imageConfig := GetExportedImageConfig()
			if imageConfig == nil {
				t.Fatal("Expected image config to be set")
			}

			if imageConfig.Config.User == "" {
				t.Error("Expected user to be set in image config")
			}

			if imageConfig.Config.WorkingDir == "" {
				t.Error("Expected working directory to be set in image config")
			}

			if len(imageConfig.Config.Env) == 0 {
				t.Error("Expected environment variables to be set in image config")
			}
		})
	}
}

func TestNodeJSBuildStructure(t *testing.T) {
	defer ResetExportedState()

	L := NewVM(nil)
	defer L.Close()

	absPath, _ := filepath.Abs("../../examples/real-world/nodejs/build.lua")

	if err := L.DoFile(absPath); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	imageConfig := GetExportedImageConfig()
	if imageConfig == nil {
		t.Fatal("Expected image config to be set")
	}

	if imageConfig.Config.User != "nodejs" {
		t.Errorf("Expected user 'nodejs', got '%s'", imageConfig.Config.User)
	}

	if imageConfig.Config.WorkingDir != "/app" {
		t.Errorf("Expected workdir '/app', got '%s'", imageConfig.Config.WorkingDir)
	}

	nodeEnvFound := slices.Contains(imageConfig.Config.Env, "NODE_ENV=production")
	if !nodeEnvFound {
		t.Error("Expected NODE_ENV=production in environment variables")
	}
}

func TestGoBuildStructure(t *testing.T) {
	defer ResetExportedState()

	L := NewVM(nil)
	defer L.Close()

	absPath, _ := filepath.Abs("../../examples/real-world/go/build.lua")

	if err := L.DoFile(absPath); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	imageConfig := GetExportedImageConfig()
	if imageConfig == nil {
		t.Fatal("Expected image config to be set")
	}

	if imageConfig.Config.User != "app" {
		t.Errorf("Expected user 'app', got '%s'", imageConfig.Config.User)
	}

	if imageConfig.Config.WorkingDir != "/root" {
		t.Errorf("Expected workdir '/root', got '%s'", imageConfig.Config.WorkingDir)
	}

	tzEnvFound := slices.Contains(imageConfig.Config.Env, "TZ=UTC")
	if !tzEnvFound {
		t.Error("Expected TZ=UTC in environment variables")
	}
}

func TestPythonBuildStructure(t *testing.T) {
	defer ResetExportedState()

	L := NewVM(nil)
	defer L.Close()

	absPath, _ := filepath.Abs("../../examples/real-world/python/build.lua")

	if err := L.DoFile(absPath); err != nil {
		t.Fatalf("Failed to execute Lua script: %v", err)
	}

	imageConfig := GetExportedImageConfig()
	if imageConfig == nil {
		t.Fatal("Expected image config to be set")
	}

	if imageConfig.Config.User != "data-scientist" {
		t.Errorf("Expected user 'data-scientist', got '%s'", imageConfig.Config.User)
	}

	if imageConfig.Config.WorkingDir != "/workspace/notebooks" {
		t.Errorf("Expected workdir '/workspace/notebooks', got '%s'", imageConfig.Config.WorkingDir)
	}

	pythonEnvFound := slices.Contains(imageConfig.Config.Env, "PYTHONUNBUFFERED=1")
	if !pythonEnvFound {
		t.Error("Expected PYTHONUNBUFFERED=1 in environment variables")
	}
}
