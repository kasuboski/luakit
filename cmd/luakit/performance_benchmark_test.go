package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
)

func BenchmarkOptimizedCLIColdStartSimple(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	tmpFile, err := os.CreateTemp("", "*.lua")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		luavm.ResetSourceFiles()
		luavm.ResetExportedState()
		dag.ClearDigestCache()

		scriptData, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			b.Fatal(err)
		}

		luavm.RegisterSourceFile(tmpFile.Name(), scriptData)

		config := &luavm.VMConfig{}
		L := luavm.NewVM(config)
		defer L.Close()

		if err = L.DoFile(tmpFile.Name()); err != nil {
			b.Fatal(err)
		}

		state := luavm.GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}

		def, err := dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}

		if len(def.Def) == 0 {
			b.Fatal("empty definition")
		}
	}
}

func BenchmarkOptimizedCLIColdStart100Line(b *testing.B) {
	var script string
	script = "local base = bk.image(\"alpine:3.19\")\n"
	for i := range 100 {
		script += "local s" + string(rune('0'+i%10)) + " = base:run(\"echo test" + string(rune('0'+i%10)) + "\")\n"
	}
	script += "bk.export(s0)\n"

	tmpFile, err := os.CreateTemp("", "*.lua")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		luavm.ResetSourceFiles()
		luavm.ResetExportedState()
		dag.ClearDigestCache()

		scriptData, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			b.Fatal(err)
		}

		luavm.RegisterSourceFile(tmpFile.Name(), scriptData)

		config := &luavm.VMConfig{}
		L := luavm.NewVM(config)
		defer L.Close()

		if err = L.DoFile(tmpFile.Name()); err != nil {
			b.Fatal(err)
		}

		state := luavm.GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}

		def, err := dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}

		if len(def.Def) == 0 {
			b.Fatal("empty definition")
		}
	}
}

func BenchmarkOptimizedFullWorkflow(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	tmpFile, err := os.CreateTemp("", "*.lua")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		luavm.ResetSourceFiles()
		luavm.ResetExportedState()
		dag.ClearDigestCache()

		scriptData, _ := os.ReadFile(tmpFile.Name())
		luavm.RegisterSourceFile(tmpFile.Name(), scriptData)

		b.StartTimer()

		config := &luavm.VMConfig{}
		L := luavm.NewVM(config)

		if err = L.DoFile(tmpFile.Name()); err != nil {
			b.Fatal(err)
		}

		state := luavm.GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}

		_, err = dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}

		var buf bytes.Buffer
		output := buf.Bytes()
		_ = output

		L.Close()
		b.StopTimer()
	}
}

func BenchmarkOptimizedVMOnly(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		luavm.ResetSourceFiles()
		luavm.ResetExportedState()

		config := &luavm.VMConfig{}
		L := luavm.NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}

		state := luavm.GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}
	}
}

func BenchmarkOptimizedSerializeOnly(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	luavm.ResetSourceFiles()
	luavm.ResetExportedState()

	config := &luavm.VMConfig{}
	L := luavm.NewVM(config)

	if err := L.DoString(script); err != nil {
		b.Fatal(err)
	}

	state := luavm.GetExportedState()
	if state == nil {
		b.Fatal("no exported state")
	}
	L.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOptimizedEvalOnly(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		luavm.ResetSourceFiles()
		luavm.ResetExportedState()

		config := &luavm.VMConfig{}
		L := luavm.NewVM(config)
		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
		state := luavm.GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}
		L.Close()
	}
}
