local base = bk.image("alpine:3.19")

local updated = base:run("echo hello > /greeting.txt")

local with_env = updated:run("cat /greeting.txt", {
    env = { MESSAGE = "world" }
})

local in_app = with_env:run("pwd", { cwd = "/app" })

local as_user = in_app:run("whoami", { user = "nobody" })

bk.export(as_user)
