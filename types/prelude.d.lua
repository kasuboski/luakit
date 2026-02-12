---@meta

---@class GoBuildOpts
---@field cwd? string
---@field flags? string
---@field ldflags? string
---@field output? string
---@field main? string

---@class GoAppOpts
---@field runtime_version? string
---@field output? string
---@field final_path? string
---@field user? string
---@field uid? integer
---@field gid? integer

---@class NodeBuildOpts
---@field cwd? string
---@field deps_only? boolean
---@field install_cmd? string

---@class NodeAppOpts
---@field runtime_version? string
---@field user? string
---@field uid? integer
---@field gid? integer

---@class PythonBuildOpts
---@field cwd? string
---@field requirements? string|string[]
---@field install_cmd? string

---@class PythonAppOpts
---@field runtime_version? string
---@field user? string
---@field uid? integer
---@field gid? integer

---@class LayeredMapping
---@field from State
---@field from_path string
---@field to_path string

---Utility functions for common build patterns
---@class Prelude
local Prelude = {}

---Create base image from Alpine
---@param version? string Version (default: "3.19")
---@return State state
function Prelude.from_alpine(version) end

---Create base image from Ubuntu
---@param version? string Version (default: "24.04")
---@return State state
function Prelude.from_ubuntu(version) end

---Create base image from Debian
---@param version? string Version (default: "bookworm-slim")
---@return State state
function Prelude.from_debian(version) end

---Create base image from Fedora
---@param version? string Version (default: "39")
---@return State state
function Prelude.from_fedora(version) end

---Create Go builder base
---@param version? string Version (default: "1.22-alpine")
---@return State state
function Prelude.go_base(version) end

---Run Go build
---@param builder State Builder state
---@param src State Source state
---@param opts? GoBuildOpts Build options
---@return State state
function Prelude.go_build(builder, src, opts) end

---Create Go runtime
---@param version? string Version (default: "3.19")
---@return State state
function Prelude.go_runtime(version) end

---Create Node.js base
---@param version? string Version (default: "20-alpine")
---@return State state
function Prelude.node_base(version) end

---Run Node.js build
---@param builder State Builder state
---@param src State Source state
---@param opts? NodeBuildOpts Build options
---@return State state
function Prelude.node_build(builder, src, opts) end

---Create Node.js runtime
---@param version? string Version (default: "20-alpine")
---@return State state
function Prelude.node_runtime(version) end

---Create Python base
---@param version? string Version (default: "3.11")
---@param variant? string Variant (default: "slim")
---@return State state
function Prelude.python_base(version, variant) end

---Run Python build
---@param builder State Builder state
---@param src State Source state
---@param opts? PythonBuildOpts Build options
---@return State state
function Prelude.python_build(builder, src, opts) end

---Create Python runtime
---@param version? string Version (default: "3.11-slim")
---@return State state
function Prelude.python_runtime(version) end

---Apply build function to base
---@param base State Base state
---@param build_fn fun(state: State):State Build function
---@return State state
function Prelude.container(base, build_fn) end

---Multi-stage build
---@param builder_image string Builder image
---@param runtime_image string Runtime image
---@param build_fn fun(state: State):State Build function
---@return State runtime, State built
function Prelude.multi_stage(builder_image, runtime_image, build_fn) end

---Copy all files from one state to another
---@param from_state State Source state
---@param to_state State Destination state
---@param from_path string Source path
---@param to_path string Destination path
---@return State state
function Prelude.copy_all(from_state, to_state, from_path, to_path) end

---Set working directory
---@param state State The state
---@param path string The working directory path
---@return State state
function Prelude.with_workdir(state, path) end

---Add user (Alpine)
---@param state State The state
---@param username string Username
---@param uid? integer User ID (default: 1000)
---@param gid? integer Group ID (default: same as uid)
---@return State state
function Prelude.with_alpine_user(state, username, uid, gid) end

---Add user (Debian/Ubuntu)
---@param state State The state
---@param username string Username
---@param uid? integer User ID (default: 1000)
---@param gid? integer Group ID (default: same as uid)
---@return State state
function Prelude.with_user(state, username, uid, gid) end

---Change ownership of path
---@param state State The state
---@param path string Path to chown
---@param user string Username
---@param group? string Group (default: same as user)
---@return State state
function Prelude.chown_path(state, path, user, group) end

---Install Debian packages
---@param base State Base state
---@param packages string|string[] Package name(s)
---@return State state
function Prelude.deb_package(base, packages) end

---Install Alpine packages
---@param base State Base state
---@param packages string|string[] Package name(s)
---@return State state
function Prelude.apk_package(base, packages) end

---Install git
---@param base State Base state
---@return State state
function Prelude.install_git(base) end

---Install curl
---@param base State Base state
---@return State state
function Prelude.install_curl(base) end

---Install CA certificates
---@param base State Base state
---@return State state
function Prelude.install_ca_certs(base) end

---Standard base image
---@param distro "alpine"|"ubuntu"|"debian"|"fedora" Distribution
---@param version? string Version
---@return State state
function Prelude.standard_base(distro, version) end

---Complete Go binary app
---@param builder_image? string Builder image
---@param src State Source state
---@param opts? GoAppOpts App options
---@return State state
function Prelude.go_binary_app(builder_image, src, opts) end

---Complete Node.js app
---@param builder_image? string Builder image
---@param src State Source state
---@param opts? NodeAppOpts App options
---@return State state
function Prelude.node_app(builder_image, src, opts) end

---Complete Python app
---@param builder_image? string Builder image
---@param src State Source state
---@param opts? PythonAppOpts App options
---@return State state
function Prelude.python_app(builder_image, src, opts) end

---Build states in parallel and merge
---@param ... State States to build in parallel
---@return State state
function Prelude.parallel_build(...) end

---Layered copy of multiple sources
---@param target State Target state
---@param sources table Source states
---@param mappings LayeredMapping[] File mappings
---@return State state
function Prelude.layered_copy(target, sources, mappings) end

---Merge multiple states
---@param states State[] States to merge
---@return State state
function Prelude.merge_multiple(states) end

---Install system dependencies
---@param base State Base state
---@param packages string|string[] Package name(s)
---@param distro? string Distribution (default: "alpine")
---@return State state
function Prelude.install_system_deps(base, packages, distro) end

---Run as non-root user
---@param state State The state
---@param username? string Username (default: "appuser")
---@param uid? integer User ID (default: 1000)
---@return State state
function Prelude.as_non_root(state, username, uid) end

return Prelude
