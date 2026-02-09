package ops

import (
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

func NewDiffOp(lower *pb.LowerDiffInput, upper *pb.UpperDiffInput) *pb.DiffOp {
	return &pb.DiffOp{
		Lower: lower,
		Upper: upper,
	}
}

func NewDiffState(lowerState, upperState *dag.State, luaFile string, luaLine int) *dag.State {
	inputs := []*pb.Input{
		{
			Digest: string(lowerState.Op().Digest()),
			Index:  int64(lowerState.OutputIndex()),
		},
		{
			Digest: string(upperState.Op().Digest()),
			Index:  int64(upperState.OutputIndex()),
		},
	}

	op := NewDiffOp(&pb.LowerDiffInput{Input: 0}, &pb.UpperDiffInput{Input: 1})
	pbOp := &pb.Op{
		Inputs: inputs,
		Op: &pb.Op_Diff{
			Diff: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	node.AddInput(dag.NewEdge(lowerState.Op(), lowerState.OutputIndex()))
	node.AddInput(dag.NewEdge(upperState.Op(), upperState.OutputIndex()))

	return dag.NewState(node)
}

func Diff(lowerState, upperState *dag.State, luaFile string, luaLine int) *dag.State {
	if lowerState == nil || upperState == nil {
		return nil
	}

	return NewDiffState(lowerState, upperState, luaFile, luaLine)
}
