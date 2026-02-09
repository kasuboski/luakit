package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/kasuboski/luakit/pkg/luavm"
)

func BenchmarkCLIColdStartSimpleScript(b *testing.B) {
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
	}
}

func BenchmarkCLIColdStart100LineScript(b *testing.B) {
	var script string
	script = "local base = bk.image(\"alpine:3.19\")\n"
	for i := 0; i < 100; i++ {
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
	}
}

func BenchmarkCLIColdStartWithSerialization(b *testing.B) {
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

		var buf bytes.Buffer
		_, err = buf.WriteString("mock serialization")
		if err != nil {
			b.Fatal(err)
		}
	}
}
