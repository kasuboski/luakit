package main

import (
	"bytes"
	"os"
	"strings"
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
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dag.ClearDigestCache()

		config := &luavm.VMConfig{}
		result, err := luavm.EvaluateFile(tmpFile.Name(), config)
		if err != nil {
			b.Fatal(err)
		}

		state := result.State
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
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(script); err != nil {
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dag.ClearDigestCache()

		config := &luavm.VMConfig{}
		result, err := luavm.EvaluateFile(tmpFile.Name(), config)
		if err != nil {
			b.Fatal(err)
		}

		state := result.State
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
	defer func() { _ = os.Remove(tmpFile.Name()) }()

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

		dag.ClearDigestCache()

		b.StartTimer()

		config := &luavm.VMConfig{}
		result, err := luavm.EvaluateFile(tmpFile.Name(), config)
		if err != nil {
			b.Fatal(err)
		}

		state := result.State
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
		config := &luavm.VMConfig{}
		result, err := luavm.Evaluate(strings.NewReader(script), "benchmark.lua", config)
		if err != nil {
			b.Fatal(err)
		}

		state := result.State
		if state == nil {
			b.Fatal("no exported state")
		}
	}
}

func BenchmarkOptimizedSerializeOnly(b *testing.B) {
	script := `local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)`

	config := &luavm.VMConfig{}
	result, err := luavm.Evaluate(strings.NewReader(script), "benchmark.lua", config)
	if err != nil {
		b.Fatal(err)
	}

	state := result.State
	if state == nil {
		b.Fatal("no exported state")
	}

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
		config := &luavm.VMConfig{}
		result, err := luavm.Evaluate(strings.NewReader(script), "benchmark.lua", config)
		if err != nil {
			b.Fatal(err)
		}
		state := result.State
		if state == nil {
			b.Fatal("no exported state")
		}
	}
}
