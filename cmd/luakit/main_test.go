package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	"github.com/kasuboski/luakit/pkg/output"
)

func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple key=value",
			input:    "key=value",
			expected: []string{"key", "value"},
		},
		{
			name:     "key with equals in value",
			input:    "key=value=with=equals",
			expected: []string{"key", "value=with=equals"},
		},
		{
			name:     "no equals",
			input:    "keyvalue",
			expected: nil,
		},
		{
			name:     "empty key",
			input:    "=value",
			expected: []string{"", "value"},
		},
		{
			name:     "empty value",
			input:    "key=",
			expected: []string{"key", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitKeyValue(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
			} else {
				if len(result) != len(tt.expected) {
					t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
					return
				}
				for i, v := range tt.expected {
					if result[i] != v {
						t.Errorf("expected [%d] to be %s, got %s", i, v, result[i])
					}
				}
			}
		})
	}
}

func TestParseBuildFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedOut  string
		expectedArgs map[string]string
	}{
		{
			name:         "no flags",
			args:         []string{"luakit", "build", "script.lua"},
			expectedOut:  "",
			expectedArgs: map[string]string{},
		},
		{
			name:         "output flag",
			args:         []string{"luakit", "build", "-o", "output.pb", "script.lua"},
			expectedOut:  "output.pb",
			expectedArgs: map[string]string{},
		},
		{
			name:         "output long flag",
			args:         []string{"luakit", "build", "--output", "output.pb", "script.lua"},
			expectedOut:  "output.pb",
			expectedArgs: map[string]string{},
		},
		{
			name:         "frontend-arg flag",
			args:         []string{"luakit", "build", "--frontend-arg", "key=value", "script.lua"},
			expectedOut:  "",
			expectedArgs: map[string]string{"key": "value"},
		},
		{
			name:         "multiple frontend-arg flags",
			args:         []string{"luakit", "build", "--frontend-arg", "key1=value1", "--frontend-arg", "key2=value2", "script.lua"},
			expectedOut:  "",
			expectedArgs: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:         "combined flags",
			args:         []string{"luakit", "build", "-o", "output.pb", "--frontend-arg", "key=value", "script.lua"},
			expectedOut:  "output.pb",
			expectedArgs: map[string]string{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			flags := parseBuildFlags()

			if flags.outputPath != tt.expectedOut {
				t.Errorf("expected output path %s, got %s", tt.expectedOut, flags.outputPath)
			}

			if len(flags.frontendArgs) != len(tt.expectedArgs) {
				t.Errorf("expected %d frontend args, got %d", len(tt.expectedArgs), len(flags.frontendArgs))
			}

			for k, v := range tt.expectedArgs {
				if flags.frontendArgs[k] != v {
					t.Errorf("expected frontend arg %s=%s, got %s=%s", k, v, k, flags.frontendArgs[k])
				}
			}
		})
	}
}

func TestParseDagFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedFmt string
		expectedOut string
	}{
		{
			name:        "no flags",
			args:        []string{"luakit", "dag", "script.lua"},
			expectedFmt: "dot",
			expectedOut: "",
		},
		{
			name:        "format dot",
			args:        []string{"luakit", "dag", "--format", "dot", "script.lua"},
			expectedFmt: "dot",
			expectedOut: "",
		},
		{
			name:        "format json",
			args:        []string{"luakit", "dag", "--format", "json", "script.lua"},
			expectedFmt: "json",
			expectedOut: "",
		},
		{
			name:        "output flag",
			args:        []string{"luakit", "dag", "-o", "output.dot", "script.lua"},
			expectedFmt: "dot",
			expectedOut: "output.dot",
		},
		{
			name:        "combined flags",
			args:        []string{"luakit", "dag", "--format", "json", "-o", "output.json", "script.lua"},
			expectedFmt: "json",
			expectedOut: "output.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			flags := parseDagFlags()

			if flags.format != tt.expectedFmt {
				t.Errorf("expected format %s, got %s", tt.expectedFmt, flags.format)
			}

			if flags.outputPath != tt.expectedOut {
				t.Errorf("expected output path %s, got %s", tt.expectedOut, flags.outputPath)
			}
		})
	}
}

