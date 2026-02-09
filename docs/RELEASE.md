# Release Process

This document describes how to create and publish releases for luakit.

## Automated Release Workflow

The release process is automated via GitHub Actions. When you push a version tag (e.g., `v0.1.0`), the workflow will:

1. **Create a GitHub Release** with changelog
2. **Build binaries** for all target platforms:
   - linux/amd64
   - linux/arm64
   - darwin/amd64
   - darwin/arm64
3. **Generate SHA256 checksums** for all binaries
4. **Create a Homebrew formula** for easy installation
5. **Build and publish multi-arch Docker images** to GHCR
6. **Generate and upload SBOM** for security analysis

## Creating a Release

### Step 1: Prepare the release

1. Ensure all tests pass: `make test`
2. Verify the code compiles: `make build`
3. Update the version in `go.mod` if needed

### Step 2: Tag the release

```bash
# Create and push a version tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

The tag must follow semantic versioning: `vX.Y.Z`

### Step 3: Monitor the workflow

The release workflow will automatically start. You can monitor its progress in the Actions tab.

## Release Artifacts

After the workflow completes, the following artifacts will be available in the GitHub release:

- `luakit-{VERSION}-linux-amd64` - Linux x86_64 binary
- `luakit-{VERSION}-linux-arm64` - Linux ARM64 binary
- `luakit-{VERSION}-darwin-amd64` - macOS Intel binary
- `luakit-{VERSION}-darwin-arm64` - macOS Apple Silicon binary
- `checksums.txt` - SHA256 checksums for all binaries
- `luakit.rb` - Homebrew formula

## Installation

### Using Pre-built Binaries

Download the appropriate binary for your platform from the release page:

```bash
# Example for Linux AMD64
wget https://github.com/kasuboski/luakit/releases/download/v0.1.0/luakit-0.1.0-linux-amd64
chmod +x luakit-0.1.0-linux-amd64
sudo mv luakit-0.1.0-linux-amd64 /usr/local/bin/luakit
```

Verify the checksum:

```bash
# Download checksums
wget https://github.com/kasuboski/luakit/releases/download/v0.1.0/checksums.txt

# Verify
sha256sum -c checksums.txt
```

### Using Homebrew

The release includes a Homebrew formula:

```bash
brew install https://github.com/kasuboski/luakit/releases/download/v0.1.0/luakit.rb
```

### Using Docker

Pull the image from GitHub Container Registry:

```bash
docker pull ghcr.io/kasuboski/luakit:latest
# or specific version
docker pull ghcr.io/kasuboski/luakit:v0.1.0
```

## Local Testing

Before creating a release, you can test the build locally:

```bash
# Build all binaries
make build-all

# Generate checksums
make checksums

# Verify all binaries
make verify-release
```

## Versioning

luakit follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: incompatible API changes
- **MINOR**: new functionality in a backwards compatible manner
- **PATCH**: backwards compatible bug fixes

## Troubleshooting

### Workflow fails

Check the workflow logs in the Actions tab. Common issues:

- Missing dependencies: Ensure `go mod` is up to date
- Test failures: Run `make test` locally to reproduce
- Build errors: Verify `make build-all` succeeds locally

### Verification fails

If SHA256 checksums don't match:

1. Re-download the binary
2. Verify you're using the correct checksums.txt file
3. Check that the download completed without errors

### Installation issues

If the binary doesn't run:

1. Verify the architecture matches your system: `file luakit-*`
2. Check execute permissions: `chmod +x luakit-*`
3. Ensure you're using a compatible OS version

## Security

Each release includes:

- **SBOM** (Software Bill of Materials) generated with Anchore
- **Provenance** attestation for Docker images
- **SHA256 checksums** for all binaries

These artifacts help verify the integrity and security of the release.
