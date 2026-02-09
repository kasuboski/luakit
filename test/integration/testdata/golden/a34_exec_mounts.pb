

docker-image://alpine:3.19
Ø
I
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3Š

/bin/sh
-c
npm ci
/root/.npm0¢
	npm-cache#/run/secrets/npmrc0ª

npmrc €/run/ssh0²
default €/tmp0š€€€€È
W
Gsha256:3385bc520e8604b535102ba174616e36901cc378a22468a3208e76f4cc00ea73



W
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3


“
O/Users/josh/projects/luakit/test/integration/golden_scripts/a34_exec_mounts.luaºlocal base = bk.image("alpine:3.19")
local result = base:run("npm ci", {
    mounts = {
        bk.cache("/root/.npm", { id = "npm-cache" }),
        bk.secret("/run/secrets/npmrc", { id = "npmrc" }),
        bk.ssh({ id = "default" }),
        bk.tmpfs("/tmp", { size = 1073741824 }),
    },
})
bk.export(result)
"Lua