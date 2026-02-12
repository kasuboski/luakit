package dag_test

import (
	"fmt"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
)

// ExampleSerialize demonstrates how to serialize a DAG to a BuildKit Definition.
func ExampleSerialize() {
	baseOp := &pb.Op{
		Inputs: []*pb.Input{},
		Op: &pb.Op_Source{
			Source: &pb.SourceOp{
				Identifier: "docker-image://alpine:latest",
			},
		},
	}
	baseNode := dag.NewOpNode(baseOp, "build.lua", 10)

	runOp := &pb.Op{
		Inputs: []*pb.Input{{}},
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo hello > /hello.txt"},
				},
				Mounts: []*pb.Mount{
					{
						Input:  0,
						Dest:   "/",
						Output: 0,
					},
				},
			},
		},
	}
	runNode := dag.NewOpNode(runOp, "build.lua", 15)
	runNode.AddInput(dag.NewEdge(baseNode, 0))

	state := dag.NewState(runNode)

	def, err := dag.Serialize(state, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Serialized %d operations\n", len(def.Def))
	fmt.Printf("Metadata entries: %d\n", len(def.Metadata))

	// Output:
	// Serialized 3 operations
	// Metadata entries: 0
}
