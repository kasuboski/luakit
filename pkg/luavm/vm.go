package luavm

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

var (
	sourceFiles map[string][]byte
	sourceMu    sync.RWMutex
)

func init() {
	sourceFiles = make(map[string][]byte)
}

type VMConfig struct {
	BuildContextDir string
	StdlibDir       string
}

func NewVM(config *VMConfig) *lua.LState {
	L := lua.NewState()

	if config == nil {
		config = &VMConfig{}
	}

	data := &vmData{}
	data.L = L
	L.SetGlobal("__luakit_vm_data", L.NewUserData())
	L.GetGlobal("__luakit_vm_data").(*lua.LUserData).Value = data

	registerStateType(L)
	registerMountType(L)
	registerPlatformType(L)
	registerAPI(L)
	sandbox(L)

	if config.BuildContextDir != "" || config.StdlibDir != "" {
		setupModuleLoader(L, config)
	}

	return L
}

func setupModuleLoader(L *lua.LState, config *VMConfig) {
	loader := L.NewFunction(func(L *lua.LState) int {
		moduleName := L.CheckString(1)

		searchPaths := []string{}

		if strings.HasSuffix(moduleName, ".lua") {
			if config.BuildContextDir != "" {
				searchPaths = append(searchPaths, filepath.Join(config.BuildContextDir, moduleName))
			}
			if config.StdlibDir != "" {
				searchPaths = append(searchPaths, filepath.Join(config.StdlibDir, moduleName))
			}
		} else {
			if config.BuildContextDir != "" {
				searchPaths = append(searchPaths, filepath.Join(config.BuildContextDir, moduleName+".lua"))
				searchPaths = append(searchPaths, filepath.Join(config.BuildContextDir, moduleName, "init.lua"))
			}
			if config.StdlibDir != "" {
				searchPaths = append(searchPaths, filepath.Join(config.StdlibDir, moduleName+".lua"))
				searchPaths = append(searchPaths, filepath.Join(config.StdlibDir, moduleName, "init.lua"))
			}
		}

		var moduleData []byte
		var moduleFile string
		for _, path := range searchPaths {
			data, err := os.ReadFile(path)
			if err == nil {
				moduleData = data
				moduleFile = path
				break
			}
		}

		if moduleData == nil {
			return 0
		}

		RegisterSourceFile(moduleFile, moduleData)

		fn, err := L.Load(strings.NewReader(string(moduleData)), moduleFile)
		if err != nil {
			L.RaiseError("error loading module '%s' from '%s': %v", moduleName, moduleFile, err)
			return 0
		}

		L.Push(fn)

		if err := L.PCall(0, 1, nil); err != nil {
			L.RaiseError("error running module '%s': %v", moduleName, err)
			return 0
		}

		moduleCache := L.GetField(L.GetGlobal("package"), "loaded")
		L.SetField(moduleCache, moduleName, L.Get(-1))

		return 1
	})

	loaders := L.GetField(L.GetGlobal("package"), "loaders")
	if loaders.Type() == lua.LTTable {
		loadersTable := loaders.(*lua.LTable)
		for i := loadersTable.Len() + 1; i >= 2; i-- {
			val := L.RawGet(loadersTable, lua.LNumber(i-1))
			L.RawSet(loadersTable, lua.LNumber(i), val)
		}
		L.RawSet(loadersTable, lua.LNumber(1), loader)
	}

	var newPathComponents []string
	if config.BuildContextDir != "" {
		newPathComponents = append(newPathComponents, filepath.Join(config.BuildContextDir, "?.lua"))
		newPathComponents = append(newPathComponents, filepath.Join(config.BuildContextDir, "?", "init.lua"))
	}
	if config.StdlibDir != "" {
		newPathComponents = append(newPathComponents, filepath.Join(config.StdlibDir, "?.lua"))
		newPathComponents = append(newPathComponents, filepath.Join(config.StdlibDir, "?", "init.lua"))
	}

	if len(newPathComponents) > 0 {
		currentPath := L.GetField(L.GetGlobal("package"), "path")
		var currentPathStr string
		if currentPath.Type() == lua.LTString {
			currentPathStr = currentPath.String()
		}
		if currentPathStr != "" {
			newPathComponents = append(newPathComponents, currentPathStr)
		}
		L.SetField(L.GetGlobal("package"), "path", lua.LString(strings.Join(newPathComponents, ";")))
	}
}

func sandbox(L *lua.LState) {
	os := L.GetGlobal("os")
	if os != lua.LNil {
		L.SetField(os, "execute", lua.LNil)
		L.SetField(os, "exit", lua.LNil)
		L.SetField(os, "remove", lua.LNil)
		L.SetField(os, "rename", lua.LNil)
		L.SetField(os, "tmpname", lua.LNil)
	}

	io := L.GetGlobal("io")
	if io != lua.LNil {
		L.SetField(io, "open", lua.LNil)
		L.SetField(io, "popen", lua.LNil)
		L.SetField(io, "input", lua.LNil)
		L.SetField(io, "output", lua.LNil)
		L.SetField(io, "lines", lua.LNil)
	}

	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("debug", lua.LNil)
}

func RegisterSourceFile(filename string, data []byte) {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	sourceFiles[filename] = data
}

func GetAllSourceFiles() map[string][]byte {
	sourceMu.RLock()
	defer sourceMu.RUnlock()
	result := make(map[string][]byte, len(sourceFiles))
	maps.Copy(result, sourceFiles)
	return result
}

func ResetSourceFiles() {
	sourceMu.Lock()
	defer sourceMu.Unlock()
	sourceFiles = make(map[string][]byte)
}
