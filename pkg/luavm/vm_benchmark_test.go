package luavm

import (
	"testing"
)

func BenchmarkVMStartup(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		config := &VMConfig{}
		L := NewVM(config)
		L.Close()
	}
}

func BenchmarkVMStartupWithDirs(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		config := &VMConfig{
			BuildContextDir: "/tmp/test",
			StdlibDir:       "/tmp/stdlib",
		}
		L := NewVM(config)
		L.Close()
	}
}

func BenchmarkEvalSimpleScript(b *testing.B) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEval100LineScript(b *testing.B) {
	var script string
	script = "local base = bk.image(\"alpine:3.19\")\n"
	for i := range 100 {
		script += "local s" + string(rune('0'+i%10)) + " = base:run(\"echo test" + string(rune('0'+i%10)) + "\")\n"
	}
	script += "bk.export(s0)\n"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvalComplexScriptWithMounts(b *testing.B) {
	script := `
local base = bk.image("alpine:3.19")
local result = base:run("apk add --no-cache curl", {
	mounts = {
		bk.cache("/var/cache/apk", {sharing = "shared"}),
		bk.tmpfs("/tmp")
	}
})
bk.export(result)
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvalMultiStageBuild(b *testing.B) {
	script := `
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", {
	cwd = "/app",
	mounts = { bk.cache("/go/pkg/mod") },
})
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server", {mode = "0755"})
bk.export(final)
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvalMergeOperations(b *testing.B) {
	script := `
local base = bk.image("alpine:3.19")
local deps = base:run("apk add --no-cache git")
local source = base:run("mkdir -p /app/src")
local config = base:run("mkdir -p /app/config")
local merged = bk.merge(deps, source, config)
bk.export(merged)
`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateState(b *testing.B) {
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

func BenchmarkRunOperation(b *testing.B) {
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

func BenchmarkCopyOperation(b *testing.B) {
	config := &VMConfig{}
	L := NewVM(config)
	defer L.Close()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		script := `local base = bk.image("alpine:3.19"); local src = bk.local_("context"); local result = base:copy(src, ".", "/app")`
		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileOperations(b *testing.B) {
	script := `
local base = bk.image("alpine:3.19")
local s1 = base:mkdir("/app/data", {mode = "0755"})
local s2 = s1:mkfile("/app/data/config.json", '{"key": "value"}', {mode = "0644"})
local s3 = s2:symlink("/app/data/config.json", "/app/config")
local s4 = s3:rm("/app/data/config.json", {allow_not_found = true})
bk.export(s4)
`

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ResetSourceFiles()
		ResetExportedState()

		config := &VMConfig{}
		L := NewVM(config)
		defer L.Close()

		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLuaTableParsing(b *testing.B) {
	config := &VMConfig{}
	L := NewVM(config)
	defer L.Close()

	script := `
local t = {
	foo = "bar",
	baz = 42,
	qux = true,
	items = {1, 2, 3, 4, 5}
}
`
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCallSiteCapture(b *testing.B) {
	config := &VMConfig{}
	L := NewVM(config)
	defer L.Close()

	script := `
function test()
	local file, line = debug.getinfo(1, "Sl").source, debug.getinfo(1).currentline
	return file, line
end
for i = 1, 1000 do
	test()
end
`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := L.DoString(script); err != nil {
			b.Fatal(err)
		}
	}
}
