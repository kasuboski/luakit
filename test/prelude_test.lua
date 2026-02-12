local prelude = require("prelude")

local tests_passed = 0
local tests_failed = 0

local function assert(condition, message)
	if condition then
		tests_passed = tests_passed + 1
		print("✓ " .. (message or "assertion passed"))
	else
		tests_failed = tests_failed + 1
		print("✗ " .. (message or "assertion failed"))
		error("Assertion failed", 0)
	end
end

local function assert_type(value, expected_type, name)
	local actual_type = type(value)
	assert(actual_type == expected_type, name .. " should be " .. expected_type .. ", got " .. actual_type)
end

local function assert_error(fn, message)
	local ok, err = pcall(fn)
	assert(not ok, message or "expected error")
end

print("Testing prelude.base functions...")

local alpine = prelude.from_alpine()
assert_type(alpine, "userdata", "from_alpine()")

local alpine_318 = prelude.from_alpine("3.18")
assert_type(alpine_318, "userdata", "from_alpine('3.18')")

local ubuntu = prelude.from_ubuntu()
assert_type(ubuntu, "userdata", "from_ubuntu()")

local ubuntu_2204 = prelude.from_ubuntu("22.04")
assert_type(ubuntu_2204, "userdata", "from_ubuntu('22.04')")

local debian = prelude.from_debian()
assert_type(debian, "userdata", "from_debian()")

local fedora = prelude.from_fedora()
assert_type(fedora, "userdata", "from_fedora()")

print("Testing Go builders...")

local go_base = prelude.go_base()
assert_type(go_base, "userdata", "go_base()")

local go_base_custom = prelude.go_base("1.21-alpine")
assert_type(go_base_custom, "userdata", "go_base('1.21-alpine')")

local go_runtime = prelude.go_runtime()
assert_type(go_runtime, "userdata", "go_runtime()")

local src = bk.local_("context")

local go_built = prelude.go_build(go_base, src, {
	cwd = "/app",
	main = "./cmd/main",
	output = "/out/main",
})
assert_type(go_built, "userdata", "go_build()")

local go_final = prelude.go_binary_app("1.22-alpine", src, {
	cwd = "/app",
	main = ".",
	user = "app",
})
assert_type(go_final, "userdata", "go_binary_app()")

print("Testing Node.js builders...")

local node_base = prelude.node_base()
assert_type(node_base, "userdata", "node_base()")

local node_base_custom = prelude.node_base("18-alpine")
assert_type(node_base_custom, "userdata", "node_base('18-alpine')")

local node_runtime = prelude.node_runtime()
assert_type(node_runtime, "userdata", "node_runtime()")

local node_built = prelude.node_build(node_base, src, {
	cwd = "/app",
})
assert_type(node_built, "userdata", "node_build()")

local node_final = prelude.node_app("20-alpine", src, {
	cwd = "/app",
	user = "nodejs",
})
assert_type(node_final, "userdata", "node_app()")

print("Testing Python builders...")

local python_base = prelude.python_base()
assert_type(python_base, "userdata", "python_base()")

local python_base_custom = prelude.python_base("3.10", "slim")
assert_type(python_base_custom, "userdata", "python_base('3.10', 'slim')")

local python_runtime = prelude.python_runtime()
assert_type(python_runtime, "userdata", "python_runtime()")

local python_built = prelude.python_build(python_base, src, {
	cwd = "/workspace",
})
assert_type(python_built, "userdata", "python_build()")

local python_final = prelude.python_app("3.11", src, {
	cwd = "/workspace",
	user = "appuser",
})
assert_type(python_final, "userdata", "python_app()")

print("Testing container helpers...")

local base = bk.image("alpine:3.19")
local result = prelude.container(base, function(s)
	return s:run("echo test")
end)
assert_type(result, "userdata", "container()")

local runtime, built = prelude.multi_stage("golang:1.22-alpine", "alpine:3.19", function(builder)
	return builder:run("echo building")
end)
assert_type(runtime, "userdata", "multi_stage() runtime")
assert_type(built, "userdata", "multi_stage() built")

print("Testing copy helpers...")

local target = bk.scratch()
local source = base:run("echo 'hello' > /file.txt")
local copied = prelude.copy_all(source, target, "/file.txt", "/file.txt")
assert_type(copied, "copied_all()")

