package dag

import (
	"sync"

	pb "github.com/moby/buildkit/solver/pb"
)

var (
	statePool  sync.Pool
	edgePool   sync.Pool
	opNodePool sync.Pool
)

func initStatePools() {
	statePool.New = func() interface{} {
		return &State{}
	}
	edgePool.New = func() interface{} {
		return &Edge{}
	}
	opNodePool.New = func() interface{} {
		return &OpNode{
			metadata: &pb.OpMetadata{},
			inputs:   make([]*Edge, 0, 4),
		}
	}
}

func init() {
	initStatePools()
}

// State represents a filesystem state at a point in the build graph.
// It is immutable — each operation returns a new State.
type State struct {
	op          *OpNode
	outputIndex int
	platform    *pb.Platform
}

// OpNode is a vertex in the DAG.
type OpNode struct {
	op       *pb.Op
	metadata *pb.OpMetadata
	inputs   []*Edge

	luaFile string
	luaLine int
	digest  string
}

// Edge represents a dependency from one OpNode to another.
type Edge struct {
	node        *OpNode
	outputIndex int
}

// NewEdge creates a new Edge from an OpNode.
func NewEdge(node *OpNode, outputIndex int) *Edge {
	edge := edgePool.Get().(*Edge)
	edge.node = node
	edge.outputIndex = outputIndex
	return edge
}

// Node returns the OpNode that this edge points to.
func (e *Edge) Node() *OpNode {
	return e.node
}

// OutputIndex returns the output index of this edge.
func (e *Edge) OutputIndex() int {
	return e.outputIndex
}

// NewState creates a new State from an OpNode.
func NewState(op *OpNode) *State {
	state := statePool.Get().(*State)
	state.op = op
	state.outputIndex = 0
	state.platform = nil
	return state
}

// NewStateWithOutput creates a new State with a specific output index.
func NewStateWithOutput(op *OpNode, outputIndex int) *State {
	state := statePool.Get().(*State)
	state.op = op
	state.outputIndex = outputIndex
	state.platform = nil
	return state
}

// WithPlatform returns a new State with the platform set.
func (s *State) WithPlatform(platform *pb.Platform) *State {
	state := statePool.Get().(*State)
	state.op = s.op
	state.outputIndex = s.outputIndex
	state.platform = platform
	return state
}

// Op returns the OpNode that produces this state.
func (s *State) Op() *OpNode {
	return s.op
}

// OutputIndex returns which output of the Op this state represents.
func (s *State) OutputIndex() int {
	return s.outputIndex
}

// Platform returns the platform override for this state.
func (s *State) Platform() *pb.Platform {
	return s.platform
}

// NewOpNode creates a new OpNode.
func NewOpNode(op *pb.Op, luaFile string, luaLine int) *OpNode {
	node := opNodePool.Get().(*OpNode)
	node.op = op
	node.luaFile = luaFile
	node.luaLine = luaLine
	node.digest = ""
	if len(node.inputs) > 0 {
		node.inputs = node.inputs[:0]
	}
	return node
}

// AddInput adds an input edge to the OpNode.
func (n *OpNode) AddInput(edge *Edge) {
	n.inputs = append(n.inputs, edge)
}

// SetMetadata sets the metadata for the OpNode.
func (n *OpNode) SetMetadata(metadata *pb.OpMetadata) {
	n.metadata = metadata
}

// LuaFile returns the Lua file where this Op was created.
func (n *OpNode) LuaFile() string {
	return n.luaFile
}

// LuaLine returns the Lua line where this Op was created.
func (n *OpNode) LuaLine() int {
	return n.luaLine
}

// Inputs returns all input edges.
func (n *OpNode) Inputs() []*Edge {
	return n.inputs
}

// Op returns the pb.Op.
func (n *OpNode) Op() *pb.Op {
	return n.op
}

// Metadata returns the metadata.
func (n *OpNode) Metadata() *pb.OpMetadata {
	return n.metadata
}
