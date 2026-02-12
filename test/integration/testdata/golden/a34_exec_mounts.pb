
0.
,docker-image://docker.io/library/alpine:3.19
İ
I
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86

/bin/sh
-c
npm ci/
/root/.npm0¢
	npm-cache#/run/secrets/npmrc0ª

npmrc €/run/ssh0²
default €/tmp0š€€€€
K
I
Gsha256:ac51c21606be8e7322fa2c0724921dd653b692de2bb1509dec447108eee1ff0d­
W
Gsha256:ac51c21606be8e7322fa2c0724921dd653b692de2bb1509dec447108eee1ff0d



W
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86


ø
3test/integration/golden_scripts/a34_exec_mounts.lua»
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
"Lua