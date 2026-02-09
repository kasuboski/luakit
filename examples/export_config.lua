-- Example demonstrating bk.export() with image configuration

local base = bk.image("alpine:3.19")

local result = base:run("echo 'Hello, World!' > /hello.txt")

bk.export(result, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "cat /hello.txt"},
    env = {
        PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
        APP_ENV = "production",
    },
    user = "root",
    workdir = "/",
    labels = {
        ["org.opencontainers.image.title"] = "Hello World App",
        ["org.opencontainers.image.description"] = "A simple example demonstrating luakit export options",
        ["org.opencontainers.image.version"] = "1.0.0",
        ["org.opencontainers.image.authors"] = "luakit team",
    },
    expose = {"8080/tcp"},
    os = "linux",
    arch = "amd64",
})
