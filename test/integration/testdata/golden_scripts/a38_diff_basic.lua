local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache nginx")
local delta = bk.diff(base, installed)
bk.export(delta)
