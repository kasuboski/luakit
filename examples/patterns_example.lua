-- Example: Multi-stage Go build with pattern filtering
-- This demonstrates include/exclude patterns for copy operations

local builder = bk.image("golang:1.22-bookworm")
local src = bk.local_("context", {
    include = {
        "*.go",
        "go.mod",
        "go.sum",
    },
    exclude = {
        "*_test.go",
        "vendor/",
    },
    shared_key_hint = "go-sources",
})

local deps = builder:mkdir("/app")
                 :copy(src, "go.mod", "/app/go.mod")
                 :copy(src, "go.sum", "/app/go.sum")

local downloaded = deps:run("go mod download", { cwd = "/app" })

local workspace = downloaded:copy(src, ".", "/app")

local built = workspace:run(
    "CGO_ENABLED=0 go build -o /out/server ./cmd/server",
    {
        cwd = "/app",
        mounts = {
            bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
            bk.cache("/root/.cache/go-build", { sharing = "shared", id = "go-build" }),
        },
    }
)

local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server", {
    mode = "0755",
})

bk.export(final, {
    entrypoint = {"/server"},
    env = {
        "PATH=/",
    },
})
