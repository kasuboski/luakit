package dag_test

import (
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"

	"github.com/kasuboski/luakit/pkg/dag"
	"github.com/kasuboski/luakit/pkg/ops"
)

func BenchmarkDAGConstructionSimple(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		result := ops.Run(base, []string{"/bin/sh", "-c", "echo hello"}, nil, "test.lua", 2)
		_ = result
	}
}

func BenchmarkDAGConstruction50Ops(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		state := base
		for j := range 50 {
			state = ops.Run(state, []string{"/bin/sh", "-c", "echo test"}, nil, "test.lua", j+2)
		}
		_ = state
	}
}

func BenchmarkDAGConstruction100Ops(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		state := base
		for j := range 100 {
			state = ops.Run(state, []string{"/bin/sh", "-c", "echo test"}, nil, "test.lua", j+2)
		}
		_ = state
	}
}

func BenchmarkDAGConstructionWithFileOps(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		s1 := ops.Mkdir(base, "/app", nil, "test.lua", 2)
		s2 := ops.Mkfile(s1, "/app/file.txt", "content", nil, "test.lua", 3)
		s3 := ops.Symlink(s2, "/app/file.txt", "/app/link", "test.lua", 4)
		_ = s3
	}
}

func BenchmarkDAGConstructionMultiStage(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder := ops.Image("golang:1.22", "test.lua", 1, nil, nil)
		src := ops.Local("context", "test.lua", 2, nil)
		workspace := ops.Copy(builder, src, ".", "/app", nil, "test.lua", 3)
		built := ops.Run(workspace, []string{"/bin/sh", "-c", "go build -o /out/server ./cmd/server"}, nil, "test.lua", 4)
		runtime := ops.Image("gcr.io/distroless/static-debian12", "test.lua", 5, nil, nil)
		final := ops.Copy(runtime, built, "/out/server", "/server", nil, "test.lua", 6)
		_ = final
	}
}

func BenchmarkDAGConstructionMerge(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		deps := ops.Run(base, []string{"/bin/sh", "-c", "apk add --no-cache git"}, nil, "test.lua", 2)
		source := ops.Run(base, []string{"/bin/sh", "-c", "mkdir -p /app/src"}, nil, "test.lua", 3)
		config := ops.Run(base, []string{"/bin/sh", "-c", "mkdir -p /app/config"}, nil, "test.lua", 4)
		merged := ops.Merge([]*dag.State{deps, source, config}, "test.lua", 5)
		_ = merged
	}
}

func BenchmarkDAGConstructionDiff(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
		installed := ops.Run(base, []string{"/bin/sh", "-c", "apk add --no-cache curl"}, nil, "test.lua", 2)
		diffed := ops.Diff(base, installed, "test.lua", 3)
		_ = diffed
	}
}

func BenchmarkSerializeSimpleDAG(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	result := ops.Run(base, []string{"/bin/sh", "-c", "echo hello"}, nil, "test.lua", 2)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(result, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerialize50OpsDAG(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	state := base
	for j := range 50 {
		state = ops.Run(state, []string{"/bin/sh", "-c", "echo test"}, nil, "test.lua", j+2)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerialize100OpsDAG(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	state := base
	for j := range 100 {
		state = ops.Run(state, []string{"/bin/sh", "-c", "echo test"}, nil, "test.lua", j+2)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(state, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerializeWithSourceMaps(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	result := ops.Run(base, []string{"/bin/sh", "-c", "echo hello"}, nil, "test.lua", 2)

	sourceFiles := map[string][]byte{
		"test.lua": []byte(`local base = bk.image("alpine:3.19")`),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(result, &dag.SerializeOptions{
			SourceFiles: sourceFiles,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerializeWithImageConfig(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	result := ops.Run(base, []string{"/bin/sh", "-c", "echo hello"}, nil, "test.lua", 2)

	imageConfig := &dockerspec.DockerOCIImage{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := dag.Serialize(result, &dag.SerializeOptions{
			ImageConfig: imageConfig,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDigestComputation(b *testing.B) {
	op := &pb.Op{
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo hello"},
				},
			},
		},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := op.MarshalVT()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWalkDAG(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)
	state := base
	for j := range 50 {
		state = ops.Run(state, []string{"/bin/sh", "-c", "echo test"}, nil, "test.lua", j+2)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		visited := make(map[string]bool)
		def := &pb.Definition{
			Def:      [][]byte{},
			Metadata: map[string]*pb.OpMetadata{},
			Source:   &pb.Source{},
		}
		smb := dag.NewSourceMapBuilder()
		walkDAG(state.Op(), visited, def, smb)
	}
}

func walkDAG(node *dag.OpNode, visited map[string]bool, def *pb.Definition, smb *dag.SourceMapBuilder) {
	dig := string(node.Digest())
	if visited[dig] {
		return
	}
	visited[dig] = true

	for _, edge := range node.Inputs() {
		walkDAG(edge.Node(), visited, def, smb)
	}

	op := node.Op()
	dt, _ := op.MarshalVT()
	def.Def = append(def.Def, dt)

	luaFile := node.LuaFile()
	luaLine := node.LuaLine()
	if luaFile != "" && luaLine > 0 {
		smb.AddLocation(dig, luaFile, luaLine)
	}
}

func BenchmarkEdgeCreation(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		edge := dag.NewEdge(base.Op(), 0)
		_ = edge
	}
}

func BenchmarkStateCreation(b *testing.B) {
	base := ops.Image("alpine:3.19", "test.lua", 1, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		state := dag.NewState(base.Op())
		_ = state
	}
}

func BenchmarkOpNodeCreation(b *testing.B) {
	op := &pb.Op{
		Op: &pb.Op_Exec{
			Exec: &pb.ExecOp{
				Meta: &pb.Meta{
					Args: []string{"/bin/sh", "-c", "echo test"},
				},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		node := dag.NewOpNode(op, "test.lua", 1)
		_ = node
	}
}
