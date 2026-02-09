-- Example demonstrating owner and permission support in luakit file operations
-- This shows how to control file ownership and permissions in the generated image

local base = bk.image("alpine:3.19")

-- Create a directory with specific permissions (0755) and ownership
local app_dir = base:mkdir("/app", {
    mode = "0755",          -- rwxr-xr-x
    make_parents = true,
    owner = { user = "appuser", group = "appgroup" }
})

-- Create a configuration file with restricted permissions (0600)
local with_config = app_dir:mkfile("/app/config.json", '{"key":"value"}', {
    mode = 493,              -- 0755 octal = 493 decimal (use string mode "0755" for octal)
    owner = { user = "root", group = "root" }
})

-- Create an executable script with execute permissions (0755)
local with_script = with_config:mkfile("/app/start.sh", "#!/bin/sh\necho 'Starting app'\n", {
    mode = "0755",          -- rwxr-xr-x (executable)
    owner = { user = "appuser", group = "appgroup" }
})

-- Example: Copy files from another state with ownership and mode changes
local source = bk.image("ubuntu:24.04")
local with_copy = with_script:copy(source, "/etc/hosts", "/app/hosts", {
    mode = "0644",          -- rw-r--r--
    owner = { user = 1000, group = 1000 }  -- Can use numeric IDs
})

-- Example: Mixed ownership types (user by name, group by ID)
local with_mixed = with_copy:mkdir("/app/data", {
    mode = "0700",          -- rwx------
    owner = { user = "appuser", group = 1000 }
})

bk.export(with_mixed, {
    user = "appuser",
    workdir = "/app"
})
