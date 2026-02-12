local M = {}

function M.go_build(base, src, opts)
	local deps = base:run("go mod download", {
		cwd = opts.cwd or "/app",
		mounts = { bk.cache("/go/pkg/mod") },
	})
	local with_src = deps:copy(src, ".", opts.cwd or "/app")
	return with_src:run("go build -o /out/app " .. (opts.main or "."), {
		cwd = opts.cwd or "/app",
		mounts = { bk.cache("/root/.cache/go-build") },
	})
end

function M.node_build(base, src, opts)
	local deps = base:run("npm ci", {
		cwd = opts.cwd or "/app",
		mounts = { bk.cache("/root/.npm") },
	})
	local with_src = deps:copy(src, ".", opts.cwd or "/app")
	return with_src:run("npm run build", {
		cwd = opts.cwd or "/app",
	})
end

return M
