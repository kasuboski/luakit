-- Test environment variable inheritance
-- This should use the resolve_digest option to load image config

local golang = bk.image("golang:1.21-alpine", {
    resolve_digest = true,
})

-- This should now have access to /go/bin in PATH
local result = golang:run({ "go", "version" })

bk.export(result)
