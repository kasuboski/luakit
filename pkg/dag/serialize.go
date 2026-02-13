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
	Resolver    resolver.Interface
}

// resolveImageConfigs walks the DAG and resolves image configs for SourceOps.
func resolveImageConfigs(ctx context.Context, state *State, reslv resolver.Interface) error {
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

	// Always propagate ImageConfig through the DAG and apply to ExecOps.
	// This ensures ExecOps get WorkingDir/Env from base images,
	// or default cwd to "/" if no config available.
	propagateImageConfigs(state)

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

	node.InvalidateDigest()
	dig = node.DigestString()

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

// propagateImageConfigs walks the DAG and propagates image configs from SourceOps to ExecOps.
// For each ExecOp, it finds the image config from the root mount input and applies
// WorkingDir and Env to the ExecOp's Meta. User-specified values take precedence.
func propagateImageConfigs(state *State) {
	visited := make(map[string]bool)
	propagateWalk(state.Op(), visited)
}

// propagateWalk recursively walks the DAG and propagates image configs to ExecOps.
func propagateWalk(node *OpNode, visited map[string]bool) {
	dig := node.DigestString()
	if visited[dig] {
		return
	}
	visited[dig] = true

	for _, edge := range node.Inputs() {
		propagateWalk(edge.Node(), visited)
	}

	exec := node.Op().GetExec()
	if exec == nil {
		return
	}

	config := findImageConfigForExec(node)
	if applyImageConfigToExec(exec, config) {
		node.InvalidateDigest()
		visited[node.DigestString()] = true
	}
}

// findImageConfigForExec finds the image config from the root mount of an ExecOp.
func findImageConfigForExec(node *OpNode) *ImageConfig {
	exec := node.Op().GetExec()
	if exec == nil {
		return nil
	}

	for _, mount := range exec.Mounts {
		if mount.Dest == "/" && mount.Input >= 0 && int(mount.Input) < len(node.Inputs()) {
			input := node.Inputs()[mount.Input]
			return findImageConfigFromNode(input.Node())
		}
	}

	if len(node.Inputs()) > 0 {
		return findImageConfigFromNode(node.Inputs()[0].Node())
	}

	return nil
}

// findImageConfigFromNode recursively finds an image config from a node.
func findImageConfigFromNode(node *OpNode) *ImageConfig {
	if config := node.ImageConfig(); config != nil {
		return config
	}

	if len(node.Inputs()) > 0 {
		return findImageConfigFromNode(node.Inputs()[0].Node())
	}

	return nil
}

// applyImageConfigToExec applies image config to an ExecOp's Meta.
// User-specified values (Cwd, Env) take precedence over image config values.
// Returns true if changes were made.
func applyImageConfigToExec(exec *pb.ExecOp, config *ImageConfig) bool {
	if exec.Meta == nil {
		exec.Meta = &pb.Meta{}
	}

	changed := false

	if exec.Meta.Cwd == "" {
		if config != nil && config.Config != nil && config.Config.Config.WorkingDir != "" {
			exec.Meta.Cwd = config.Config.Config.WorkingDir
			changed = true
		} else {
			exec.Meta.Cwd = "/"
			changed = true
		}
	}

	if config != nil && config.Config != nil && len(config.Config.Config.Env) > 0 {
		exec.Meta.Env = mergeEnv(config.Config.Config.Env, exec.Meta.Env)
		changed = true
	}

	return changed
}

// mergeEnv merges environment variables, with user env taking precedence.
// The result preserves the order of keys: imageEnv keys first, then new userEnv keys.
func mergeEnv(imageEnv, userEnv []string) []string {
	envMap := make(map[string]string)
	var keyOrder []string
	seen := make(map[string]bool)

	for _, e := range imageEnv {
		if key, val := splitEnv(e); key != "" {
			if !seen[key] {
				keyOrder = append(keyOrder, key)
				seen[key] = true
			}
			envMap[key] = val
		}
	}

	for _, e := range userEnv {
		if key, val := splitEnv(e); key != "" {
			if !seen[key] {
				keyOrder = append(keyOrder, key)
				seen[key] = true
			}
			envMap[key] = val
		}
	}

	result := make([]string, 0, len(keyOrder))
	for _, k := range keyOrder {
		result = append(result, k+"="+envMap[k])
	}

	return result
}

// splitEnv splits an environment variable into key and value.
func splitEnv(env string) (string, string) {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return env[:i], env[i+1:]
		}
	}
	return env, ""
}
