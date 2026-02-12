local base = bk.image("ubuntu:24.04")
local installed = base:run("apt-get update && apt-get install -y nginx")
local just_nginx = bk.diff(base, installed)

local alpine = bk.image("alpine:3.19")
local final = bk.merge(alpine, just_nginx)
bk.export(final)
