package luavm

import (
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

	"github.com/kasuboski/luakit/pkg/dag"
)

type EvalResult struct {
	State       *dag.State
	ImageConfig *dockerspec.DockerOCIImage
	SourceFiles map[string][]byte
}
