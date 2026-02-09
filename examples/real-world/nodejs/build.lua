-- Node.js Web Application - Luakit Port
-- Multi-stage build with production optimizations

local builder = bk.image("node:20-alpine")

local deps = builder:run({ "npm", "ci", "--only=production" }, {
    cwd = "/app",
    mounts = {
        bk.local_("context", { include_patterns = { "package*.json" } }),
    },
})

local built = deps:run({ "npm", "run", "build" }, {
    cwd = "/app",
    mounts = {
        bk.local_("context"),
        bk.cache("/root/.npm", { sharing = "locked" }),
    },
})

local runtime = bk.image("node:20-alpine")

local runtime_deps = runtime:copy(built, "/app/node_modules", "/app/node_modules")

local runtime_dist = runtime_deps:copy(built, "/app/dist", "/app/dist")

local runtime_pkg = runtime_dist:copy(built, "/app/package.json", "/app/package.json")

local with_user = runtime_pkg:run({
    "sh", "-c",
    "addgroup -g 1001 -S nodejs && " ..
    "adduser -S nodejs -u 1001 && " ..
    "chown -R nodejs:nodejs /app"
})

bk.export(with_user, {
    env = {
        NODE_ENV = "production",
        PORT = "3000",
    },
    user = "nodejs",
    workdir = "/app",
    expose = {"3000/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Node.js Web Application",
        ["org.opencontainers.image.description"] = "Multi-stage Node.js application with production optimizations",
        ["org.opencontainers.image.version"] = "1.0.0",
    },
})
