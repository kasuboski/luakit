local base = bk.image("alpine:3.19")

-- Test network modes
local no_network = base:run("echo 'No network access'", {
    network = "none"
})

local host_network = base:run("echo 'Host network access'", {
    network = "host"
})

local sandbox_network = base:run("echo 'Sandbox network access'", {
    network = "sandbox"
})

-- Test security modes
local sandbox_security = base:run("echo 'Sandbox security'", {
    security = "sandbox"
})

local insecure_security = base:run("echo 'Insecure security'", {
    security = "insecure"
})

-- Test hostname option
local with_hostname = base:run("echo 'With hostname'", {
    hostname = "custom-builder"
})

-- Test valid_exit_codes option
local with_exit_codes = base:run("echo 'With valid exit codes'", {
    valid_exit_codes = {0, 1}
})

-- Test running as non-root
local as_nonroot = base:run("echo 'Running as non-root'", {
    user = "nobody"
})

-- Combine all options together
local all_options = base:run("echo 'All options combined'", {
    network = "none",
    security = "sandbox",
    user = "builder",
    cwd = "/app",
    hostname = "builder",
    valid_exit_codes = {0, 1, 2},
    env = {
        PATH = "/usr/local/bin:/usr/bin:/bin",
        CUSTOM_VAR = "value"
    }
})

-- Combine all test states
local result = bk.merge(
    no_network,
    host_network,
    sandbox_network,
    sandbox_security,
    insecure_security,
    with_hostname,
    with_exit_codes,
    as_nonroot,
    all_options
)

bk.export(result)