print("Testing directory helpers...")

local with_workdir = prelude.with_workdir(base, "/app")
assert_type(with_workdir, "userdata", "with_workdir()")

local with_user = prelude.with_user(debian, "testuser", 1000, 1000)
assert_type(with_user, "userdata", "with_user()")

local with_alpine_user = prelude.with_alpine_user(alpine, "testuser", 1000, 1000)
assert_type(with_alpine_user, "userdata", "with_alpine_user()")

local with_chown = prelude.chown_path(alpine, "/app", "appuser", "appuser")
assert_type(with_chown, "userdata", "chown_path()")

print("Testing package installers...")

local deb_pkgs = prelude.deb_package(debian, { "git", "curl", "wget" })
assert_type(deb_pkgs, "userdata", "deb_package() with table")

local deb_pkgs_str = prelude.deb_package(debian, "git curl")
assert_type(deb_pkgs_str, "userdata", "deb_package() with string")

local apk_pkgs = prelude.apk_package(alpine, { "git", "curl" })
assert_type(apk_pkgs, "userdata", "apk_package() with table")

local apk_pkgs_str = prelude.apk_package(alpine, "git curl")
assert_type(apk_pkgs_str, "userdata", "apk_package() with string")

print("Testing convenience functions...")

local with_git = prelude.install_git(alpine)
assert_type(with_git, "userdata", "install_git()")

local with_curl = prelude.install_curl(alpine)
assert_type(with_curl, "userdata", "install_curl()")

local with_ca = prelude.install_ca_certs(alpine)
assert_type(with_ca, "userdata", "install_ca_certs()")

print("Testing standard_base...")

local std_alpine = prelude.standard_base("alpine")
assert_type(std_alpine, "userdata", "standard_base('alpine')")

local std_ubuntu = prelude.standard_base("ubuntu")
assert_type(std_ubuntu, "userdata", "standard_base('ubuntu')")

local std_debian = prelude.standard_base("debian")
assert_type(std_debian, "userdata", "standard_base('debian')")

local std_fedora = prelude.standard_base("fedora")
assert_type(std_fedora, "userdata", "standard_base('fedora')")

local ok, err = pcall(function()
	prelude.standard_base("unknown")
end)
assert(not ok, "standard_base('unknown') should error")

print("Testing parallel_build...")

local state1 = base:run("echo '1' > /1.txt")
local state2 = base:run("echo '2' > /2.txt")
local state3 = base:run("echo '3' > /3.txt")
local merged = prelude.parallel_build(state1, state2, state3)
assert_type(merged, "userdata", "parallel_build()")

print("Testing merge_multiple...")

local merged_multi = prelude.merge_multiple({ state1, state2, state3 })
assert_type(merged_multi, "userdata", "merge_multiple()")

print("Testing layered_copy...")

local target2 = bk.scratch()
local src1 = base:run("echo '1' > /a.txt")
local src2 = base:run("echo '2' > /b.txt")
local layered = prelude.layered_copy(target2, {}, {
	{ from = src1, from_path = "/a.txt", to_path = "/a.txt" },
	{ from = src2, from_path = "/b.txt", to_path = "/b.txt" },
})
assert_type(layered, "userdata", "layered_copy()")

print("Testing as_non_root...")

local app = base:run("mkdir -p /app && echo 'app' > /app/file.txt")
local non_root = prelude.as_non_root(app, "myapp", 2000)
assert_type(non_root, "userdata", "as_non_root()")

print("Testing install_system_deps...")

local sys_deps_alpine = prelude.install_system_deps(alpine, { "git", "curl" }, "alpine")
assert_type(sys_deps_alpine, "userdata", "install_system_deps() alpine")

local sys_deps_debian = prelude.install_system_deps(debian, { "git", "curl" }, "debian")
assert_type(sys_deps_debian, "userdata", "install_system_deps() debian")

print("Testing deb_package_state...")

local deb_state = prelude.deb_package_state(debian, { "vim", "nano" })
assert_type(deb_state, "userdata", "deb_package_state()")

print("\n=== Test Summary ===")
print("Passed: " .. tests_passed)
print("Failed: " .. tests_failed)
print("Total:  " .. (tests_passed + tests_failed))

if tests_failed > 0 then
	error("Some tests failed")
end
