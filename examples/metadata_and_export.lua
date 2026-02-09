-- Example demonstrating state:with_metadata() and bk.export() with full configuration

local base = bk.image("alpine:3.19")

-- Build with custom metadata for progress display
local built = base:run("apk add --no-cache nodejs npm", {
    cwd = "/workspace"
}):with_metadata({
    description = "Installing Node.js runtime",
    progress_group = "dependencies"
})

local app = built:run("npm install", {
    cwd = "/workspace/app"
}):with_metadata({
    description = "Installing npm dependencies",
    progress_group = "dependencies"
})

local compiled = app:run("npm run build", {
    cwd = "/workspace/app"
}):with_metadata({
    description = "Building application bundle",
    progress_group = "build"
})

-- Export with complete image configuration
bk.export(compiled, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "node /workspace/app/dist/index.js"},
    env = {
        NODE_ENV = "production",
        PORT = "8080",
    },
    workdir = "/workspace/app",
    user = "node",
    expose = {"8080/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Node.js Application",
        ["org.opencontainers.image.description"] = "Production-ready Node.js app built with luakit",
        ["org.opencontainers.image.version"] = "1.0.0",
    },
})
