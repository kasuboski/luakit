package luavm

import (
	"fmt"
	"io"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func Evaluate(r io.Reader, filename string, config *VMConfig) (*EvalResult, error) {
	ResetSourceFiles()

	scriptData, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read script: %w", err)
	}

	RegisterSourceFile(filename, scriptData)

	L := NewVM(config)
	defer L.Close()

	data := getVMData(L)
	if data == nil {
		return nil, fmt.Errorf("failed to get vm data")
	}

	fn, err := L.Load(strings.NewReader(string(scriptData)), filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load script: %w", err)
	}

	L.Push(fn)

	if err := L.PCall(0, lua.MultRet, nil); err != nil {
		return nil, fmt.Errorf("failed to run script: %w", err)
	}

	return &EvalResult{
		State:       data.exportedState,
		ImageConfig: data.exportedImageConfig,
		SourceFiles: GetAllSourceFiles(),
	}, nil
}

func EvaluateFile(path string, config *VMConfig) (*EvalResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return Evaluate(f, path, config)
}
