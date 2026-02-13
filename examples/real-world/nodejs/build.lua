-- Node.js Web Application - Luakit Port
-- Multi-stage build with production optimizations

local builder = bk.image("node:20-alpine")

local pkg_files = bk.local_("context", { include = { "package*.json" } })

local deps = builder:run({ "sh", "-c", "npm ci --only=production && cp -r /app/node_modules /node_modules" }, {
    cwd = "/app",
    mounts = {
        bk.bind(pkg_files, "/app"),
    },
})

local full_context = bk.local_("context")

local built = deps:run("npm run build && cp -r /app/dist /dist", {
    cwd = "/app",
    mounts = {
        bk.bind(full_context, "/app"),
        bk.cache("/root/.npm", { sharing = "locked" }),
    },
})

local runtime = bk.image("node:20-alpine")

local runtime_deps = runtime:copy(deps, "/node_modules", "/app/node_modules")

local runtime_dist = runtime_deps:copy(built, "/dist", "/app/dist")

local runtime_pkg = runtime_dist:copy(full_context, "package.json", "/app/package.json")

local with_user = runtime_pkg:run(
    "addgroup -g 1001 -S nodejs && " ..
    "adduser -S nodejs -u 1001 && " ..
    "chown -R nodejs:nodejs /app"
)

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
