local prelude = {}

local M = {}

function M.from_alpine(version)
	return bk.image("alpine:" .. (version or "3.19"))
end

function M.from_ubuntu(version)
	return bk.image("ubuntu:" .. (version or "24.04"))
end

function M.from_debian(version)
	return bk.image("debian:" .. (version or "bookworm-slim"))
end

function M.from_fedora(version)
	return bk.image("fedora:" .. (version or "39"))
end

function M.go_base(version)
	local base = bk.image("golang:" .. (version or "1.22-alpine"))
	return base:run({ "apk", "add", "--no-cache", "git", "ca-certificates", "build-base" })
end

function M.go_build(builder, src, opts)
	opts = opts or {}
	local cwd = opts.cwd or "/app"

	local mod_cache = builder:run({ "go", "mod", "download" }, {
		cwd = cwd,
		mounts = {
			bk.local_("context", { include_patterns = { "go.*" } }),
			bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
		},
	})

	local with_src = mod_cache:copy(src, ".", cwd)

	local build_flags = opts.flags or ""
	local ldflags = opts.ldflags or "-s -w"
	local output = opts.output or "/out/app"

	local built = with_src:run({
		"sh", "-c",
		"CGO_ENABLED=0 GOOS=linux go build -ldflags='" .. ldflags .. "' " .. build_flags .. " -o " .. output .. " " .. (opts.main or ".")
	}, {
		cwd = cwd,
		mounts = {
			bk.cache("/go/pkg/mod", { sharing = "shared", id = "gomod" }),
			bk.cache("/root/.cache/go-build", { sharing = "shared", id = "gobuild" }),
		},
	})

	return built
end

function M.go_runtime(version)
	local runtime = bk.image("alpine:" .. (version or "3.19"))
	return runtime:run({ "apk", "--no-cache", "add", "ca-certificates", "tzdata" })
end

function M.node_base(version)
	return bk.image("node:" .. (version or "20-alpine"))
end

function M.node_build(builder, src, opts)
	opts = opts or {}
	local cwd = opts.cwd or "/app"

	local deps_only = opts.deps_only or false
	local npm_cmd = deps_only and "npm ci --only=production" or "npm ci"

	local deps = builder:run({ "sh", "-c", npm_cmd }, {
		cwd = cwd,
		mounts = {
			bk.local_("context", { include_patterns = { "package*.json" } }),
			bk.cache("/root/.npm", { sharing = "locked" }),
		},
	})

	local with_src = deps:copy(src, ".", cwd)

	if opts.install_cmd then
		return with_src:run({ "sh", "-c", opts.install_cmd }, { cwd = cwd })
	end

	return with_src
end

function M.node_runtime(version)
	return bk.image("node:" .. (version or "20-alpine"))
end

function M.python_base(version, variant)
	local img = "python:" .. (version or "3.11")
	if variant then
		img = img .. "-" .. variant
	else
		img = img .. "-slim"
	end
	local base = bk.image(img)

	return base:run({
		"sh", "-c",
		"export DEBIAN_FRONTEND=noninteractive && " ..
		"apt-get update && apt-get install -y --no-install-recommends " ..
		"build-essential gcc g++ git curl wget && " ..
		"rm -rf /var/lib/apt/lists/*"
	}, {
		env = {
			DEBIAN_FRONTEND = "noninteractive",
			PYTHONUNBUFFERED = "1",
		},
	})
end

function M.python_build(builder, src, opts)
	opts = opts or {}
	local cwd = opts.cwd or "/workspace"

	local with_pip = builder:run({
		"sh", "-c",
		"pip install --no-cache-dir --upgrade pip"
	}, {
		mounts = {
			bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" }),
		},
	})

	if opts.requirements then
		local req_path = type(opts.requirements) == "string" and opts.requirements or "requirements.txt"
		local with_req = with_pip:copy(src, req_path, cwd .. "/" .. req_path)
		with_pip = with_req:run({
			"pip", "install", "--no-cache-dir", "-r", req_path
		}, {
			cwd = cwd,
			mounts = {
				bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" }),
			},
		})
	end

	local with_code = with_pip:copy(src, ".", cwd)

	if opts.install_cmd then
		with_code = with_code:run({ "sh", "-c", opts.install_cmd }, { cwd = cwd })
	end

	return with_code
end

function M.python_runtime(version)
	local base = bk.image("python:" .. (version or "3.11-slim"))
	return base:run({
		"sh", "-c",
		"apt-get update && " ..
		"apt-get install -y --no-install-recommends ca-certificates && " ..
		"rm -rf /var/lib/apt/lists/*"
	})
end

function M.container(base, build_fn)
	return build_fn(base)
end

function M.multi_stage(builder_image, runtime_image, build_fn)
	local builder = bk.image(builder_image)
	local built = build_fn(builder)
	local runtime = bk.image(runtime_image)
	return runtime, built
end

function M.copy_all(from_state, to_state, from_path, to_path)
	return to_state:copy(from_state, from_path, to_path)
end

function M.with_workdir(state, path)
	return state:mkdir(path, { make_parents = true })
end

function M.with_user(state, username, uid, gid)
	uid = uid or 1000
	gid = gid or uid
	return state:run({
		"sh", "-c",
		"addgroup -g " .. gid .. " " .. username .. " && " ..
		"adduser -D -u " .. uid .. " -G " .. username .. " " .. username
	})
