package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/moby/buildkit/client/llb/sourceresolver"
	gwclient "github.com/moby/buildkit/frontend/gateway/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type GatewayResolver struct {
	client gwclient.Client
}

func NewGatewayResolver(client gwclient.Client) *GatewayResolver {
	return &GatewayResolver{client: client}
}

func (r *GatewayResolver) Resolve(ctx context.Context, ref string, platform ocispec.Platform) (*ImageConfig, error) {
	ref = strings.TrimPrefix(ref, "docker-image://")
	ref = strings.TrimPrefix(ref, "oci-layout://")

	opt := sourceresolver.Opt{
		ImageOpt: &sourceresolver.ResolveImageOpt{
			Platform: &ocispec.Platform{
				OS:           platform.OS,
				Architecture: platform.Architecture,
				Variant:      platform.Variant,
			},
		},
	}

	resolvedRef, dgst, configBytes, err := r.client.ResolveImageConfig(ctx, ref, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve image config for %s: %w", ref, err)
	}

	var img ocispec.Image
	if err := json.Unmarshal(configBytes, &img); err != nil {
		return nil, fmt.Errorf("failed to unmarshal image config: %w", err)
	}

	return &ImageConfig{
		Ref:      resolvedRef,
		Digest:   dgst.String(),
		Config:   &img,
		Platform: platform,
	}, nil
}

var _ Interface = (*GatewayResolver)(nil)
