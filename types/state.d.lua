---@meta

---@class ExecOptions
---@field env? table<string, string>
---@field cwd? string
---@field user? string
---@field network? "host"|"none"|"sandbox"
---@field security? "sandbox"|"insecure"
---@field mounts? Mount[]
---@field hostname? string
---@field valid_exit_codes? integer|integer[]|string

---@class CopyOptions
---@field mode? string|integer
---@field follow_symlink? boolean
---@field create_dest_path? boolean
---@field allow_wildcard? boolean
---@field include? string[]
---@field exclude? string[]
---@field owner? ChownOpt

---@class MkdirOptions
---@field mode? string|integer
---@field make_parents? boolean
---@field owner? ChownOpt

---@class MkfileOptions
---@field mode? string|integer
---@field owner? ChownOpt

---@class RmOptions
---@field allow_not_found? boolean
---@field allow_wildcard? boolean

---@class MetadataOptions
---@field description? string
---@field progress_group? string

---@class ChownOpt
---@field user? UserOpt
---@field group? UserOpt

---@alias UserOpt string|integer

---A build state representing a point in the build graph
---@class State
local State = {}

---Execute a command in the state
---@param cmd string|string[] The command to run (string for shell, table for args)
---@param opts? ExecOptions Optional execution options
---@return State state
function State:run(cmd, opts) end

---Copy files from another state
---@param from State The source state
---@param src string The source path
---@param dest string The destination path
---@param opts? CopyOptions Optional copy options
---@return State state
function State:copy(from, src, dest, opts) end

---Create a directory
---@param path string The directory path
---@param opts? MkdirOptions Optional mkdir options
---@return State state
function State:mkdir(path, opts) end

---Create a file with data
---@param path string The file path
---@param data string The file contents
---@param opts? MkfileOptions Optional mkfile options
---@return State state
function State:mkfile(path, data, opts) end

---Remove a file or directory
---@param path string The path to remove
---@param opts? RmOptions Optional rm options
---@return State state
function State:rm(path, opts) end

---Create a symbolic link
---@param oldpath string The existing path
---@param newpath string The new link path
---@return State state
function State:symlink(oldpath, newpath) end

---Add metadata to the state
---@param opts MetadataOptions Metadata options
---@return State state
function State:with_metadata(opts) end
