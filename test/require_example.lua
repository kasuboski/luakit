local helpers = require("test.modules.build")

local base = bk.image("golang:1.22")
local src = bk.local_("context")

local built = helpers.go_build(base, src, {
	main = "./cmd/server",
	cwd = "/app",
})

local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/app", "/server")

bk.export(final, {
	entrypoint = {"/server"},
})
