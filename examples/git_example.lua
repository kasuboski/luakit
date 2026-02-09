-- Example: Using bk.git() to clone a Git repository as a build source

-- Clone a specific version (tag)
local repo = bk.git("https://github.com/moby/buildkit.git", {
    ref = "v0.12.0"
})

-- Clone with .git directory preserved (useful for git operations during build)
-- local repo_with_git = bk.git("https://github.com/moby/buildkit.git", {
--     ref = "main",
--     keep_git_dir = true
-- })

-- Use the cloned repository in a build
local base = bk.image("golang:1.22")
local workspace = base:copy(repo, ".", "/src")

local built = workspace:run("go build -o /out/app ./cmd/buildctl", {
    cwd = "/src",
    mounts = {
        bk.cache("/go/pkg/mod"),
        bk.cache("/root/.cache/go-build"),
    },
})

local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/app", "/buildctl")

bk.export(final, {
    entrypoint = {"/buildctl"},
})

-- bk.git() options:
-- - ref: Branch, tag, or commit hash (optional)
-- - keep_git_dir: Preserve .git directory (default: false)
