package ops

import (
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

func NewMergeOp(inputs []*pb.MergeInput) *pb.MergeOp {
	return &pb.MergeOp{
		Inputs: inputs,
	}
}

func NewMergeState(states []*dag.State, luaFile string, luaLine int) *dag.State {
	inputs := []*pb.Input{}
	mergeInputs := []*pb.MergeInput{}

	for i, state := range states {
		inputs = append(inputs, &pb.Input{
			Digest: string(state.Op().Digest()),
			Index:  int64(state.OutputIndex()),
		})
		mergeInputs = append(mergeInputs, &pb.MergeInput{
			Input: int64(i),
		})
	}

	op := NewMergeOp(mergeInputs)
	pbOp := &pb.Op{
		Inputs: inputs,
		Op: &pb.Op_Merge{
			Merge: op,
		},
	}

	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	for _, state := range states {
		node.AddInput(dag.NewEdge(state.Op(), state.OutputIndex()))
	}

	return dag.NewState(node)
}

func Merge(states []*dag.State, luaFile string, luaLine int) *dag.State {
	if len(states) < 2 {
		return nil
	}

	return NewMergeState(states, luaFile, luaLine)
}
