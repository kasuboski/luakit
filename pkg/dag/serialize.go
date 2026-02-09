package dag

import (
	"encoding/json"

	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
)

type SerializeOptions struct {
	ImageConfig *dockerspec.DockerOCIImage
	SourceFiles map[string][]byte
}

// Serialize converts the DAG starting from the given state to a pb.Definition.
func Serialize(state *State, opts *SerializeOptions) (*pb.Definition, error) {
	visited := make(map[string]bool, 128)
	smb := NewSourceMapBuilder()

	def := &pb.Definition{
		Def:      make([][]byte, 0, 64),
		Metadata: make(map[string]*pb.OpMetadata, 64),
		Source:   &pb.Source{},
	}

	if opts != nil {
		for filename, data := range opts.SourceFiles {
			smb.AddFile(filename, data)
		}
	}

	if err := walk(state.Op(), visited, def, smb); err != nil {
		return nil, err
	}

	if opts != nil && opts.ImageConfig != nil {
		configBytes, err := json.Marshal(opts.ImageConfig)
		if err != nil {
			return nil, err
		}

		digest := state.Op().DigestString()

		if def.Metadata[digest] == nil {
			def.Metadata[digest] = &pb.OpMetadata{}
		}

		if def.Metadata[digest].Description == nil {
			def.Metadata[digest].Description = make(map[string]string, 1)
		}

		def.Metadata[digest].Description[exptypes.ExporterImageConfigKey] = string(configBytes)
	}

	def.Source = smb.Build()

	return def, nil
}

// walk recursively visits all OpNodes in the DAG and serializes them.
func walk(node *OpNode, visited map[string]bool, def *pb.Definition, smb *SourceMapBuilder) error {
	dig := node.DigestString()
	if visited[dig] {
		return nil
	}
	visited[dig] = true

	for _, edge := range node.Inputs() {
		if err := walk(edge.node, visited, def, smb); err != nil {
			return err
		}
	}

	populateInputDigests(node)

	dt, err := node.MarshalOp()
	if err != nil {
		return err
	}

	def.Def = append(def.Def, dt)
	meta := node.Metadata()
	if meta != nil && (len(meta.Description) > 0 || meta.ProgressGroup != nil) {
		def.Metadata[dig] = meta
	}

	luaFile := node.LuaFile()
	luaLine := node.LuaLine()
	if luaFile != "" && luaLine > 0 {
		smb.AddLocation(dig, luaFile, luaLine)
	}

	return nil
}

// populateInputDigests sets the digest field for each input in the Op.
func populateInputDigests(node *OpNode) {
	op := node.Op()
	if len(op.Inputs) != len(node.Inputs()) {
		return
	}

	for i, edge := range node.Inputs() {
		inputDigest := edge.node.DigestString()
		if op.Inputs[i] == nil {
			op.Inputs[i] = &pb.Input{}
		}
		op.Inputs[i].Digest = inputDigest
		op.Inputs[i].Index = int64(edge.outputIndex)
	}
}
