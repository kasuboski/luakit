local base = bk.image("alpine:3.19")

local installed = base:run("apk add --no-cache git vim")
local just_git = bk.diff(base, installed)

local clean_base = bk.image("alpine:3.19")
local with_git = bk.merge(clean_base, just_git)
bk.export(with_git)
