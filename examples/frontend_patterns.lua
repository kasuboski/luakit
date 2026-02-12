-- Example: Frontend build with pattern filtering
-- Demonstrates using patterns to include only necessary files

-- Builder stage with Node.js
local builder = bk.image("node:20-alpine")

-- Local context with pattern filtering
local src = bk.local_("context", {
    include = {
        "package.json",
        "package-lock.json",
        "src/**/*.ts",
        "src/**/*.tsx",
        "public/**/*",
    },
    exclude = {
        "**/*.test.ts",
        "**/*.test.tsx",
        "**/*.spec.ts",
        "**/*.spec.tsx",
        "src/**/*.stories.tsx",
        ".git/",
        "coverage/",
        ".next/",
    },
    shared_key_hint = "frontend-sources",
})

-- Copy package files first for better caching
local deps = builder:mkdir("/app")
                 :copy(src, "package.json", "/app/package.json")
                 :copy(src, "package-lock.json", "/app/package-lock.json")

-- Install dependencies
local installed = deps:run("npm ci --only=production", {
    cwd = "/app",
    mounts = {
        bk.cache("/root/.npm", { sharing = "shared" }),
    },
})

-- Copy source code
local workspace = installed:copy(src, ".", "/app", {
    include = {
        "src/**/*",
        "public/**/*",
    },
})

-- Build application
local built = workspace:run("npm run build", {
    cwd = "/app",
    mounts = {
        bk.cache("/root/.npm", { sharing = "shared" }),
        bk.cache("/app/.next", { sharing = "locked", id = "next-cache" }),
    },
})

-- Runtime stage with nginx
local runtime = bk.image("nginx:alpine")

-- Copy only built artifacts
local final = runtime:copy(built, "/app/.next", "/usr/share/nginx/html", {
    include = {
        "static/**/*",
        "*.html",
    },
    exclude = {
        "static/**/*.map",
    },
})

-- Copy nginx config
local config = bk.local_("context", {
    include = {
        "nginx.conf",
    },
})

local final = final:copy(config, "nginx.conf", "/etc/nginx/nginx.conf")

bk.export(final, {
    expose = {"80/tcp"},
    cmd = {"nginx", "-g", "daemon off;"},
})
