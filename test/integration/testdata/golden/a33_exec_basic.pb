
0.
,docker-image://docker.io/library/alpine:3.19
³
I
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86f
_
/bin/sh
-c

echo helloAPATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin//
K
I
Gsha256:a1b796906160088132dcc8b7de42521d8c0d94aec2f5e7505708fec4b0091737Î
W
Gsha256:a1b796906160088132dcc8b7de42521d8c0d94aec2f5e7505708fec4b0091737



W
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86


™
2test/integration/golden_scripts/a33_exec_basic.lua^
local base = bk.image("alpine:3.19")
local result = base:run("echo hello")
bk.export(result)
"Lua