end

function M.with_alpine_user(state, username, uid, gid)
	uid = uid or 1000
	gid = gid or uid
	return state:run({
		"sh", "-c",
		"addgroup -g " .. gid .. " -S " .. username .. " && " ..
		"adduser -S -u " .. uid .. " -G " .. username .. " " .. username
	})
end

function M.chown_path(state, path, user, group)
	return state:run({
		"sh", "-c",
		"chown -R " .. user .. ":" .. (group or user) .. " " .. path
	})
end

function M.deb_package(base, packages)
	local pkg_list = type(packages) == "table" and table.concat(packages, " ") or packages
	return base:run({
		"sh", "-c",
		"export DEBIAN_FRONTEND=noninteractive && " ..
		"apt-get update && apt-get install -y --no-install-recommends " .. pkg_list .. " && " ..
		"rm -rf /var/lib/apt/lists/*"
	}, {
		env = { DEBIAN_FRONTEND = "noninteractive" },
	})
end

function M.apk_package(base, packages)
	local pkg_list = type(packages) == "table" and table.concat(packages, " ") or packages
	return base:run({ "apk", "add", "--no-cache", pkg_list })
end

function M.install_git(base)
	return base:run({ "apk", "add", "--no-cache", "git" })
end

function M.install_curl(base)
	return base:run({ "apk", "add", "--no-cache", "curl" })
end

function M.install_ca_certs(base)
	return base:run({ "apk", "add", "--no-cache", "ca-certificates" })
end

function M.standard_base(distro, version)
	if distro == "alpine" then
		return M.from_alpine(version)
	elseif distro == "ubuntu" then
		return M.from_ubuntu(version)
	elseif distro == "debian" then
		return M.from_debian(version)
	elseif distro == "fedora" then
		return M.from_fedora(version)
	else
		error("Unknown distro: " .. tostring(distro))
	end
end

function M.go_binary_app(builder_image, src, opts)
	opts = opts or {}

	local builder = M.go_base(builder_image or "1.22-alpine")
	local built = M.go_build(builder, src, opts)

	local runtime = M.go_runtime(opts.runtime_version or "3.19")

	local binary_path = opts.output or "/out/app"
	local final = runtime:copy(built, binary_path, opts.final_path or "/app/app", {
		mode = "0755",
	})

	if opts.user then
		final = M.with_alpine_user(final, opts.user, opts.uid, opts.gid)
		final = M.chown_path(final, "/app", opts.user, opts.gid)
	end

	return final
end

function M.node_app(builder_image, src, opts)
	opts = opts or {}

	local builder = M.node_base(builder_image or "20-alpine")
	local built = M.node_build(builder, src, opts)

	local runtime = M.node_runtime(opts.runtime_version or "20-alpine")

	local final = runtime:copy(built, "/app/node_modules", "/app/node_modules")
	final = final:copy(built, "/app/dist", "/app/dist")
	final = final:copy(built, "/app/package.json", "/app/package.json")

	if opts.user then
		final = M.with_alpine_user(final, opts.user, opts.uid, opts.gid)
		final = M.chown_path(final, "/app", opts.user, opts.gid)
	end

	return final
end

function M.python_app(builder_image, src, opts)
	opts = opts or {}

	local builder = M.python_base(builder_image or "3.11", "slim")
	local built = M.python_build(builder, src, opts)

	local runtime = M.python_runtime(opts.runtime_version or "3.11-slim")

	local final = runtime:copy(built, "/workspace", "/app")

	if opts.user then
		final = M.with_user(final, opts.user or "appuser", opts.uid, opts.gid)
		final = M.chown_path(final, "/app", opts.user or "appuser")
	end

	return final
end

function M.parallel_build(...)
	local states = { ... }
	return bk.merge(unpack(states))
end

function M.layered_copy(target, sources, mappings)
	local result = target
	for _, mapping in ipairs(mappings) do
		result = result:copy(mapping.from, mapping.from_path, mapping.to_path)
	end
	return result
end

function M.merge_multiple(states)
	return bk.merge(unpack(states))
end

function M.deb_package_state(base, packages)
	local pkg_list = type(packages) == "table" and table.concat(packages, " ") or packages
	return base:run({
		"sh", "-c",
		"export DEBIAN_FRONTEND=noninteractive && " ..
		"apt-get update && apt-get install -y --no-install-recommends " .. pkg_list .. " && " ..
		"rm -rf /var/lib/apt/lists/*"
	}, {
		env = { DEBIAN_FRONTEND = "noninteractive" },
	})
end

function M.as_non_root(state, username, uid)
	username = username or "appuser"
	uid = uid or 1000
	local gid = uid

	local with_user = state:run({
		"sh", "-c",
		"addgroup -g " .. gid .. " " .. username .. " && " ..
		"adduser -D -u " .. uid .. " -G " .. username .. " " .. username
	})

	return with_user:run({
		"sh", "-c",
		"chown -R " .. username .. ":" .. username .. " /app"
	})
end

function M.install_system_deps(base, packages, distro)
	distro = distro or "alpine"

	if distro == "alpine" then
		return M.apk_package(base, packages)
	elseif distro == "debian" or distro == "ubuntu" then
		return M.deb_package(base, packages)
	else
		error("Unsupported distro: " .. tostring(distro))
	end
end

return M
