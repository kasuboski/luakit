local base = bk.image("alpine:3.19")

local simple = base:run("echo hello > /test.txt")

local with_array = simple:run({ "cat", "/test.txt" })

local with_env = with_array:run("echo $PATH", {
    env = { PATH = "/usr/bin:/bin" }
})

local with_cwd = with_env:run("pwd", { cwd = "/tmp" })

local with_user = with_cwd:run("whoami", { user = "nobody" })

bk.export(with_user)
