local base = bk.image("alpine:3.19")

-- Build with no network access (secure, isolated)
local isolated_build = base:run("echo 'Building with no network access'", {
    network = "none",
    security = "sandbox"
})

-- Build with host network and insecure mode (for testing)
local privileged_build = base:run("echo 'Building with host network'", {
    network = "host",
    security = "insecure"
})

-- Default: sandboxed but with network access
local normal_build = base:run("echo 'Normal build with network'", {
    network = "sandbox",
    security = "sandbox"
})

-- Build with custom hostname and valid exit codes
local custom_build = base:run("echo 'Custom hostname and exit codes'", {
    hostname = "builder",
    valid_exit_codes = {0, 1}
})

-- Combine all builds
local result = bk.merge(isolated_build, privileged_build, normal_build, custom_build)
bk.export(result)
