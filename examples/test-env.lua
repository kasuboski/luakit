-- Test env var inheritance
local golang = bk.image("golang:1.21-alpine", {
    resolve_digest = true,
})

-- This should have access to /go/bin in PATH
local result = golang:run({ "go", "version" })

bk.export(result)
