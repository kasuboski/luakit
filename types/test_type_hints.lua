---@meta
---This is a test file to demonstrate type hints working in Lua editors
---Your editor should provide autocomplete and type information for the bk global

local base = bk.image("alpine:3.19")
local result = base:run("echo 'hello'")
local cache_mount = bk.cache("/cache", { id = "mycache", sharing = "shared" })
local secret_mount = bk.secret("/secret", { id = "mysecret" })

local merged = bk.merge(base, result)
local exported = bk.export(result, {
    entrypoint = { "/bin/sh" },
    env = { PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" }
})

---State methods should be autocompleted
local with_dir = result:mkdir("/app", { make_parents = true })
local with_file = result:mkfile("/app/test.txt", "content", { mode = "0644" })
local copied = result:copy(base, "/src", "/dest", { create_dest_path = true })
