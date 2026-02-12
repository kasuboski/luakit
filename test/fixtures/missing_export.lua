local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
