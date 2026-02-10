package output

import (
	"fmt"

	"github.com/moby/buildkit/solver/pb"
)

type ProtobufWriter struct {
	outputPath string
}

func NewProtobufWriter(outputPath string) *ProtobufWriter {
	return &ProtobufWriter{
		outputPath: outputPath,
	}
}

func (w *ProtobufWriter) Write(def *pb.Definition) error {
	data, err := def.MarshalVT()
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}

	return writeOutput(data, w.outputPath)
}
