## Luakit - Buildkit frontend driven by lua
We are implementing the plan in @SPEC.md

## Developer Environment
* Always use `mise` to manage your environment.
* Use `go` commands to add dependencies. Never editing `go.mod` or `go.sum` files.
* Remember to run `go mod tidy` after adding dependencies.
* Use modern `go` features. `mise run tasks.go-modernize` will automatically modernize your code.

## Verification
* Use `go test` to run tests.
* Use `go vet` to check for potential issues.
* Use `go fmt` to format code.
