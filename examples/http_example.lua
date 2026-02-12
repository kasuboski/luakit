-- Example: Using bk.http() and bk.https() to download files from the web

-- Download a file with checksum verification
local file = bk.http("https://example.com/archive.tar.gz", {
    checksum = "sha256:abc123def456789...",  -- Verify checksum after download
    filename = "archive.tar.gz",            -- Set output filename
    chmod = 0644,                           -- Set file permissions
})

-- Download with custom headers (e.g., for authenticated APIs)
local private_file = bk.https("https://api.example.com/release/v1.0.0.tar.gz", {
    headers = {
        Authorization = "Bearer your-token-here",
        ["User-Agent"] = "luakit/0.1.0",
    },
})

-- Download with basic authentication
local auth_file = bk.https("https://user:pass@example.com/private/file.tar.gz", {
    username = "user",
    password = "pass",
    checksum = "sha256:...",
})

-- Use downloaded file in a build
local base = bk.image("alpine:3.19")

-- Extract the archive
local extracted = base:copy(file, "archive.tar.gz", "/tmp/")
local result = extracted:run("cd /tmp && tar -xzf archive.tar.gz")

-- Copy the extracted files to the final location
local final = result:run("cp -r /tmp/app/* /app/")

bk.export(final, {
    entrypoint = {"/app/start"},
})

-- bk.http() / bk.https() options:
-- - checksum: Verify file integrity (format: "sha256:<hash>" or "sha512:<hash>")
-- - filename: Set the output filename (optional)
-- - chmod: Set file permissions in octal format (e.g., 0644)
-- - headers: Map of HTTP headers to include in the request
-- - username: Basic auth username (use with password)
-- - password: Basic auth password (use with username)

-- Note: Both bk.http() and bk.https() accept the same options.
-- They are provided as aliases for clarity in your build scripts.
