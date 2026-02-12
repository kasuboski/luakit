package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	"github.com/kasuboski/luakit/pkg/output"
	"github.com/kasuboski/luakit/pkg/resolver"
	pb "github.com/moby/buildkit/solver/pb"
)

const version = "0.1.0-dev"

func getStdlibDir() string {
	if dir := os.Getenv("LUAKIT_STDLIB_DIR"); dir != "" {
		return dir
	}
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "..", "share", "luakit", "stdlib")
}

func createVMConfig(scriptPath string) *luavm.VMConfig {
	scriptDir := filepath.Dir(scriptPath)
	stdlibDir := getStdlibDir()

	config := &luavm.VMConfig{
		BuildContextDir: scriptDir,
		StdlibDir:       stdlibDir,
	}

	if _, err := os.Stat(stdlibDir); os.IsNotExist(err) {
		config.StdlibDir = ""
	}

	return config
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		handleBuild()
	case "dag":
		handleDag()
	case "validate":
		handleValidate()
	case "version", "--version", "-v":
		fmt.Printf("luakit %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command) // #nosec G705 -- CLI tool output to stderr
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `luakit - Lua frontend for BuildKit

USAGE:
    luakit build <script>     Build from a Lua script
    luakit dag <script>       Print the LLB DAG without building
    luakit validate <script>  Validate a script without building
    luakit version            Print version information

BUILD FLAGS:
    --output, -o <path>         Write pb.Definition to file (default: stdout)
    --frontend-arg KEY=VALUE    Set a frontend argument (repeatable)

 DAG FLAGS:
     --format <dot|json>         Output format (default: dot)
     --output, -o <path>         Write to file (default: stdout)
     --filter <type>              Filter by operation type (Exec, Source, File, Merge, Diff)

EXAMPLES:
    luakit build build.lua
    luakit build -o output.pb build.lua
    luakit build --frontend-arg=target=linux/arm64 build.lua
    luakit dag build.lua | dot -Tsvg > dag.svg
    luakit dag --format=json build.lua
    luakit validate build.lua
`)
}

type buildFlags struct {
	outputPath   string
	frontendArgs map[string]string
}

func parseBuildFlags() *buildFlags {
	flags := &buildFlags{
		frontendArgs: make(map[string]string),
	}

	args := os.Args[2:]
	i := 0
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "--output", "-o":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: %s requires a value\n", arg) // #nosec G705 -- CLI tool output to stderr
				os.Exit(1)
			}
			flags.outputPath = args[i+1]
			i += 2
		case "--frontend-arg":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --frontend-arg requires a value\n")
				os.Exit(1)
			}
			parts := splitKeyValue(args[i+1])
			if parts == nil {
				fmt.Fprintf(os.Stderr, "error: --frontend-arg value must be in KEY=VALUE format\n")
				os.Exit(1)
			}
			flags.frontendArgs[parts[0]] = parts[1]
			i += 2
		case "--help", "-h":
			fmt.Fprintf(os.Stderr, `luakit build - Build from a Lua script

USAGE:
    luakit build [flags] <script>

FLAGS:
    --output, -o <path>         Write pb.Definition to file (default: stdout)
    --frontend-arg KEY=VALUE    Set a frontend argument (repeatable)
    --help, -h                  Show this help message

EXAMPLES:
    luakit build build.lua
    luakit build -o output.pb build.lua
    luakit build --frontend-arg=target=linux/arm64 build.lua
`)
			os.Exit(0)
		default:
			if arg[0] == '-' {
				fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", arg) // #nosec G705 -- CLI tool output to stderr
				os.Exit(1)
			}
			i++
		}
	}

	return flags
}

