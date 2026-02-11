# Luakit - Buildkit frontend driven by lua
We are implementing the plan in @SPEC.md

## Developer Environment
* Always use `mise` to manage your environment and tasks.
* Always build with `mise run build` not `go build`.
* Use `go` commands to add dependencies. Never editing `go.mod` or `go.sum` files.
* Remember to run `go mod tidy` after adding dependencies.
* Use modern `go` features. `mise run go-modernize` will automatically modernize your code.

## Verification
* Use `mise run test` to run tests.
* Use `mise run test:coverage` to generate a coverage report.
* Use `mise run test:integration` to run integration tests (requires BuildKit daemon).
* Use `mise run fmt` to format code.
* Use `mise run lint` to run all linters.
* Use `mise run lint:go` to run golangci-lint.
* Use `mise run lint:lua` to type check Lua files.
* Use `mise run lint:vet` to run go vet.

## Type Definitions
* Lua editor type definitions are in `types/` directory
* When adding/modifying API in `pkg/luavm/api.go` or `pkg/luavm/state.go`, update corresponding `.d.lua` files
* Type definitions use LuaLS/LuaCATS annotations for VSCode, Neovim, and other Lua editors

## Documentation
* Documentation is in `docs/` directory

<!-- opensrc:start -->

## Source Code Reference

Source code for dependencies is available in `opensrc/` for deeper understanding of implementation details.

See `opensrc/sources.json` for the list of available packages and their versions.

Use this source code when you need to understand how a package works internally, not just its types/interface.

### Fetching Additional Source Code

To fetch source code for a package or repository you need to understand, run:

```bash
bunx opensrc <package>           # npm package (e.g., npx opensrc zod)
bunx opensrc pypi:<package>      # Python package (e.g., npx opensrc pypi:requests)
bunx opensrc crates:<package>    # Rust crate (e.g., npx opensrc crates:serde)
bunx opensrc <owner>/<repo>      # GitHub repo (e.g., npx opensrc vercel/ai)
```

<!-- opensrc:end -->
