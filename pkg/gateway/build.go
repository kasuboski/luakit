package gateway

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	"github.com/kasuboski/luakit/pkg/resolver"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
)

const (
	defaultEntrypoint = "build.lua"
)

type BuildOpts struct {
	Entrypoint string
}

func WithEntrypoint(path string) BuildOpt {
	return func(o *BuildOpts) {
		o.Entrypoint = path
	}
}

type BuildOpt func(*BuildOpts)

func Build(ctx context.Context, c gwclient.Client, opts ...BuildOpt) (*gwclient.Result, error) {
	options := &BuildOpts{
		Entrypoint: defaultEntrypoint,
	}
	for _, opt := range opts {
		opt(options)
	}

	luaSource, err := readLuaFile(ctx, c, options.Entrypoint)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", options.Entrypoint, err)
	}

	if len(luaSource) == 0 {
		return nil, fmt.Errorf("no lua source code provided")
	}

	result, err := evaluateLua(luaSource, c.BuildOpts().Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate lua script: %w", err)
	}

	if result.State == nil {
		return nil, fmt.Errorf("no bk.export() call — nothing to build")
	}

	gwResolver := resolver.NewGatewayResolver(c)

	def, err := dag.Serialize(result.State, &dag.SerializeOptions{
		ImageConfig: result.ImageConfig,
		SourceFiles: result.SourceFiles,
		Resolver:    gwResolver,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize definition: %w", err)
	}

	if len(def.Def) == 0 {
		return nil, fmt.Errorf("empty definition")
	}

	res, err := c.Solve(ctx, gwclient.SolveRequest{
		Definition: def,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to solve definition: %w", err)
	}

	return res, nil
}

func readLuaFile(ctx context.Context, c gwclient.Client, filename string) ([]byte, error) {
	inputs, err := c.Inputs(ctx)
	if err != nil || len(inputs) == 0 {
		inputs = map[string]llb.State{
			"context": llb.Local("context"),
		}
	}

	stateCtx, ok := inputs["context"]
	if !ok {
		stateCtx = llb.Local("context")
	}

	llbDef, err := stateCtx.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context state: %w", err)
	}

	res, err := c.Solve(ctx, gwclient.SolveRequest{
		Definition: llbDef.ToPB(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to solve context: %w", err)
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, fmt.Errorf("failed to get reference from result: %w", err)
	}

	data, err := ref.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	return data, nil
}

func stripSyntaxDirective(source []byte) []byte {
	lines := strings.Split(string(source), "\n")
	for len(lines) > 0 {
		line := strings.TrimSpace(lines[0])
		if line == "" {
			lines = lines[1:]
			continue
		}
		if strings.HasPrefix(line, "# syntax=") || strings.HasPrefix(line, "#syntax=") {
			lines = lines[1:]
			continue
		}
		break
	}
	return []byte(strings.Join(lines, "\n"))
}

func evaluateLua(source []byte, frontendOpts map[string]string) (*luavm.EvalResult, error) {
	for k, v := range frontendOpts {
		_ = os.Setenv(k, v)
	}

	source = stripSyntaxDirective(source)

	result, err := luavm.Evaluate(strings.NewReader(string(source)), "build.lua", nil)
	if err != nil {
		return nil, err
	}

	if result.State == nil {
		return nil, fmt.Errorf("no bk.export() call")
	}

	return result, nil
}
