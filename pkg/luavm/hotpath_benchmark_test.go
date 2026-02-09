package luavm

import (
	"testing"
)

func BenchmarkEvalHotPath100LineScript(b *testing.B) {
	var script string
	script = "local base = bk.image(\"alpine:3.19\")\n"
	for i := 0; i < 100; i++ {
		script += "local s" + string(rune('0'+i%10)) + " = base:run(\"echo test" + string(rune('0'+i%10)) + "\")\n"
	}
	script += "bk.export(s0)\n"

	config := &VMConfig{}
	L := NewVM(config)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}

		state := GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}
	}
	L.Close()
}

func BenchmarkEvalHotPathSimpleScript(b *testing.B) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`

	config := &VMConfig{}
	L := NewVM(config)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}

		state := GetExportedState()
		if state == nil {
			b.Fatal("no exported state")
		}
	}
	L.Close()
}

func BenchmarkImageCallOnly(b *testing.B) {
	config := &VMConfig{}
	L := NewVM(config)
	defer L.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		if err := L.DoString(`local base = bk.image("alpine:3.19")`); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRunCallOnly(b *testing.B) {
	config := &VMConfig{}
	L := NewVM(config)
	defer L.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		if err := L.DoString(`local base = bk.image("alpine:3.19"); local result = base:run("echo test")`); err != nil {
			b.Fatal(err)
		}
	}
}
