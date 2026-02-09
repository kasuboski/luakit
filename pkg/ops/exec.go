package ops

import (
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

func NewExecState(state *dag.State, op *pb.ExecOp, luaFile string, luaLine int) *dag.State {
	pbOp := &pb.Op{
		Inputs: []*pb.Input{
			{
				Digest: string(state.Op().Digest()),
				Index:  int64(state.OutputIndex()),
			},
		},
		Op: &pb.Op_Exec{
			Exec: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	node.AddInput(dag.NewEdge(state.Op(), state.OutputIndex()))

	return dag.NewState(node)
}

func Run(state *dag.State, cmd []string, opts *ExecOptions, luaFile string, luaLine int) *dag.State {
	if len(cmd) == 0 {
		return nil
	}

	op := NewExecOp(cmd, opts)
	return NewExecState(state, op, luaFile, luaLine)
}

func WithMetadata(state *dag.State, metadata *pb.OpMetadata) *dag.State {
	if metadata == nil {
		return nil
	}

	state.Op().SetMetadata(metadata)
	return state
}
