
1/
-docker-image://docker.io/library/ubuntu:24.04
K
I
Gsha256:cb7eb42125e9a19b9feaa097f497c75d1f11358059efd2b7490a03012b731f38è
W
Gsha256:cb7eb42125e9a19b9feaa097f497c75d1f11358059efd2b7490a03012b731f38


Œ
0test/integration/golden_scripts/a40_platform.luaS
local arm = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
bk.export(arm)
"Lua