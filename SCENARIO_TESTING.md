# luakit Validation Scenarios — By Validation Method
These are the scenarios we want to test for luakit.

## Validation Methods

**Method A — Protobuf Inspection**
Parse the `pb.Definition` output from `luakit build` and assert on its structure: Op types, field values, edge topology, metadata, and digests. No BuildKit daemon required. Tests run fast and deterministically.

**Method B — Container Structure Test**
Build the image via BuildKit, then run [container-structure-test](https://github.com/GoogleContainerTools/container-structure-test) against the resulting image. Validates that the *built artifact* is correct: files exist, permissions are set, commands produce expected output, image metadata matches.

---

## Method A — Protobuf Inspection

These scenarios validate correctness by deserializing the `pb.Definition` protobuf output and asserting on its contents. Each scenario lists exactly which protobuf fields to check.

---

### A-01 · Image source resolves shorthand reference

**Lua input:**
```lua
local base = bk.image("ubuntu:24.04")
bk.export(base)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.identifier` | `docker-image://docker.io/library/ubuntu:24.04` |

---

### A-02 · Image source with platform override

**Lua input:**
```lua
local arm = bk.image("ubuntu:24.04", { platform = "linux/arm64" })
bk.export(arm)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `Op.platform.os` | `linux` |
| `Op.platform.architecture` | `arm64` |

---

### A-03 · Image source with resolve mode

**Lua input:**
```lua
local pinned = bk.image("ubuntu:24.04", { resolve = "always" })
bk.export(pinned)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.attrs["image.resolvemode"]` | `pull` (or BuildKit's equivalent for "always") |

---

### A-04 · Scratch source

**Lua input:**
```lua
bk.export(bk.scratch())
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.identifier` | Empty string or BuildKit scratch sentinel |
| Op count in `def` array | 1 |

---

### A-05 · Local source with filters

**Lua input:**
```lua
local ctx = bk.local_("context", {
    include = {"*.go", "go.mod"},
    exclude = {"vendor/"},
    shared_key_hint = "go-sources"
})
bk.export(ctx)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.identifier` | `local://context` |
| `SourceOp.attrs["local.includepattern"]` | `*.go,go.mod` (or equivalent encoding) |
| `SourceOp.attrs["local.excludepatterns"]` | `vendor/` |
| `SourceOp.attrs["local.sharedkeyhint"]` | `go-sources` |

---

### A-06 · Exec with string command and options

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("whoami", {
    cwd = "/tmp",
    user = "nobody",
    env = { FOO = "bar" },
})
bk.export(s)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.meta.args` | `["/bin/sh", "-c", "whoami"]` |
| `ExecOp.meta.cwd` | `/tmp` |
| `ExecOp.meta.user` | `nobody` |
| `ExecOp.meta.env` | Contains `FOO=bar` |
| `Op.inputs` | 1 input edge referencing the SourceOp |

---

### A-07 · Exec with array command (no shell wrapping)

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run({"ls", "-la", "/"}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.meta.args` | `["ls", "-la", "/"]` (no `/bin/sh -c` wrapper) |

---

### A-08 · State immutability produces forked DAG

**Lua input:**
```lua
local s1 = bk.image("alpine:3.19")
local s2 = s1:run("echo a")
local s3 = s1:run("echo b")
bk.export(bk.merge(s2, s3))
```

**Validate in protobuf:**
| Assertion | Expected |
|-----------|----------|
| Total ExecOp nodes | 2 |
| Total SourceOp nodes | 1 |
| Both ExecOps reference the same SourceOp as input | True |
| No edge exists between the two ExecOps | True |
| MergeOp has 2 input edges | True |

---

### A-09 · Output is valid parseable protobuf

**Lua input:** Any valid script.

**When:** `luakit build build.lua > output.pb`

**Validate:** `output.pb` deserializes as `pb.Definition` without error. `def.def` array is non-empty. `def.metadata` map is non-empty.

---

### A-10 · Deterministic output across runs

**Lua input:** Any valid script, run twice.

**Validate:** The two `pb.Definition` byte streams are identical.

---

### A-11 · Copy serializes FileActionCopy with options

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local src = bk.image("golang:1.22")
local s = base:copy(src, "/app/build/", "/app/", {
    owner = { user = "app", group = "app" },
    mode = "0755",
    follow_symlink = true,
    create_dest_path = true,
    include = {"*.so", "server"},
    exclude = {"*.a"},
    allow_wildcard = true,
})
bk.export(s)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `FileOp.actions[0]` type | `FileActionCopy` |
| `FileActionCopy.src` | `/app/build/` |
| `FileActionCopy.dest` | `/app/` |
| `FileActionCopy.owner` | Correct ChownOpt for "app:app" |
| `FileActionCopy.mode` | `0755` |
| `FileActionCopy.followSymlink` | `true` |
| `FileActionCopy.dirCopyContents` / `createDestPath` | `true` |
| `FileActionCopy.includePatterns` | `["*.so", "server"]` |
| `FileActionCopy.excludePatterns` | `["*.a"]` |
| `FileActionCopy.allowWildcard` | `true` |
| Secondary input edge | References the `src` (golang) SourceOp |

---

### A-12 · mkdir serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:mkdir("/app/data/logs", {
    mode = 0755,
    make_parents = true,
    owner = { user = 1000, group = 1000 },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `FileOp.actions[0]` type | `FileActionMkDir` |
| `FileActionMkDir.path` | `/app/data/logs` |
| `FileActionMkDir.makeParents` | `true` |
| `FileActionMkDir.mode` | `0755` |
| `FileActionMkDir.owner.user` | `1000` |
| `FileActionMkDir.owner.group` | `1000` |

---

### A-13 · mkfile serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:mkfile("/etc/config.json", '{"key": "value"}', { mode = 0644 }))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `FileOp.actions[0]` type | `FileActionMkFile` |
| `FileActionMkFile.path` | `/etc/config.json` |
| `FileActionMkFile.data` | `{"key": "value"}` (as bytes) |
| `FileActionMkFile.mode` | `0644` |

---

### A-14 · rm serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:rm("/tmp/junk", { allow_not_found = true, allow_wildcard = true }))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `FileOp.actions[0]` type | `FileActionRm` |
| `FileActionRm.path` | `/tmp/junk` |
| `FileActionRm.allowNotFound` | `true` |
| `FileActionRm.allowWildcard` | `true` |

---

### A-15 · symlink serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:symlink("/usr/bin/python3", "/usr/bin/python"))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `FileOp.actions[0]` type | `FileActionSymlink` |
| `FileActionSymlink.oldpath` | `/usr/bin/python3` |
| `FileActionSymlink.newpath` | `/usr/bin/python` |

---

### A-16 · Cache mount serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("npm ci", {
    mounts = { bk.cache("/root/.npm", { sharing = "shared", id = "npm-cache" }) },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.mounts[N].mountType` | `CACHE` |
| `ExecOp.mounts[N].dest` | `/root/.npm` |
| `ExecOp.mounts[N].cacheOpt.ID` | `npm-cache` |
| `ExecOp.mounts[N].cacheOpt.sharing` | `SHARED` |

---

### A-17 · Secret mount serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("cat /run/secrets/npmrc", {
    mounts = { bk.secret("/run/secrets/npmrc", { id = "npmrc", mode = 0400 }) },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.mounts[N].mountType` | `SECRET` |
| `ExecOp.mounts[N].dest` | `/run/secrets/npmrc` |
| `ExecOp.mounts[N].secretOpt.ID` | `npmrc` |
| `ExecOp.mounts[N].secretOpt.mode` | `0400` |

---

### A-18 · SSH mount serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("ssh-add -l", {
    mounts = { bk.ssh({ id = "default", mode = 0600 }) },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.mounts[N].mountType` | `SSH` |
| `ExecOp.mounts[N].sshOpt.ID` | `default` |
| `ExecOp.mounts[N].sshOpt.mode` | `0600` |

---

### A-19 · Tmpfs mount serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("df /tmp", {
    mounts = { bk.tmpfs("/tmp", { size = 1073741824 }) },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.mounts[N].mountType` | `TMPFS` |
| `ExecOp.mounts[N].dest` | `/tmp` |
| `ExecOp.mounts[N].tmpfsOpt.size` | `1073741824` |

---

### A-20 · Bind mount serialization

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local other = bk.image("ubuntu:24.04")
bk.export(base:run("ls /mnt", {
    mounts = { bk.bind(other, "/mnt", { selector = "/etc", readonly = true }) },
}))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.mounts[N].mountType` | `BIND` |
| `ExecOp.mounts[N].dest` | `/mnt` |
| `ExecOp.mounts[N].selector` | `/etc` |
| `ExecOp.mounts[N].readonly` | `true` |
| `ExecOp.mounts[N].input` | References the `other` SourceOp |

---

### A-21 · Merge DAG topology

**Lua input:**
```lua
local base = bk.image("node:20")
local deps = base:run("npm ci", { cwd = "/app" })
local lint = deps:run("npm run lint", { cwd = "/app" })
local test = deps:run("npm run test", { cwd = "/app" })
local build = deps:run("npm run build", { cwd = "/app" })
bk.export(bk.merge(lint, test, build))
```

**Validate in protobuf:**
| Assertion | Expected |
|-----------|----------|
| `MergeOp` exists | True |
| `MergeOp.inputs` count | 3 |
| No edge exists between lint, test, and build ExecOps | True (they only share the `deps` ancestor) |
| Terminal Op in `def.def` array | The MergeOp |

---

### A-22 · Diff DAG topology

**Lua input:**
```lua
local base = bk.image("ubuntu:24.04")
local installed = base:run("apt-get update && apt-get install -y nginx")
local delta = bk.diff(base, installed)
bk.export(delta)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `DiffOp` exists | True |
| `DiffOp.lower` input | References `base` SourceOp |
| `DiffOp.upper` input | References `installed` ExecOp |

---

### A-23 · Git source serialization

**Lua input:**
```lua
local repo = bk.git("https://github.com/moby/buildkit.git", {
    ref = "v0.12.0",
    keep_git_dir = false,
})
bk.export(repo)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.identifier` | `git://github.com/moby/buildkit.git#v0.12.0` |
| `SourceOp.attrs["git.keepgitdir"]` | `false` or absent |

---

### A-24 · HTTP source serialization

**Lua input:**
```lua
local file = bk.http("https://example.com/archive.tar.gz", {
    checksum = "sha256:abc123",
    filename = "archive.tar.gz",
    chmod = 0644,
})
bk.export(file)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `SourceOp.identifier` | `https://example.com/archive.tar.gz` |
| `SourceOp.attrs["http.checksum"]` | `sha256:abc123` |
| `SourceOp.attrs["http.filename"]` | `archive.tar.gz` |
| `SourceOp.attrs["http.perm"]` | `0644` |

---

### A-25 · Network mode none

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("echo offline", { network = "none" }))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.network` | `NetMode_NONE` |

---

### A-26 · Security mode insecure

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("echo privileged", { security = "insecure" }))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp.security` | `SecurityMode_INSECURE` |

