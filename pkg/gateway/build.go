package gateway

import (
	"context"
	"fmt"
	"os"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/luavm"
	"github.com/moby/buildkit/client/llb"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	"github.com/moby/buildkit/solver/pb"
	"github.com/moby/buildkit/util/apicaps"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/pkg/errors"
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

	buildOpts := c.BuildOpts()

	if err := validateCaps(buildOpts.Caps); err != nil {
		return nil, err
	}

	luaSource, err := readLuaFile(ctx, c, options.Entrypoint)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %s", options.Entrypoint)
	}

	if len(luaSource) == 0 {
		return nil, errors.Errorf("no lua source code provided")
	}

	state, imageConfig, err := evaluateLua(luaSource, buildOpts.Opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to evaluate lua script")
	}

	if state == nil {
		return nil, errors.Errorf("no bk.export() call — nothing to build")
	}

	def, err := dag.Serialize(state, &dag.SerializeOptions{
		ImageConfig: imageConfig,
		SourceFiles: luavm.GetAllSourceFiles(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to serialize definition")
	}

	if len(def.Def) == 0 {
		return nil, errors.New("empty definition")
	}

	res, err := c.Solve(ctx, gwclient.SolveRequest{
		Definition: def,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to solve definition")
	}

	return res, nil
}

func readLuaFile(ctx context.Context, c gwclient.Client, filename string) ([]byte, error) {
	inputs, err := c.Inputs(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get inputs")
	}

	if len(inputs) == 0 {
		return nil, errors.Errorf("no build context provided. Provide at least the 'context' input")
	}

	stateCtx, ok := inputs["context"]
	if !ok {
		return nil, errors.Errorf("required input 'context' not found. Available inputs: %v", getAvailableInputNames(inputs))
	}

	if len(inputs) > 1 {
		var unexpectedInputs []string
		for name := range inputs {
			if name != "context" {
				unexpectedInputs = append(unexpectedInputs, name)
			}
		}
		if len(unexpectedInputs) > 0 {
			return nil, errors.Errorf("unsupported input(s) provided: %v. Currently only 'context' input is supported", unexpectedInputs)
		}
	}

	llbDef, err := stateCtx.Marshal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal context state")
	}

	res, err := c.Solve(ctx, gwclient.SolveRequest{
		Definition: llbDef.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to solve context")
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reference from result")
	}

	data, err := ref.ReadFile(ctx, gwclient.ReadRequest{
		Filename: filename,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", filename)
	}

	return data, nil
}

func getAvailableInputNames(inputs map[string]llb.State) []string {
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	return names
}

func evaluateLua(source []byte, frontendOpts map[string]string) (state *dag.State, imageConfig *dockerspec.DockerOCIImage, err error) {
	luavm.ResetExportedState()
	luavm.ResetSourceFiles()

	luavm.RegisterSourceFile("build.lua", source)

	L := luavm.NewVM(nil)
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("lua evaluation panic: %v", r)
		}
		L.Close()
	}()

	for k, v := range frontendOpts {
		os.Setenv(k, v)
	}

	if err := L.DoString(string(source)); err != nil {
		return nil, nil, err
	}

	state = luavm.GetExportedState()
	if state == nil {
		return nil, nil, fmt.Errorf("no bk.export() call")
	}

	imageConfig = luavm.GetExportedImageConfig()

	return state, imageConfig, nil
}

func validateCaps(caps apicaps.CapSet) error {
	if err := caps.Supports(pb.CapFileBase); err != nil {
		return errors.Wrap(err, "needs BuildKit 0.5 or later")
	}
	return nil
}
