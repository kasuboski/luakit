package output

import (
	"os"

	pb "github.com/moby/buildkit/solver/pb"
)

const outputFileMode = 0644

func writeOutput(output []byte, outputPath string) error {
	if outputPath == "" || outputPath == "-" {
		_, err := os.Stdout.Write(output)
		return err
	}
	return os.WriteFile(outputPath, output, outputFileMode)
}

func getOpType(op *pb.Op) string {
	if op == nil {
		return "Unknown"
	}

	switch op.Op.(type) {
	case *pb.Op_Exec:
		return "Exec"
	case *pb.Op_Source:
		return "Source"
	case *pb.Op_File:
		return "File"
	case *pb.Op_Build:
		return "Build"
	case *pb.Op_Merge:
		return "Merge"
	case *pb.Op_Diff:
		return "Diff"
	default:
		return "Unknown"
	}
}
