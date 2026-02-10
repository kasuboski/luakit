-- Go Microservice - Luakit Port
-- Multi-stage build with optimized binary and minimal runtime

local builder = bk.image("golang:1.21-alpine")

local builder_deps = builder:run({
    "apk", "add", "--no-cache", "git", "ca-certificates"
})

local context_files = bk.local_("context", { include_patterns = { "go.*" } })

local mod_cache = builder_deps:run({ "go", "mod", "download" }, {
    cwd = "/app",
    mounts = {
        bk.bind(context_files, "/app"),
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
    },
})

local full_context = bk.local_("context")

local built = mod_cache:run({
    "sh", "-c",
    "CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -a -installsuffix cgo -o main ."
}, {
    cwd = "/app",
    mounts = {
        bk.bind(full_context, "/app"),
        bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
        bk.cache("/root/.cache/go-build", { sharing = "shared", id = "gobuild" }),
    },
})

local runtime = bk.image("alpine:3.19")

local with_certs = runtime:run({
    "apk", "--no-cache", "add", "ca-certificates", "tzdata"
})

local with_binary = with_certs:copy(built, "/app/main", "/root/main")

local with_tzdata = with_binary:copy(built, "/usr/local/go/lib/time/zoneinfo.zip", "/usr/local/zoneinfo.zip")

local with_user = with_tzdata:run({
    "sh", "-c",
    "addgroup -g 1000 app && " ..
    "adduser -D -u 1000 -G app app && " ..
    "chown -R app:app /root"
})

bk.export(with_user, {
    env = {
        TZ = "UTC",
        GIN_MODE = "release",
    },
    user = "app",
    workdir = "/root",
    expose = {"8080/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Go Microservice",
        ["org.opencontainers.image.description"] = "Multi-stage Go microservice with optimized binary",
        ["org.opencontainers.image.version"] = "1.0.0",
    },
})