---

### A-27 · Valid exit codes

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
bk.export(base:run("grep maybe /file", { valid_exit_codes = {0, 1} }))
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `ExecOp` metadata or constraints for valid exit codes | Contains `[0, 1]` |

---

### A-28 · Metadata on operation

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("make"):with_metadata({
    description = "Compiling application",
    progress_group = "build",
})
bk.export(s)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `OpMetadata.description` (keyed by ExecOp digest) | `Compiling application` |
| `OpMetadata.progress_group` | `build` |

---

### A-29 · Platform specifier object

**Lua input:**
```lua
local p = bk.platform("linux", "arm64", "v8")
local base = bk.image("ubuntu:24.04", { platform = p })
bk.export(base)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `Op.platform.os` | `linux` |
| `Op.platform.architecture` | `arm64` |
| `Op.platform.variant` | `v8` |

---

### A-30 · Source mapping Lua line numbers

**Lua input:**
```lua
-- line 1: comment
-- line 2: comment
local base = bk.image("alpine:3.19")     -- line 3
-- line 4: comment
local s = base:run("echo hi")            -- line 5
bk.export(s)
```

**Validate in protobuf:**
| Field | Expected Value |
|-------|---------------|
| `Source.locations` entry for SourceOp digest | File: `build.lua`, Line: `3` |
| `Source.locations` entry for ExecOp digest | File: `build.lua`, Line: `5` |

---

### A-31 · Multi-stage DAG edge topology

**Lua input:**
```lua
local builder = bk.image("golang:1.22")
local src = bk.local_("context")
local workspace = builder:copy(src, ".", "/app")
local built = workspace:run("go build -o /out/server .", { cwd = "/app" })
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final)
```

**Validate in protobuf:**
| Assertion | Expected |
|-----------|----------|
| SourceOp count | 3 (golang, local, distroless) |
| ExecOp count | 1 |
| FileOp count | 2 (workspace copy, final copy) |
| Final FileOp has secondary input referencing ExecOp output | True |
| Final FileOp has primary input referencing distroless SourceOp | True |
| Terminal Op in `def.def` | The final FileOp |

---

### A-32 through A-41 · Golden file tests

Each of the following produces a `pb.Definition` that must match a checked-in golden file byte-for-byte.

| ID | Test Case | Input Summary |
|----|-----------|---------------|
| A-32 | `simple_image` | `bk.image()` + `bk.export()` |
| A-33 | `exec_basic` | `image → run → export` |
| A-34 | `exec_mounts` | `run` with cache, secret, ssh, tmpfs |
| A-35 | `file_copy` | Two images + cross-state copy |
| A-36 | `file_ops` | Chained mkdir → mkfile → rm → symlink |
| A-37 | `merge_basic` | Three parallel branches merged |
| A-38 | `diff_basic` | `diff(base, installed)` |
| A-39 | `multi_stage` | Builder → runtime copy pattern |
| A-40 | `platform` | Cross-platform image pull |
| A-41 | `complex_dag` | Full Go app: parallel, cache, multi-stage |

**Validate:** `go test ./test/integration/ -run=Golden` — zero diff against golden files. Running twice confirms determinism (A-10).

---

## Method B — Container Structure Test

These scenarios build an image via BuildKit, then validate the resulting image using `container-structure-test`. Each scenario includes the YAML test config.

---

### B-01 · Minimal image contains created file

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local result = base:run("echo hello > /greeting.txt")
bk.export(result)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "greeting file exists"
    path: "/greeting.txt"
    shouldExist: true

fileContentTests:
  - name: "greeting file content"
    path: "/greeting.txt"
    expectedContents: ["hello"]
```

