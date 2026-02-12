-- Example demonstrating platform specification for cross-compilation and multi-platform builds

-- Cross-compile an application for ARM64
local arm64_base = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
local arm64_built = arm64_base:run("echo 'Building for ARM64' > /arch.txt")
bk.export(arm64_built)

-- Note: In a real multi-platform build, you would export multiple platforms
-- This example shows the syntax for platform specification
