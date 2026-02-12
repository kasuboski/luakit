-- Python Data Science Container - Luakit Port
-- Full data science environment with ML frameworks

local base = bk.image("python:3.11-slim")

local base_env = base:run({
    "sh", "-c",
    "export DEBIAN_FRONTEND=noninteractive && " ..
    "apt-get update && apt-get install -y --no-install-recommends " ..
    "build-essential gcc g++ git curl wget " ..
    "libopenblas-dev liblapack-dev libhdf5-dev && " ..
    "rm -rf /var/lib/apt/lists/*"
}, {
    env = {
        DEBIAN_FRONTEND = "noninteractive",
        PYTHONUNBUFFERED = "1",
    },
})

local with_pip = base_env:run({
    "sh", "-c",
    "pip install --no-cache-dir --upgrade pip"
}, {
    mounts = {
        bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" }),
    },
})

local req_files = bk.local_("context", { include = { "requirements.txt" } })

local with_deps = with_pip:run({ "pip", "install", "--no-cache-dir", "-r", "requirements.txt" }, {
    cwd = "/workspace",
    mounts = {
        bk.bind(req_files, "/workspace"),
        bk.cache("/root/.cache/pip", { sharing = "shared", id = "pipcache" }),
    },
})

local with_code = with_deps:copy(bk.local_("context"), ".", "/workspace")

local with_user = with_code:run({
    "sh", "-c",
    "useradd -m -u 1000 -s /bin/bash data-scientist && " ..
    "chown -R data-scientist:data-scientist /workspace"
})

bk.export(with_user, {
    env = {
        PYTHONUNBUFFERED = "1",
    },
    user = "data-scientist",
    workdir = "/workspace/notebooks",
    expose = {"8888/tcp"},
    labels = {
        ["org.opencontainers.image.title"] = "Python Data Science Environment",
        ["org.opencontainers.image.description"] = "Complete data science environment with ML frameworks",
        ["org.opencontainers.image.version"] = "1.0.0",
    },
})