func TestDOTWriter(t *testing.T) {
	state := createTestState(t)

	writer := output.NewDOTWriter("")
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := writer.Write(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_ = w.Close()
	}()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = w.Close()
	os.Stdout = oldStdout

	<-done

	output := buf.String()
	if !strings.Contains(output, "digraph dag") {
		t.Errorf("DOT output should contain 'digraph dag', got: %s", output)
	}
}

func TestJSONWriter(t *testing.T) {
	state := createTestState(t)

	writer := output.NewJSONWriter("")
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := writer.Write(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		_ = w.Close()
	}()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = w.Close()
	os.Stdout = oldStdout

	<-done

	output := buf.String()
	if !strings.Contains(output, "digest") {
		t.Errorf("JSON output should contain 'digest', got: %s", output)
	}
	if !strings.Contains(output, "type") {
		t.Errorf("JSON output should contain 'type', got: %s", output)
	}
}

func createTestState(t *testing.T) *dag.State {
	t.Helper()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	result, err := luavm.Evaluate(strings.NewReader(script), "test.lua", nil)
	if err != nil {
		t.Fatalf("failed to run test script: %v", err)
	}

	state := result.State
	if state == nil {
		t.Fatal("no exported state")
	}

	return state
}

func TestGetScriptArg(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "script at end",
			args:     []string{"luakit", "build", "script.lua"},
			expected: "script.lua",
		},
		{
			name:     "script with output flag",
			args:     []string{"luakit", "build", "-o", "output.pb", "script.lua"},
			expected: "script.lua",
		},
		{
			name:     "script with multiple flags",
			args:     []string{"luakit", "build", "-o", "output.pb", "--frontend-arg", "key=value", "script.lua"},
			expected: "script.lua",
		},
		{
			name:     "no script",
			args:     []string{"luakit", "build", "-o", "output.pb"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			args := getScriptArg()
			if args.script != tt.expected {
				t.Errorf("expected script %s, got %s", tt.expected, args.script)
			}
		})
	}
}

func TestOutputWriterToFile(t *testing.T) {
	tmpDir := t.TempDir()

	state := createTestState(t)

	outputFile := filepath.Join(tmpDir, "test.json")
	writer := output.NewJSONWriter(outputFile)

	if err := writer.Write(state); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Error("output file should not be empty")
	}

	if !strings.Contains(string(data), "digest") {
		t.Error("output should contain 'digest'")
	}
}

func TestParseDagFlagsWithFilter(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedFmt    string
		expectedOut    string
		expectedFilter string
	}{
		{
			name:           "filter Exec",
			args:           []string{"luakit", "dag", "--filter=Exec", "script.lua"},
			expectedFmt:    "dot",
			expectedOut:    "",
			expectedFilter: "Exec",
		},
		{
			name:           "filter Source",
			args:           []string{"luakit", "dag", "--filter", "Source", "script.lua"},
			expectedFmt:    "dot",
			expectedOut:    "",
			expectedFilter: "Source",
		},
		{
			name:           "filter with output",
			args:           []string{"luakit", "dag", "--filter=Exec", "-o", "out.dot", "script.lua"},
			expectedFmt:    "dot",
			expectedOut:    "out.dot",
			expectedFilter: "Exec",
		},
		{
			name:           "no filter",
			args:           []string{"luakit", "dag", "script.lua"},
			expectedFmt:    "dot",
			expectedOut:    "",
			expectedFilter: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			flags := parseDagFlags()

			if flags.format != tt.expectedFmt {
				t.Errorf("expected format %s, got %s", tt.expectedFmt, flags.format)
			}

			if flags.outputPath != tt.expectedOut {
				t.Errorf("expected output path %s, got %s", tt.expectedOut, flags.outputPath)
			}

			if flags.filterOp != tt.expectedFilter {
				t.Errorf("expected filter %s, got %s", tt.expectedFilter, flags.filterOp)
			}
		})
	}
}
