package resolver

import (
	"context"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type Interface interface {
	Resolve(ctx context.Context, ref string, platform ocispec.Platform) (*ImageConfig, error)
}

var _ Interface = (*Resolver)(nil)
