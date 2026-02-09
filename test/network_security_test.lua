local base = bk.image("alpine:3.19")

local isolated = base:run("echo 'No network access'", {
	network = "none",
	security = "sandbox"
})

local privileged = base:run("echo 'With host network and privileged mode'", {
	network = "host",
	security = "insecure"
})

local default = base:run("echo 'Default sandboxed with network'", {
	network = "sandbox",
	security = "sandbox"
})

local with_hostname = base:run("echo 'With custom hostname'", {
	network = "sandbox",
	security = "sandbox",
	hostname = "builder"
})

local with_exit_codes = base:run("echo 'With valid exit codes'", {
	network = "sandbox",
	security = "sandbox",
	valid_exit_codes = {0, 1}
})

local combined = bk.merge(isolated, privileged, default, with_hostname, with_exit_codes)
bk.export(combined)
