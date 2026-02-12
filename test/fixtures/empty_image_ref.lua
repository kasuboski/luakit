local base = bk.image("")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
