# Luakit - Buildkit frontend driven by lua
We are implementing the plan in @SPEC.md

## Developer Environment
* Always use `mise` to manage your environment and tasks.
* Use `go` commands to add dependencies. Never editing `go.mod` or `go.sum` files.
* Remember to run `go mod tidy` after adding dependencies.
* Use modern `go` features. `mise run tasks.go-modernize` will automatically modernize your code.

## Verification
* Use `go test` to run tests.
* Use `go vet` to check for potential issues.
* Use `go fmt` to format code.

## Type Definitions
* Lua editor type definitions are in `types/` directory
* When adding/modifying API in `pkg/luavm/api.go` or `pkg/luavm/state.go`, update corresponding `.d.lua` files
* Type definitions use LuaLS/LuaCATS annotations for VSCode, Neovim, and other Lua editors
