

docker-image://alpine:3.19
h
I
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3

/bin/sh
-c

echo helloé
W
Gsha256:e7019c60bc75cc086f7418729668d1fa4461f973d5397ae34a4817ad03539122



W
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3


´
N/Users/josh/projects/luakit/test/integration/golden_scripts/a33_exec_basic.lua]local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
"Lua