package cicd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func projectRoot() string {
	root, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "."
		}
		root = parent
	}
}

func TestWorkflowSyntax(t *testing.T) {
	workflowsDir := filepath.Join(projectRoot(), ".github/workflows")

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		t.Fatalf("Failed to read workflows directory: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			workflowPath := filepath.Join(workflowsDir, entry.Name())

			if _, err := exec.LookPath("yq"); err == nil {
				cmd := exec.Command("yq", "eval", ".", workflowPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Errorf("Workflow syntax error in %s: %v\nOutput: %s", workflowPath, err, string(output))
				}
			} else {
				data, err := os.ReadFile(workflowPath)
				if err != nil {
					t.Fatalf("Failed to read workflow file: %v", err)
				}
				if len(data) == 0 {
					t.Errorf("Workflow file is empty: %s", workflowPath)
				}
			}
		})
	}
}

func TestCIWorkflowContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), ".github/workflows/ci.yml"))
	if err != nil {
		t.Fatalf("Failed to read ci.yml: %v", err)
	}

	content := string(data)

	tests := []struct {
		name     string
		required string
	}{
		{"has lint job", "name: Lint"},
		{"has fmt job", "name: Format Check"},
		{"has vet job", "name: Vet"},
		{"has test job", "name: Test"},
		{"has integration test job", "name: Integration Tests"},
		{"has build job", "name: Build"},
		{"has build image job", "name: Build Docker Image"},
		{"has security scan job", "name: Security Scan"},
		{"has benchmark job", "name: Benchmark"},
		{"supports amd64", "linux/amd64"},
		{"supports arm64", "linux/arm64"},
		{"uses golangci-lint", "golangci/golangci-lint-action@v6"},
		{"uses codecov", "codecov/codecov-action@v4"},
		{"uses benchmark action", "benchmark-action/github-action-benchmark@v1"},
		{"uses QEMU", "docker/setup-qemu-action@v3"},
		{"uses Buildx", "docker/setup-buildx-action@v3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.required) {
				t.Errorf("Missing required content: %s", tt.required)
			}
		})
	}
}

func TestBuildWorkflowContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), ".github/workflows/build.yml"))
	if err != nil {
		t.Fatalf("Failed to read build.yml: %v", err)
	}

	content := string(data)

	tests := []struct {
		name     string
		required string
	}{
		{"has build and push job", "name: Build and Push"},
		{"has verify job", "name: Verify Image"},
		{"has security scan job", "name: Security Scan"},
		{"supports amd64", "linux/amd64"},
		{"supports arm64", "linux/arm64"},
		{"uses GitHub registry", "REGISTRY: ghcr.io"},
		{"uses build-push-action", "docker/build-push-action@v6"},
		{"uses metadata action", "docker/metadata-action@v5"},
		{"uses QEMU", "docker/setup-qemu-action@v3"},
		{"uses Buildx", "docker/setup-buildx-action@v3"},
		{"generates SBOM", "sbom: true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.required) {
				t.Errorf("Missing required content: %s", tt.required)
			}
		})
	}
}

func TestReleaseWorkflowContent(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), ".github/workflows/release.yml"))
	if err != nil {
		t.Fatalf("Failed to read release.yml: %v", err)
	}

	content := string(data)

	tests := []struct {
		name     string
		required string
	}{
		{"triggers on tags", "tags:\n      - 'v*.*.*'"},
		{"has release job", "name: Create Release"},
		{"has build job", "name: Build Multi-Arch Image"},
		{"generates changelog", "Generate changelog"},
		{"uses gh-release action", "softprops/action-gh-release@v2"},
		{"generates SBOM", "anchore/sbom-action@v0"},
		{"builds multi-platform", "platforms: linux/amd64,linux/arm64"},
		{"uses provenance", "provenance: true"},
		{"uses sbom", "sbom: true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.required) {
				t.Errorf("Missing required content: %s", tt.required)
			}
		})
	}
}

func TestDockerfileSupportsCrossPlatform(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "Dockerfile"))
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	content := string(data)

	tests := []struct {
		name     string
		required string
	}{
		{"uses TARGETPLATFORM", "ARG TARGETPLATFORM"},
		{"parses GOOS", "GOOS="},
		{"parses GOARCH", "GOARCH="},
		{"builds static binary", "CGO_ENABLED=0"},
		{"sets GOOS", "GOOS=$GOOS"},
		{"sets GOARCH", "GOARCH=$GOARCH"},
		{"uses build-args", "ldflags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.required) {
				t.Errorf("Missing required content: %s", tt.required)
			}
		})
	}
}

func TestCrossPlatformBuildScript(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(projectRoot(), "test_cross_platform_build.sh"))
	if err != nil {
		t.Fatalf("Failed to read test script: %v", err)
	}

	content := string(data)

	tests := []struct {
		name     string
		required string
	}{
		{"tests amd64", "linux/amd64"},
		{"tests arm64", "linux/arm64"},
		{"uses buildx", "docker buildx build"},
		{"uses platform flag", "--platform"},
		{"verifies architecture", "x86-64"},
		{"verifies arm64", "aarch64"},
		{"checks static linking", "statically linked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(content, tt.required) {
				t.Errorf("Missing required content: %s", tt.required)
			}
		})
	}
}
