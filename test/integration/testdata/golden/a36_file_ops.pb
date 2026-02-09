

docker-image://alpine:3.19
W
I
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3"
2
/app
w
I
Gsha256:5c0c7a919a8728a667fdc07af72fb63281c7deea0c32a2d3bef8f3b0785dcf1f"*(*&
/app/config.json„{"key":"value"}
c
I
Gsha256:21e6777dbbcb91d460227edc4413b090fa085de5e3db2b0ea65137d22ca6f420":
/app/config.json
t
I
Gsha256:68bd45784f9f35f59f100416b9f29400b97a1a25c1dec087de4e4edd0217b231"'%B#
/usr/bin/python3/usr/bin/python–
W
Gsha256:973cca93fae8355a719e5faa02dde2c36c63467f8847f55905126125c52634c8



W
Gsha256:68bd45784f9f35f59f100416b9f29400b97a1a25c1dec087de4e4edd0217b231



W
Gsha256:21e6777dbbcb91d460227edc4413b090fa085de5e3db2b0ea65137d22ca6f420



W
Gsha256:5c0c7a919a8728a667fdc07af72fb63281c7deea0c32a2d3bef8f3b0785dcf1f



W
Gsha256:7551aa0f1fe889c2d2e03152c0e784283fd22f1abcc941425616bb8437aebdd3


Ö
L/Users/josh/projects/luakit/test/integration/golden_scripts/a36_file_ops.lua€local base = bk.image("alpine:3.19")
local s1 = base:mkdir("/app")
local s2 = s1:mkfile("/app/config.json", '{"key":"value"}', { mode = 0644 })
local s3 = s2:rm("/app/config.json")
local s4 = s3:symlink("/usr/bin/python3", "/usr/bin/python")
bk.export(s4)
"Lua