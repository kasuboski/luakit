package output

import (
	"encoding/json"
	"fmt"

	"github.com/kasuboski/luakit/pkg/dag"
	pb "github.com/moby/buildkit/solver/pb"
)

type JSONWriter struct {
	outputPath string
	filterOp   string
}

type DAGNode struct {
	Digest  string       `json:"digest"`
	Type    string       `json:"type"`
	File    string       `json:"file,omitempty"`
	Line    int          `json:"line,omitempty"`
	Inputs  []string     `json:"inputs"`
	Details *NodeDetails `json:"details,omitempty"`
}

type NodeDetails struct {
	Identifier string            `json:"identifier,omitempty"`
	Command    []string          `json:"command,omitempty"`
	Env        []string          `json:"env,omitempty"`
	Cwd        string            `json:"cwd,omitempty"`
	User       string            `json:"user,omitempty"`
	Attrs      map[string]string `json:"attrs,omitempty"`
}

func NewJSONWriter(outputPath string) *JSONWriter {
	return &JSONWriter{
		outputPath: outputPath,
		filterOp:   "",
	}
}

func (w *JSONWriter) SetFilter(opType string) {
	w.filterOp = opType
}

func (w *JSONWriter) Write(state *dag.State) error {
	nodes := w.collectNodes(state, make(map[string]bool))

	data, err := json.MarshalIndent(nodes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return writeOutput(data, w.outputPath)
}

func (w *JSONWriter) collectNodes(state *dag.State, visited map[string]bool) []*DAGNode {
	node := state.Op()
	digest := node.Digest().String()

	if visited[digest] {
		return []*DAGNode{}
	}
	visited[digest] = true

	var allNodes []*DAGNode
	for _, edge := range node.Inputs() {
		allNodes = append(allNodes, w.collectNodes(dag.NewState(edge.Node()), visited)...)
	}

	opType := getOpType(node.Op())

	if w.filterOp != "" && opType != w.filterOp {
		return allNodes
	}

	dagNode := &DAGNode{
		Digest:  digest,
		Type:    opType,
		File:    node.LuaFile(),
		Line:    node.LuaLine(),
		Inputs:  []string{},
		Details: w.extractDetails(node.Op()),
	}

	for _, edge := range node.Inputs() {
		inputOpType := getOpType(edge.Node().Op())
		if w.filterOp == "" || inputOpType == w.filterOp {
			dagNode.Inputs = append(dagNode.Inputs, edge.Node().Digest().String())
		}
	}

	allNodes = append(allNodes, dagNode)
	return allNodes
}

func (w *JSONWriter) extractDetails(op *pb.Op) *NodeDetails {
	if op == nil {
		return nil
	}

	details := &NodeDetails{}

	switch opType := op.Op.(type) {
	case *pb.Op_Source:
		details.Identifier = opType.Source.Identifier
		details.Attrs = opType.Source.Attrs
	case *pb.Op_Exec:
		if opType.Exec != nil {
			details.Command = opType.Exec.Meta.Args
			details.Env = opType.Exec.Meta.Env
			details.Cwd = opType.Exec.Meta.Cwd
			details.User = opType.Exec.Meta.User
		}
	case *pb.Op_File:
		if opType.File != nil && len(opType.File.Actions) > 0 {
			details.Attrs = make(map[string]string)
			details.Attrs["actions"] = fmt.Sprintf("%d", len(opType.File.Actions))
		}
	case *pb.Op_Merge:
		if opType.Merge != nil && len(opType.Merge.Inputs) > 0 {
			details.Attrs = make(map[string]string)
			details.Attrs["inputs"] = fmt.Sprintf("%d", len(opType.Merge.Inputs))
		}
	case *pb.Op_Diff:
		if opType.Diff != nil {
			details.Attrs = make(map[string]string)
			if opType.Diff.Lower != nil {
				details.Attrs["lower"] = fmt.Sprintf("%d", opType.Diff.Lower.Input)
			}
			if opType.Diff.Upper != nil {
				details.Attrs["upper"] = fmt.Sprintf("%d", opType.Diff.Upper.Input)
			}
		}
	case *pb.Op_Build:
		if opType.Build != nil {
			details.Attrs = make(map[string]string)
			details.Attrs["builder"] = fmt.Sprintf("%d", opType.Build.Builder)
			if len(opType.Build.Inputs) > 0 {
				details.Attrs["input_count"] = fmt.Sprintf("%d", len(opType.Build.Inputs))
			}
		}
	}

	if len(details.Identifier) == 0 && len(details.Command) == 0 && len(details.Attrs) == 0 {
		return nil
	}

	return details
}
