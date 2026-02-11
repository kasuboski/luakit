package ops

import (
	"strings"

	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

type ExecOptions struct {
	Env            []string
	Cwd            string
	User           string
	Mounts         []*Mount
	Network        *string
	Security       *string
	Hostname       string
	ValidExitCodes []int32
}

func NewExecOp(cmd []string, opts *ExecOptions) *pb.ExecOp {
	meta := &pb.Meta{
		Args: cmd,
	}

	mounts := []*pb.Mount{}

	if opts != nil {
		if len(opts.Env) > 0 {
			meta.Env = opts.Env
		}
		if opts.Cwd != "" {
			meta.Cwd = opts.Cwd
		}
		if opts.User != "" {
			meta.User = opts.User
		}
		for _, m := range opts.Mounts {
			mounts = append(mounts, m.ToPB())
		}
	}

	op := &pb.ExecOp{
		Meta:   meta,
		Mounts: mounts,
	}

	if opts != nil {
		if opts.Network != nil {
			op.Network = parseNetworkMode(*opts.Network)
		}
		if opts.Security != nil {
			op.Security = parseSecurityMode(*opts.Security)
		}
		if opts.Hostname != "" {
			meta.Hostname = opts.Hostname
		}
		if len(opts.ValidExitCodes) > 0 {
			meta.ValidExitCodes = opts.ValidExitCodes
		}
	}

	return op
}

// mergeEnv merges environment variables from image config with user-provided ones
// User-provided env vars override image config env vars
func mergeEnv(imageEnv []string, userEnv []string) []string {
	if len(imageEnv) == 0 {
		return userEnv
	}

	// Convert image env to map
	envMap := make(map[string]string)
	for _, env := range imageEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Apply user env vars (they override)
	for _, env := range userEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			// Env var without = means unset it
			delete(envMap, parts[0])
		}
	}

	// Convert back to slice
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}

	return result
}

// collectBindMountStates extracts states from bind mounts that need inputs
func collectBindMountStates(mounts []*Mount) []*dag.State {
	var states []*dag.State
	for _, m := range mounts {
		if m.state != nil {
			states = append(states, m.state)
		}
	}
	return states
}

// assignMountIndices assigns correct input indices to bind mounts
func assignMountIndices(mounts []*Mount, startIndex int) {
	// Start after the rootfs mount (which is at index 0)
	currentIndex := startIndex
	for _, m := range mounts {
		if m.state != nil {
			m.mount.Input = int64(currentIndex)
			currentIndex++
		}
	}
}

func NewExecState(state *dag.State, op *pb.ExecOp, luaFile string, luaLine int) *dag.State {
	return NewExecStateWithOpts(state, op, nil, luaFile, luaLine)
}

// NewExecStateWithOpts creates an exec state with additional options including bind mount states
func NewExecStateWithOpts(state *dag.State, op *pb.ExecOp, opts *ExecOptions, luaFile string, luaLine int) *dag.State {
	// Add rootfs mount as first mount if not already present
	hasRootfs := false
	for _, m := range op.Mounts {
		if m.Dest == "/" {
			hasRootfs = true
			break
		}
	}
	if !hasRootfs {
		// Create rootfs mount that will be connected to input index 0
		rootfsMount := &pb.Mount{
			Input:     0,
			Output:    0,
			Dest:      "/",
			MountType: pb.MountType_BIND,
			Readonly:  false,
		}
		// Prepend to ensure it's at input 0
		op.Mounts = append([]*pb.Mount{rootfsMount}, op.Mounts...)
	}

	// Collect bind mount states from opts if available
	var bindMounts []*Mount
	var bindMountStates []*dag.State
	if opts != nil && len(opts.Mounts) > 0 {
		bindMounts = opts.Mounts
		bindMountStates = collectBindMountStates(bindMounts)
	}

	// Assign input indices to bind mounts (starting from 1, after rootfs)
	assignMountIndices(bindMounts, 1)

	// Build inputs array: rootfs at index 0, bind mounts at subsequent indices
	inputs := []*pb.Input{
		{
			Digest: string(state.Op().Digest()),
			Index:  int64(state.OutputIndex()),
		},
	}
	for _, bindState := range bindMountStates {
		inputs = append(inputs, &pb.Input{
			Digest: string(bindState.Op().Digest()),
			Index:  int64(bindState.OutputIndex()),
		})
	}

	pbOp := &pb.Op{
		Inputs: inputs,
		Op: &pb.Op_Exec{
			Exec: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	node.AddInput(dag.NewEdge(state.Op(), state.OutputIndex()))

	// Add edges for bind mount states
	for _, bindState := range bindMountStates {
		node.AddInput(dag.NewEdge(bindState.Op(), bindState.OutputIndex()))
	}

	return dag.NewState(node)
}

func Run(state *dag.State, cmd []string, opts *ExecOptions, luaFile string, luaLine int) *dag.State {
	if len(cmd) == 0 {
		return nil
	}

	// If input state's OpNode has image config, inherit environment variables and working directory
	imageConfig := state.Op().ImageConfig()
	if imageConfig != nil && imageConfig.Config != nil {
		// Create opts if nil
		if opts == nil {
			opts = &ExecOptions{}
		}

		// Merge image config env with user env (user env overrides)
		mergedEnv := mergeEnv(imageConfig.Config.Config.Env, opts.Env)
		opts.Env = mergedEnv

		// Inherit working directory from image config if not explicitly set
		if opts.Cwd == "" {
			if imageConfig.Config.Config.WorkingDir != "" {
				opts.Cwd = imageConfig.Config.Config.WorkingDir
			} else {
				opts.Cwd = "/"
			}
		}
	}

	op := NewExecOp(cmd, opts)
	return NewExecStateWithOpts(state, op, opts, luaFile, luaLine)
}

func WithMetadata(state *dag.State, metadata *pb.OpMetadata) *dag.State {
	if metadata == nil {
		return nil
	}

	state.Op().SetMetadata(metadata)
	return state
}

func parseNetworkMode(mode string) pb.NetMode {
	switch mode {
	case "host":
		return pb.NetMode_HOST
	case "none":
		return pb.NetMode_NONE
	case "sandbox", "":
		return pb.NetMode_UNSET
	default:
		return pb.NetMode_UNSET
	}
}

func parseSecurityMode(mode string) pb.SecurityMode {
	switch mode {
	case "insecure":
		return pb.SecurityMode_INSECURE
	case "sandbox", "":
		return pb.SecurityMode_SANDBOX
	default:
		return pb.SecurityMode_SANDBOX
	}
}