---

### B-02 · Multi-stage Go binary in distroless image

**Lua input:**
```lua
local builder = bk.image("golang:1.22-bookworm")
local src = bk.local_("context")
local deps = builder:mkdir("/app"):copy(src, "go.mod", "/app/go.mod")
                                   :copy(src, "go.sum", "/app/go.sum")
local downloaded = deps:run("go mod download", { cwd = "/app" })
local workspace = downloaded:copy(src, ".", "/app")
local built = workspace:run(
    "CGO_ENABLED=0 go build -o /out/server ./cmd/server",
    { cwd = "/app", mounts = { bk.cache("/root/.cache/go-build") } }
)
local runtime = bk.image("gcr.io/distroless/static-debian12")
local final = runtime:copy(built, "/out/server", "/server")
bk.export(final, { entrypoint = {"/server"} })
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "server binary exists"
    path: "/server"
    shouldExist: true
    permissions: "-rwxr-xr-x"

fileExistenceTests:
  - name: "no Go toolchain in runtime"
    path: "/usr/local/go/bin/go"
    shouldExist: false

commandTests:
  - name: "server binary is executable"
    command: "/server"
    args: ["--version"]
    exitCode: 0

metadataTest:
  entrypoint: ["/server"]
```

---

### B-03 · File created by mkfile has correct content and permissions

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:mkfile("/etc/app.conf", "listen=8080\nworkers=4", { mode = 0644 })
bk.export(s)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "config file exists"
    path: "/etc/app.conf"
    shouldExist: true
    permissions: "-rw-r--r--"

