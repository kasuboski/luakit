

local://context
0.
,docker-image://docker.io/library/golang:1.22
ß
I
Gsha256:a60212791641cbeaa3a49de4f7dff9e40ae50ec19d1be9607232037c1db16702
I
Gsha256:0b64ab77f7f430213b0cffcc4ff5d4de1f5ad85c714485d852dc8a1ce5eb1886""	
./app
ç
I
Gsha256:969d1dafdda30e7513a0adf895c574feeaf3cc1589767897d7073ed3770fcd89@
9
/bin/sh
-c
$go build -o /out/server ./cmd/server/app/
42
0docker-image://gcr.io/distroless/static-debian12
¥
I
Gsha256:2c3471ee057c115b6f78af2fc5ea1df43785b66496a3ebac19ca759b4a17859d
I
Gsha256:2a18a3190225aa23ec3739250b6f81d9aa8304693e4942d365775c56ae2c625a""
/out/server/server
K
I
Gsha256:cec2ad7e18e016633aa5b4512efccaafc2a822f4f018588171d34b2f736c38dc÷
Gsha256:cec2ad7e18e016633aa5b4512efccaafc2a822f4f018588171d34b2f736c38dcäá
containerimage.confign{"architecture":"amd64","os":"linux","rootfs":{"type":"","diff_ids":null},"config":{"Entrypoint":["/server"]}}–
W
Gsha256:2c3471ee057c115b6f78af2fc5ea1df43785b66496a3ebac19ca759b4a17859d



W
Gsha256:969d1dafdda30e7513a0adf895c574feeaf3cc1589767897d7073ed3770fcd89



W
Gsha256:0b64ab77f7f430213b0cffcc4ff5d4de1f5ad85c714485d852dc8a1ce5eb1886



W
Gsha256:a60212791641cbeaa3a49de4f7dff9e40ae50ec19d1be9607232037c1db16702



W
Gsha256:cec2ad7e18e016633aa5b4512efccaafc2a822f4f018588171d34b2f736c38dc



W
Gsha256:2a18a3190225aa23ec3739250b6f81d9aa8304693e4942d365775c56ae2c625a


∑
3test/integration/golden_scripts/a39_multi_stage.lua˙
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server ./cmd/server", { cwd = "/app" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
"Lua