
0.
,docker-image://docker.io/library/alpine:3.19
W
I
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86"
2
/app
w
I
Gsha256:b5de91fbc298975ecbdc4feb595357eed172083ae3a6e2699928c66a7ffe32e6"*(*&
/app/config.jsonÑ{"key":"value"}
c
I
Gsha256:12d2b18a5094c8b20d4d6786b4cd3324c6f5a5e1fc0f5d9b1322e547f1dafc83":
/app/config.json
t
I
Gsha256:d06f7f969675f7f30e9650b21316306d3ea0dbf93ae33b6621bfd7b2e6b6df52"'%B#
/usr/bin/python3/usr/bin/python
K
I
Gsha256:4f2b7df658603b49265287f3dbc6c6dc2a39f4237355e139d822de28ce93198e˚
W
Gsha256:12d2b18a5094c8b20d4d6786b4cd3324c6f5a5e1fc0f5d9b1322e547f1dafc83



W
Gsha256:4f2b7df658603b49265287f3dbc6c6dc2a39f4237355e139d822de28ce93198e



W
Gsha256:b5de91fbc298975ecbdc4feb595357eed172083ae3a6e2699928c66a7ffe32e6



W
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86



W
Gsha256:d06f7f969675f7f30e9650b21316306d3ea0dbf93ae33b6621bfd7b2e6b6df52


ª
0test/integration/golden_scripts/a36_file_ops.luaÅ
local base = bk.image("alpine:3.19")
local s1 = base:mkdir("/app")
local s2 = s1:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local s3 = s2:rm("/app/config.json")
local s4 = s3:symlink("/usr/bin/python3", "/usr/bin/python")
bk.export(s4)
"Lua