fileContentTests:
  - name: "config file content"
    path: "/etc/app.conf"
    expectedContents: ["listen=8080", "workers=4"]
```

---

### B-04 · mkdir creates directory tree

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:mkdir("/app/data/logs", { mode = 0755, make_parents = true })
bk.export(s)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "nested directory exists"
    path: "/app/data/logs"
    shouldExist: true
    permissions: "drwxr-xr-x"
  - name: "parent directory exists"
    path: "/app/data"
    shouldExist: true
```

---

### B-05 · rm removes file

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:mkfile("/tmp/deleteme", "gone"):rm("/tmp/deleteme")
bk.export(s)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "deleted file is gone"
    path: "/tmp/deleteme"
    shouldExist: false
```

---

### B-06 · symlink is created

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("echo '#!/bin/sh\necho hello' > /usr/local/bin/greet && chmod +x /usr/local/bin/greet")
local linked = s:symlink("/usr/local/bin/greet", "/usr/local/bin/hi")
bk.export(linked)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
commandTests:
  - name: "symlink works"
    command: "/usr/local/bin/hi"
    expectedOutput: ["hello"]
    exitCode: 0
```

---

### B-07 · Copy with mode applied

**Lua input:**
```lua
local builder = bk.image("alpine:3.19")
local built = builder:run("echo '#!/bin/sh\necho running' > /out/app && chmod +x /out/app")
local runtime = bk.image("alpine:3.19")
local final = runtime:copy(built, "/out/app", "/usr/local/bin/app", { mode = "0755" })
bk.export(final)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "copied binary exists with correct mode"
    path: "/usr/local/bin/app"
    shouldExist: true
    permissions: "-rwxr-xr-x"

commandTests:
  - name: "copied binary runs"
    command: "/usr/local/bin/app"
    expectedOutput: ["running"]
    exitCode: 0
```

