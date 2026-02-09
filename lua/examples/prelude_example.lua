local prelude = require("prelude")

local src = bk.local_("context")

local final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = "./cmd/server",
	output = "/out/server",
	user = "app",
	uid = 1000,
	gid = 1000,
})

bk.export(final, {
	entrypoint = {"/app/server"},
	user = "app",
	workdir = "/app",
	expose = {"8080/tcp"},
	env = {
		GIN_MODE = "release",
		PORT = "8080",
	},
	labels = {
		["org.opencontainers.image.title"] = "Go Microservice",
		["org.opencontainers.image.description"] = "Built with luakit prelude library",
	},
})
