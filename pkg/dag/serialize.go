package dag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kasuboski/luakit/pkg/resolver"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type SerializeOptions struct {
	ImageConfig *dockerspec.DockerOCIImage
	SourceFiles map[string][]byte
	Resolver    *resolver.Resolver
}

// resolveImageConfigs walks the DAG and resolves image configs for SourceOps.
func resolveImageConfigs(ctx context.Context, state *State, reslv *resolver.Resolver) error {
	visited := make(map[string]bool)

	var walkAndResolve func(*OpNode) error
	walkAndResolve = func(node *OpNode) error {
		dig := node.DigestString()
		if visited[dig] {
			return nil
		}
		visited[dig] = true

		// Walk inputs first
		for _, edge := range node.Inputs() {
			if err := walkAndResolve(edge.Node()); err != nil {
				return err
			}
		}

		// Check if this is a SourceOp that needs resolution
		if node.ResolveConfig() && node.Op().GetSource() != nil {
			source := node.Op().GetSource()
			identifier := source.Identifier

			// Get or default platform
			platform := node.Platform()
			if platform == nil {
				defaultSpec := resolver.DefaultPlatform()
				platform = &pb.Platform{
					OS:           defaultSpec.OS,
					Architecture: defaultSpec.Architecture,
					Variant:      defaultSpec.Variant,
				}
			}

			ocispecPlatform := ocispec.Platform{
				OS:           platform.OS,
				Architecture: platform.Architecture,
				Variant:      platform.Variant,
			}

			// Resolve image config
			imgConfig, err := reslv.Resolve(ctx, identifier, ocispecPlatform)
			if err != nil {
				return fmt.Errorf("failed to resolve image config for %s: %w", identifier, err)
			}

			// Store image config in the OpNode
			node.SetImageConfig(&ImageConfig{
				Config: imgConfig.Config,
			})
		}

		return nil
	}

	return walkAndResolve(state.Op())
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

	// Resolve image configs if resolver is provided
	if opts != nil && opts.Resolver != nil {
		ctx := context.Background()
		if err := resolveImageConfigs(ctx, state, opts.Resolver); err != nil {
			return nil, err
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

	// Add final output vertex with no operation type (op.Op == nil)
	// This vertex just references the actual final state via its input
	// This is required for provenance/sbom generation
	finalInput := &pb.Input{
		Digest: string(state.Op().Digest()),
		Index:  int64(state.OutputIndex()),
	}
	finalOp := &pb.Op{
		Inputs: []*pb.Input{finalInput},
		// No operation type set (no Exec, Source, File, etc.)
		// This makes op.Op == nil, which is required for provenance
	}
	finalOpBytes, err := finalOp.MarshalVT()
	if err != nil {
		return nil, err
	}
	def.Def = append(def.Def, finalOpBytes)

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
