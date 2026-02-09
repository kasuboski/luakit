
42
0docker-image://gcr.io/distroless/static-debian12

docker-image://golang:1.22

local://context
¥
I
Gsha256:70fadfe6f3ac8953c3430503e0645718d20da42735a58508ec4891c6809ec4e5
I
Gsha256:a60212791641cbeaa3a49de4f7dff9e40ae50ec19d1be9607232037c1db16702""	
./app
ˆ
I
Gsha256:86eb697bb945f6627d5f60d527b1f862fb899f33b9b636a3a4b0a0582eaa1ec0;
9
/bin/sh
-c
$go build -o /out/server ./cmd/server/app
²
I
Gsha256:2a18a3190225aa23ec3739250b6f81d9aa8304693e4942d365775c56ae2c625a
I
Gsha256:a673d666e186ddcb591485efc325de84094bd7692556a3f4d6022b24dcbc0c08""
/out/server/serverÖ
Gsha256:5fa892c9a7b0fcd9fba955be37d927a609455b785434a773438eea94617dc05eŠ‡
containerimage.confign{"architecture":"amd64","os":"linux","rootfs":{"type":"","diff_ids":null},"config":{"Entrypoint":["/server"]}}ë
W
Gsha256:70fadfe6f3ac8953c3430503e0645718d20da42735a58508ec4891c6809ec4e5



W
Gsha256:2a18a3190225aa23ec3739250b6f81d9aa8304693e4942d365775c56ae2c625a



W
Gsha256:5fa892c9a7b0fcd9fba955be37d927a609455b785434a773438eea94617dc05e



W
Gsha256:a673d666e186ddcb591485efc325de84094bd7692556a3f4d6022b24dcbc0c08



W
Gsha256:86eb697bb945f6627d5f60d527b1f862fb899f33b9b636a3a4b0a0582eaa1ec0



W
Gsha256:a60212791641cbeaa3a49de4f7dff9e40ae50ec19d1be9607232037c1db16702


Ò
O/Users/josh/projects/luakit/test/integration/golden_scripts/a39_multi_stage.luaùlocal builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", { cwd = "/app" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
"Lua