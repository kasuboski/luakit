local base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local result = base:copy(src, "/usr/local/bin/", "/usr/local/bin/")
bk.export(result)
