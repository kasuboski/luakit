-- Example demonstrating require() functionality in luakit

-- Import the prelude module from stdlib
local prelude = require("prelude")

-- Create a base image
local base = bk.image("alpine:3.19")

-- Run a simple command
local result = base:run("echo 'Hello from require!' > /greeting.txt")

-- Export the result
bk.export(result)
