package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
)

type DOTWriter struct {
	outputPath string
	filterOp   string
}

func NewDOTWriter(outputPath string) *DOTWriter {
	return &DOTWriter{
		outputPath: outputPath,
		filterOp:   "",
	}
}

func (w *DOTWriter) SetFilter(opType string) {
	w.filterOp = opType
}

func (w *DOTWriter) Write(state *dag.State) error {
	var builder strings.Builder
	visited := make(map[string]bool)

	builder.WriteString("digraph dag {\n")
	builder.WriteString("  rankdir=TB;\n")
	builder.WriteString("  node [shape=box];\n")
	builder.WriteString("\n")

	w.writeNode(state, visited, &builder)

	builder.WriteString("}\n")

	output := builder.String()

	if w.outputPath == "" || w.outputPath == "-" {
		_, err := os.Stdout.WriteString(output)
		return err
	}

	return os.WriteFile(w.outputPath, []byte(output), 0644)
}

func (w *DOTWriter) writeNode(state *dag.State, visited map[string]bool, builder *strings.Builder) {
	node := state.Op()
	digest := node.Digest().String()

	if visited[digest] {
		return
	}
	visited[digest] = true

	for _, edge := range node.Inputs() {
		w.writeNode(dag.NewState(edge.Node()), visited, builder)
	}

	opType := getOpType(node.Op())

	if w.filterOp != "" && opType != w.filterOp {
		return
	}

	digestLabel := digest
	if len(digest) > 12 {
		digestLabel = digest[:12]
	}
	label := fmt.Sprintf("%s\\n%s", opType, digestLabel)

	if node.LuaFile() != "" {
		label += fmt.Sprintf("\\n%s:%d", node.LuaFile(), node.LuaLine())
	}

	switch opType := node.Op().Op.(type) {
	case *pb.Op_Exec:
		if opType.Exec != nil && len(opType.Exec.Meta.Args) > 0 {
			cmd := opType.Exec.Meta.Args[0]
			if len(opType.Exec.Meta.Args) > 1 {
				cmd += " ..."
			}
			label += fmt.Sprintf("\\ncmd: %s", cmd)
		}
	case *pb.Op_Source:
		if opType.Source != nil && opType.Source.Identifier != "" {
			identifier := opType.Source.Identifier
			if len(identifier) > 40 {
				identifier = identifier[:37] + "..."
			}
			label += fmt.Sprintf("\\n%s", identifier)
		}
	case *pb.Op_File:
		if opType.File != nil {
			label += fmt.Sprintf("\\nactions: %d", len(opType.File.Actions))
		}
	}

	builder.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", digest, label))

	for _, edge := range node.Inputs() {
		inputDigest := edge.Node().Digest().String()
		inputOpType := getOpType(edge.Node().Op())
		if w.filterOp == "" || inputOpType == w.filterOp {
			builder.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", inputDigest, digest))
		}
	}
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
