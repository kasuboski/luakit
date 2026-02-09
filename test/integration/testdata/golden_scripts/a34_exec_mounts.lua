local base = bk.image("alpine:3.19")
local result = base:run("npm ci", {
    mounts = {
        bk.cache("/root/.npm", { id = "npm-cache" }),
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.ssh({ id = "default" }),
        bk.tmpfs("/tmp", { size = 1073741824 }),
    },
})
bk.export(result)
