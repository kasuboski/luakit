---@meta

---@class ImageOptions
---@field platform Platform|platform_string

---@class LocalOptions
---@field include string[]
---@field exclude string[]
---@field shared_key_hint string

---@class GitOptions
---@field ref string
---@field keep_git_dir boolean

---@class HTTPOptions
---@field checksum string
---@field filename string
---@field chmod integer
---@field headers table<string, string>
---@field username string
---@field password string

---@class CacheOptions
---@field id string
---@field sharing "shared"|"private"|"locked"

---@class SecretOptions
---@field id string
---@field uid integer
---@field gid integer
---@field mode integer
---@field optional boolean

---@class SSHOptions
---@field dest string
---@field id string
---@field uid integer
---@field gid integer
---@field mode integer
---@field optional boolean

---@class TmpfsOptions
---@field size integer

---@class BindOptions
---@field readonly boolean
---@field selector string

---@class ExportConfig
---@field entrypoint string[]
---@field cmd string[]
---@field env table<string, string>
---@field user string
---@field workdir string
---@field expose string[]
---@field labels table<string, string>
---@field os string
---@field arch string
---@field variant string

---@alias platform_string string

---@class Platform
---@field os string
---@field arch string
---@field variant string

---@class BK
local BK = {}

bk = BK

---@param ref string Image reference
---@param opts? ImageOptions Optional image options
---@return State state
function BK.image(ref, opts) end

---@return State state
function BK.scratch() end

---@param name string Local source name
---@param opts? LocalOptions Optional local options
---@return State state
function BK.local_(name, opts) end

---@param url string Git repository URL
---@param opts? GitOptions Optional git options
---@return State state
function BK.git(url, opts) end

---@param url string HTTP URL
---@param opts? HTTPOptions Optional HTTP options
---@return State state
function BK.http(url, opts) end

---@param url string HTTPS URL
---@param opts? HTTPOptions Optional HTTP options
---@return State state
function BK.https(url, opts) end

---@param dest string Destination path
---@param opts? CacheOptions Optional cache options
---@return Mount mount
function BK.cache(dest, opts) end

---@param dest string Destination path
---@param opts? SecretOptions Optional secret options
---@return Mount mount
function BK.secret(dest, opts) end

---@param opts? SSHOptions Optional SSH options
---@return Mount mount
function BK.ssh(opts) end

---@param dest string Destination path
---@param opts? TmpfsOptions Optional tmpfs options
---@return Mount mount
function BK.tmpfs(dest, opts) end

---@param state State Source state
---@param dest string Destination path
---@param opts? BindOptions Optional bind options
---@return Mount mount
function BK.bind(state, dest, opts) end

---@param ... State States to merge
---@return State state
function BK.merge(...) end

---@param lower State Lower state
---@param upper State Upper state
---@return State state
function BK.diff(lower, upper) end

---@param state State State to export
---@param opts? ExportConfig Optional export config
function BK.export(state, opts) end

---@param os string OS name
---@param arch string Architecture
---@param variant? string Variant
---@return Platform platform
function BK.platform(os, arch, variant) end