---

### B-08 · Image config: entrypoint, cmd, env, user, workdir, expose

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("adduser -D app"):mkdir("/app")
bk.export(s, {
    entrypoint = {"/bin/sh"},
    cmd = {"-c", "echo configured"},
    env = {"APP_ENV=production", "PATH=/usr/local/bin:/usr/bin:/bin"},
    expose = {"8080/tcp", "9090/tcp"},
    user = "app",
    workdir = "/app",
    labels = {
        ["org.opencontainers.image.source"] = "https://github.com/example/app",
    },
})
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
metadataTest:
  entrypoint: ["/bin/sh"]
  cmd: ["-c", "echo configured"]
  workdir: "/app"
  user: "app"
  exposedPorts: ["8080/tcp", "9090/tcp"]
  env:
    - key: "APP_ENV"
      value: "production"
  labels:
    - key: "org.opencontainers.image.source"
      value: "https://github.com/example/app"

commandTests:
  - name: "default command output"
    command: "/bin/sh"
    args: ["-c", "echo configured"]
    expectedOutput: ["configured"]
    exitCode: 0
```

---

### B-09 · Diff + Merge applies delta correctly

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local installed = base:run("apk add --no-cache curl")
local delta = bk.diff(base, installed)

local target = bk.image("alpine:3.19")
local final = bk.merge(target, delta)
bk.export(final)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
commandTests:
  - name: "curl is available from diff delta"
    command: "curl"
    args: ["--version"]
    exitCode: 0
```

---

### B-10 · Local context files included in build

**Lua input:** (assumes context directory contains `hello.txt` with "world")
```lua
local base = bk.image("alpine:3.19")
local ctx = bk.local_("context")
local final = base:copy(ctx, "hello.txt", "/data/hello.txt")
bk.export(final)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "context file copied"
    path: "/data/hello.txt"
    shouldExist: true

fileContentTests:
  - name: "context file content"
    path: "/data/hello.txt"
    expectedContents: ["world"]
```

---

### B-11 · Git source content available

**Lua input:**
```lua
local repo = bk.git("https://github.com/moby/buildkit.git", { ref = "v0.12.0" })
local base = bk.image("alpine:3.19")
local final = base:copy(repo, "README.md", "/README.md")
bk.export(final)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "README from git source exists"
    path: "/README.md"
    shouldExist: true

fileContentTests:
  - name: "README has buildkit content"
    path: "/README.md"
    expectedContents: ["BuildKit"]
```

---

