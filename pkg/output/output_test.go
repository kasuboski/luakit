package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
)

func createTestState(t *testing.T) *dag.State {
	t.Helper()

	L := luavm.NewVM(nil)
	defer L.Close()

	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	if err := L.DoString(script); err != nil {
		t.Fatalf("failed to run test script: %v", err)
	}

	state := luavm.GetExportedState()
	if state == nil {
		t.Fatal("no exported state")
	}

	return state
}

func TestDOTWriter(t *testing.T) {
	t.Run("write to stdout", func(t *testing.T) {
		writer := NewDOTWriter("")

		if writer == nil {
			t.Error("NewDOTWriter returned nil")
		}
	})

	t.Run("write to file", func(t *testing.T) {
		state := createTestState(t)
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.dot")

		writer := NewDOTWriter(outputFile)
		if writer == nil {
			t.Error("NewDOTWriter returned nil")
		}

		if err := writer.Write(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)
		if !strings.Contains(output, "digraph dag") {
			t.Error("DOT output should contain 'digraph dag'")
		}
		if !strings.Contains(output, "rankdir=TB") {
			t.Error("DOT output should contain 'rankdir=TB'")
		}
	})
}

func TestJSONWriter(t *testing.T) {
	t.Run("write to stdout", func(t *testing.T) {
		writer := NewJSONWriter("")

		if writer == nil {
			t.Error("NewJSONWriter returned nil")
		}
	})

	t.Run("write to file", func(t *testing.T) {
		luavm.ResetExportedState()
		state := createTestState(t)
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.json")

		writer := NewJSONWriter(outputFile)
		if writer == nil {
			t.Error("NewJSONWriter returned nil")
		}

		if err := writer.Write(state); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)
		if !strings.Contains(output, "digest") {
			t.Error("JSON output should contain 'digest'")
		}
		if !strings.Contains(output, "type") {
			t.Error("JSON output should contain 'type'")
		}
		if !strings.Contains(output, "inputs") {
			t.Error("JSON output should contain 'inputs'")
		}
	})
}

func TestProtobufWriter(t *testing.T) {
	t.Run("write to stdout", func(t *testing.T) {
		luavm.ResetExportedState()
		exportedState := createTestState(t)

		if exportedState == nil {
			t.Skip("No exported state")
		}

		writer := NewProtobufWriter("")

		if writer == nil {
			t.Error("NewProtobufWriter returned nil")
		}
	})

	t.Run("write to file", func(t *testing.T) {
		luavm.ResetExportedState()
		exportedState := createTestState(t)

		if exportedState == nil {
			t.Skip("No exported state")
		}

		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.pb")

		writer := NewProtobufWriter(outputFile)

		if writer == nil {
			t.Error("NewProtobufWriter returned nil")
		}

		if err := writer.Write(nil); err != nil {
			t.Logf("expected error with nil definition: %v", err)
		}
	})
}

func TestDOTWriterWithFilter(t *testing.T) {
	luavm.ResetExportedState()
	state := createTestState(t)

	t.Run("filter Exec", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.dot")

		writer := NewDOTWriter(outputFile)
		writer.SetFilter("Exec")

		if err := writer.Write(state); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, "Exec") {
			t.Error("filtered output should contain Exec nodes")
		}

		if strings.Contains(output, "Source") {
			t.Error("filtered output should not contain Source nodes")
		}
	})

	t.Run("filter Source", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.dot")

		writer := NewDOTWriter(outputFile)
		writer.SetFilter("Source")

		if err := writer.Write(state); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, "Source") {
			t.Error("filtered output should contain Source nodes")
		}

		if strings.Contains(output, "Exec") {
			t.Error("filtered output should not contain Exec nodes")
		}
	})
}

func TestJSONWriterWithFilter(t *testing.T) {
	luavm.ResetExportedState()
	state := createTestState(t)

	t.Run("filter Exec", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.json")

		writer := NewJSONWriter(outputFile)
		writer.SetFilter("Exec")

		if err := writer.Write(state); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, `"type": "Exec"`) {
			t.Error("filtered output should contain Exec type")
		}

		if strings.Contains(output, `"type": "Source"`) {
			t.Error("filtered output should not contain Source type")
		}
	})

	t.Run("filter Source", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputFile := filepath.Join(tmpDir, "test.json")

		writer := NewJSONWriter(outputFile)
		writer.SetFilter("Source")

		if err := writer.Write(state); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, `"type": "Source"`) {
			t.Error("filtered output should contain Source type")
		}

		if strings.Contains(output, `"type": "Exec"`) {
			t.Error("filtered output should not contain Exec type")
		}
	})
}