func splitKeyValue(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func handleBuild() {
	flags := parseBuildFlags()

	args := getScriptArg()
	if args.script == "" {
		fmt.Fprintln(os.Stderr, "error: missing script file")
		fmt.Fprintln(os.Stderr, "Usage: luakit build [flags] <script>")
		os.Exit(1)
	}

	scriptData, err := os.ReadFile(args.script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to read script: %v\n", err)
		os.Exit(1)
	}

	config := createVMConfig(args.script)

	for k, v := range flags.frontendArgs {
		_ = os.Setenv(k, v)
	}

	result, err := luavm.Evaluate(strings.NewReader(string(scriptData)), args.script, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if result.State == nil {
		fmt.Fprintln(os.Stderr, "error: no bk.export() call — nothing to build")
		os.Exit(1)
	}

	var def *pb.Definition
	reslv := resolver.NewResolver()
	def, err = dag.Serialize(result.State, &dag.SerializeOptions{
		ImageConfig: result.ImageConfig,
		SourceFiles: result.SourceFiles,
		Resolver:    reslv,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to serialize definition: %v\n", err)
		os.Exit(1)
	}

	writer := output.NewProtobufWriter(flags.outputPath)
	if err = writer.Write(def); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to write output: %v\n", err)
		os.Exit(1)
	}
}

type dagFlags struct {
	format     string
	outputPath string
	filterOp   string
}

func parseDagFlags() *dagFlags {
	flags := &dagFlags{
		format: "dot",
	}

	args := os.Args[2:]
	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--format":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --format requires a value\n")
				os.Exit(1)
			}
			flags.format = args[i+1]
			if flags.format != "dot" && flags.format != "json" {
				fmt.Fprintf(os.Stderr, "error: --format must be 'dot' or 'json'\n")
				os.Exit(1)
			}
			i += 2
		case len(arg) > 9 && arg[:9] == "--format=":
			flags.format = arg[9:]
			if flags.format != "dot" && flags.format != "json" {
				fmt.Fprintf(os.Stderr, "error: --format must be 'dot' or 'json'\n")
				os.Exit(1)
			}
			i += 1
		case arg == "--output" || arg == "-o":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: %s requires a value\n", arg) // #nosec G705 -- CLI tool output to stderr
				os.Exit(1)
			}
			flags.outputPath = args[i+1]
			i += 2
		case arg == "--filter":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --filter requires a value\n")
				os.Exit(1)
			}
			flags.filterOp = args[i+1]
			validOps := map[string]bool{"Exec": true, "Source": true, "File": true, "Merge": true, "Diff": true, "Build": true}
			if !validOps[flags.filterOp] {
				fmt.Fprintf(os.Stderr, "error: --filter must be one of: Exec, Source, File, Merge, Diff, Build\n")
				os.Exit(1)
			}
			i += 2
		case len(arg) > 9 && arg[:9] == "--filter=":
			flags.filterOp = arg[9:]
			validOps := map[string]bool{"Exec": true, "Source": true, "File": true, "Merge": true, "Diff": true, "Build": true}
			if !validOps[flags.filterOp] {
				fmt.Fprintf(os.Stderr, "error: --filter must be one of: Exec, Source, File, Merge, Diff, Build\n")
				os.Exit(1)
			}
			i += 1
		case arg == "--help" || arg == "-h":
			fmt.Fprintf(os.Stderr, `luakit dag - Print the LLB DAG without building

USAGE:
    luakit dag [flags] <script>

 FLAGS:
     --format <dot|json>         Output format (default: dot)
     --output, -o <path>         Write to file (default: stdout)
     --filter <type>              Filter by operation type (Exec, Source, File, Merge, Diff)
     --help, -h                  Show this help message

 EXAMPLES:
     luakit dag build.lua
     luakit dag build.lua | dot -Tsvg > dag.svg
     luakit dag --format=json build.lua
     luakit dag --filter=Exec build.lua | dot -Tsvg > exec-only.svg
 `)
			os.Exit(0)
		default:
			if arg[0] == '-' {
				fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", arg) // #nosec G705 -- CLI tool output to stderr
				os.Exit(1)
			}
			i++
		}
	}

	return flags
}

func handleDag() {
	flags := parseDagFlags()

	args := getScriptArg()
	if args.script == "" {
		fmt.Fprintln(os.Stderr, "error: missing script file")
		fmt.Fprintln(os.Stderr, "Usage: luakit dag [flags] <script>")
		os.Exit(1)
	}

	config := createVMConfig(args.script)

	result, err := luavm.EvaluateFile(args.script, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if result.State == nil {
		fmt.Fprintln(os.Stderr, "error: no bk.export() call — nothing to build")
		os.Exit(1)
	}

	switch flags.format {
	case "dot":
		writer := output.NewDOTWriter(flags.outputPath)
		if flags.filterOp != "" {
			writer.SetFilter(flags.filterOp)
		}
		if err := writer.Write(result.State); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write DOT: %v\n", err)
			os.Exit(1)
		}
	case "json":
		writer := output.NewJSONWriter(flags.outputPath)
		if flags.filterOp != "" {
			writer.SetFilter(flags.filterOp)
		}
		if err := writer.Write(result.State); err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to write JSON: %v\n", err)
			os.Exit(1)
		}
	}
}

func handleValidate() {
	if err := validateScript(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Script is valid")
}

func validateScript() error {
	args := getScriptArg()
	if args.script == "" {
		return fmt.Errorf("missing script file\nUsage: luakit validate <script>")
	}

	scriptData, err := os.ReadFile(args.script)
	if err != nil {
		return fmt.Errorf("failed to read script: %w", err)
	}

	config := createVMConfig(args.script)

	result, err := luavm.Evaluate(strings.NewReader(string(scriptData)), args.script, config)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if result.State == nil {
		return fmt.Errorf("no bk.export() call found in script")
	}

	return nil
}

type scriptArgs struct {
	script string
}

func getScriptArg() *scriptArgs {
	args := os.Args[2:]
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg[0] != '-' {
			return &scriptArgs{script: arg}
		}
		if arg == "--output" || arg == "-o" || arg == "--frontend-arg" || arg == "--format" {
			i += 2
		} else {
			i++
		}
	}
	return &scriptArgs{script: ""}
}