### B-12 · Secret mount available during build, absent from image

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("cat /run/secrets/mytoken > /proof.txt", {
    mounts = { bk.secret("/run/secrets/mytoken", { id = "mytoken" }) },
})
bk.export(s)
```

**Build command:** `luakit build build.lua | buildctl build --no-frontend --secret id=mytoken,src=token.txt ...`

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "proof file from secret exists"
    path: "/proof.txt"
    shouldExist: true
  - name: "secret is not baked into image"
    path: "/run/secrets/mytoken"
    shouldExist: false

fileContentTests:
  - name: "proof file has secret value"
    path: "/proof.txt"
    expectedContents: ["TOKEN_CONTENT"]
```

---

### B-13 · Network mode none blocks access

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("apk add --no-cache curl || echo 'network blocked' > /result.txt", {
    network = "none",
})
bk.export(s)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileContentTests:
  - name: "network was blocked"
    path: "/result.txt"
    expectedContents: ["network blocked"]
```

---

### B-14 · Valid exit codes accept non-zero

**Lua input:**
```lua
local base = bk.image("alpine:3.19")
local s = base:run("sh -c 'exit 1'", { valid_exit_codes = {0, 1} })
local final = s:mkfile("/ok.txt", "passed")
bk.export(final)
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "build continued after exit code 1"
    path: "/ok.txt"
    shouldExist: true
```

---

### B-15 · Real-world port: Node.js app

**Lua input:**
```lua
local base = bk.image("node:20-slim")
local src = bk.local_("context")
local workspace = base:mkdir("/app"):copy(src, "package.json", "/app/package.json")
                                     :copy(src, "package-lock.json", "/app/package-lock.json")
local deps = workspace:run("npm ci --production", {
    cwd = "/app",
    mounts = { bk.cache("/root/.npm") },
})
local full = deps:copy(src, ".", "/app")
local built = full:run("npm run build", { cwd = "/app" })

local runtime = bk.image("nginx:alpine")
local final = runtime:copy(built, "/app/dist", "/usr/share/nginx/html")
bk.export(final, { expose = {"80/tcp"} })
```

**container-structure-test config:**
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "built assets exist"
    path: "/usr/share/nginx/html/index.html"
    shouldExist: true

metadataTest:
  exposedPorts: ["80/tcp"]

commandTests:
  - name: "nginx config is valid"
    command: "nginx"
    args: ["-t"]
    exitCode: 0
```

---

### B-16 · Frontend gateway image mode

**Build command:**
```bash
buildctl build \
  --frontend=gateway.v0 \
  --opt source=luakit:test \
  --local context=. \
  --output type=docker,name=test-frontend | docker load
```

**container-structure-test config:** (same as whatever the build.lua in the context defines — reuse B-01 config)
```yaml
schemaVersion: "2.0.0"
fileExistenceTests:
  - name: "frontend-built image has expected file"
    path: "/greeting.txt"
    shouldExist: true
```

**Additional assertion:** The build completes successfully through the gateway protocol (exit code 0 from `buildctl`).

---

## Summary Matrix

| Method | Scenario IDs | Count | Requires BuildKit |
|--------|-------------|-------|-------------------|
| **A — Protobuf Inspection** | A-01 through A-41 | 41 | No |
| **B — Container Structure Test** | B-01 through B-16 | 16 | Yes |
| **Total** | | **57** | |

### Not covered by A or B

The following scenario categories from the original 88 validate CLI behavior, Lua VM behavior, or non-functional requirements. They are tested through Go unit tests, Lua-level assertions, and manual/CI checks rather than protobuf inspection or container structure tests:

| Category | Original IDs | Test Method |
|----------|-------------|-------------|
| Sandbox & security | SCN-5.1 – 5.6 | Go unit tests (`sandbox_test.go`) |
| Error handling | SCN-6.1 – 6.12 | Go unit tests + Lua test harness |
| Developer experience CLI | SCN-4.1 – 4.7 | Go integration tests (CLI output assertions) |
| Performance | SCN-9.1 – 9.5 | Go benchmarks (`go test -bench`) |
| Usability | SCN-11.1 | Manual acceptance test |
| Cross-compilation | SCN-12.1 | CI matrix build |
