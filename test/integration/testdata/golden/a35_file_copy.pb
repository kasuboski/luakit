

docker-image://alpine:3.19

docker-image://golang:1.22
¾
I
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3
I
Gsha256:70fadfe6f3ac8953c3430503e0645718d20da42735a58508ec4891c6809ec4e5"&$""
/usr/local/bin//usr/local/bin/„
W
Gsha256:2abe9eed26011bd041c372b150e1d327c1caad5a385a880110670a51a8124331



W
Gsha256:70fadfe6f3ac8953c3430503e0645718d20da42735a58508ec4891c6809ec4e5



W
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3


ö
M/Users/josh/projects/luakit/test/integration/golden_scripts/a35_file_copy.luaŸlocal base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local result = base:copy(src, "/usr/local/bin/", "/usr/local/bin/")
bk.export(result)
"Lua