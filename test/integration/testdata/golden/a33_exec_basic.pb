
0.
,docker-image://docker.io/library/alpine:3.19
m
I
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86 

/bin/sh
-c

echo hello/
K
I
Gsha256:e7b49ba7a9623be811edbe278bc6682878cfc386aaa1b29bce4e646dab16f39aÎ
W
Gsha256:e7b49ba7a9623be811edbe278bc6682878cfc386aaa1b29bce4e646dab16f39a

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