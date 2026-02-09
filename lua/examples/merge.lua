local base = bk.image("alpine:3.19")

local deps = base:run("apk add --no-cache git")
local source = base:run("mkdir -p /app/src")
local config = base:run("mkdir -p /app/config")

local merged = bk.merge(deps, source, config)
bk.export(merged)
