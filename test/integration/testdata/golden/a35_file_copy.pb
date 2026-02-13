
0.
,docker-image://docker.io/library/golang:1.22
0.
,docker-image://docker.io/library/alpine:3.19
À
I
Gsha256:0b64ab77f7f430213b0cffcc4ff5d4de1f5ad85c714485d852dc8a1ce5eb1886
I
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86"(&""
/usr/local/bin//usr/local/bin/
K
I
Gsha256:c5f916c2906f4316dda261e8e8abf44508b61c803e5884d86b9e3c85ef5bf9a2é
W
Gsha256:0b64ab77f7f430213b0cffcc4ff5d4de1f5ad85c714485d852dc8a1ce5eb1886



W
Gsha256:bc08f217bb0d78e6387ff76155b3190f915a2dcb9ff101c8ddbcbe0b498f4a86



W
Gsha256:c5f916c2906f4316dda261e8e8abf44508b61c803e5884d86b9e3c85ef5bf9a2


Û
1test/integration/golden_scripts/a35_file_copy.lua 
local base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local result = base:copy(src, "/usr/local/bin/", "/usr/local/bin/")
bk.export(result)
"Lua