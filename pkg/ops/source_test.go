package ops

import (
	"strings"
	"testing"

	pb "github.com/moby/buildkit/solver/pb"
)

func TestNewSourceOp(t *testing.T) {
	attrs := map[string]string{"key": "value"}
	op := NewSourceOp("docker-image://alpine:3.19", attrs)

	if op.Identifier != "docker-image://alpine:3.19" {
		t.Errorf("Expected identifier 'docker-image://alpine:3.19', got '%s'", op.Identifier)
	}

	if op.Attrs["key"] != "value" {
		t.Errorf("Expected attrs key 'value', got '%s'", op.Attrs["key"])
	}
}

func TestImage(t *testing.T) {
	state := Image("alpine:3.19", "test.lua", 10, nil)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if state.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", state.Op().LuaFile())
	}

	if state.Op().LuaLine() != 10 {
		t.Errorf("Expected Lua line 10, got %d", state.Op().LuaLine())
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "docker-image://alpine:3.19"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestImageWithPrefix(t *testing.T) {
	state := Image("docker-image://alpine:3.19", "test.lua", 10, nil)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Identifier != "docker-image://alpine:3.19" {
		t.Errorf("Expected identifier 'docker-image://alpine:3.19', got '%s'", sourceOp.Identifier)
	}
}

func TestImageWithPlatform(t *testing.T) {
	platform := &pb.Platform{
		OS:           "linux",
		Architecture: "arm64",
	}
	state := Image("alpine:3.19", "test.lua", 10, platform)

	if state.Platform() == nil {
		t.Fatal("Expected non-nil platform")
	}

	if state.Platform().OS != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", state.Platform().OS)
	}

	if state.Platform().Architecture != "arm64" {
		t.Errorf("Expected Architecture 'arm64', got '%s'", state.Platform().Architecture)
	}
}

func TestImageEmptyRef(t *testing.T) {
	state := Image("", "test.lua", 10, nil)

	if state != nil {
		t.Error("Expected nil state for empty ref")
	}
}

func TestScratch(t *testing.T) {
	state := Scratch()

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "scratch" {
		t.Errorf("Expected identifier 'scratch', got '%s'", sourceOp.Identifier)
	}
}

func TestLocal(t *testing.T) {
	state := Local("context", "test.lua", 5, nil)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if state.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", state.Op().LuaFile())
	}

	if state.Op().LuaLine() != 5 {
		t.Errorf("Expected Lua line 5, got %d", state.Op().LuaLine())
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "local://context"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestLocalWithPatterns(t *testing.T) {
	opts := &LocalOptions{
		IncludePatterns: []string{"*.go", "*.mod"},
		ExcludePatterns: []string{"*_test.go", "vendor/"},
		SharedKeyHint:   "go-sources",
	}

	state := Local("context", "test.lua", 5, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Attrs["includepattern0"] != "*.go" {
		t.Errorf("Expected includepattern0 '*.go', got '%s'", sourceOp.Attrs["includepattern0"])
	}

	if sourceOp.Attrs["includepattern1"] != "*.mod" {
		t.Errorf("Expected includepattern1 '*.mod', got '%s'", sourceOp.Attrs["includepattern1"])
	}

	if sourceOp.Attrs["excludepattern0"] != "*_test.go" {
		t.Errorf("Expected excludepattern0 '*_test.go', got '%s'", sourceOp.Attrs["excludepattern0"])
	}

	if sourceOp.Attrs["excludepattern1"] != "vendor/" {
		t.Errorf("Expected excludepattern1 'vendor/', got '%s'", sourceOp.Attrs["excludepattern1"])
	}

	if sourceOp.Attrs["sharedkeyhint"] != "go-sources" {
		t.Errorf("Expected sharedkeyhint 'go-sources', got '%s'", sourceOp.Attrs["sharedkeyhint"])
	}
}

func TestLocalEmptyName(t *testing.T) {
	state := Local("", "test.lua", 10, nil)

	if state != nil {
		t.Error("Expected nil state for empty name")
	}
}

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		s      string
		prefix string
		want   bool
	}{
		{"docker-image://alpine", "docker-image://", true},
		{"alpine", "docker-image://", false},
		{"docker-image://", "docker-image://", true},
		{"", "docker-image://", false},
		{"http://example.com", "https://", false},
	}

	for _, tt := range tests {
		if got := hasPrefix(tt.s, tt.prefix); got != tt.want {
			t.Errorf("hasPrefix(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
		}
	}
}

func TestImageIdentifier(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
	}{
		{"alpine:3.19", "docker-image://alpine:3.19"},
		{"docker-image://alpine:3.19", "docker-image://alpine:3.19"},
		{"ubuntu:24.04", "docker-image://ubuntu:24.04"},
	}

	for _, tt := range tests {
		if got := ImageIdentifier(tt.ref); got != tt.expected {
			t.Errorf("ImageIdentifier(%q) = %q, want %q", tt.ref, got, tt.expected)
		}
	}
}

func TestLocalIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"context", "local://context"},
		{"src", "local://src"},
	}

	for _, tt := range tests {
		if got := LocalIdentifier(tt.name); got != tt.expected {
			t.Errorf("LocalIdentifier(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestGitIdentifier(t *testing.T) {
	tests := []struct {
		url      string
		ref      string
		expected string
	}{
		{"https://github.com/moby/buildkit.git", "v0.12.0", "git://https://github.com/moby/buildkit.git#v0.12.0"},
		{"https://github.com/moby/buildkit.git", "", "git://https://github.com/moby/buildkit.git"},
		{"https://github.com/moby/buildkit.git", "main", "git://https://github.com/moby/buildkit.git#main"},
	}

	for _, tt := range tests {
		if got := GitIdentifier(tt.url, tt.ref); got != tt.expected {
			t.Errorf("GitIdentifier(%q, %q) = %q, want %q", tt.url, tt.ref, got, tt.expected)
		}
	}
}

func TestValidateImageRef(t *testing.T) {
	tests := []struct {
		ref string
		err bool
	}{
		{"alpine:3.19", false},
		{"alpine", false},
		{"alpine:latest", false},
		{"library/alpine:3.19", false},
		{"docker.io/library/alpine:3.19", false},
		{"localhost:5000/myimage:latest", false},
		{"gcr.io/myproject/myimage:v1.2.3", false},
		{"ghcr.io/user/repo:tag", false},
		{"", true},
		{"docker-image://alpine:3.19", false},
		{"invalid:ref:with:too:many:parts", true},
		{"", true},
	}

	for _, tt := range tests {
		err := ValidateImageRef(tt.ref)
		if (err != nil) != tt.err {
			t.Errorf("ValidateImageRef(%q) error = %v, want error %v", tt.ref, err, tt.err)
		}
	}
}

func TestValidateLocalName(t *testing.T) {
	tests := []struct {
		name string
		err  bool
	}{
		{"context", false},
		{"src", false},
		{"my-context", false},
		{"my_context", false},
		{"mycontext123", false},
		{"a", false},
		{"", true},
		{"..", true},
		{"../parent", true},
		{"/absolute/path", true},
		{"relative/path", true},
		{"windows\\path", true},
		{"name:with:colons", true},
		{".hidden", true},
		{"..hidden", true},
		{strings.Repeat("a", 257), true},
		{"name@with@symbols", true},
		{"name with spaces", true},
	}

	for _, tt := range tests {
		err := ValidateLocalName(tt.name)
		if (err != nil) != tt.err {
			t.Errorf("ValidateLocalName(%q) error = %v, want error %v", tt.name, err, tt.err)
		}
	}
}

func TestNewSourceState(t *testing.T) {
	op := &pb.SourceOp{
		Identifier: "docker-image://alpine:3.19",
	}
	state := NewSourceState(op, "test.lua", 10)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if state.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", state.Op().LuaFile())
	}

	if state.Op().LuaLine() != 10 {
		t.Errorf("Expected Lua line 10, got %d", state.Op().LuaLine())
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Identifier != "docker-image://alpine:3.19" {
		t.Errorf("Expected identifier 'docker-image://alpine:3.19', got '%s'", sourceOp.Identifier)
	}
}

func TestGit(t *testing.T) {
	state := Git("https://github.com/moby/buildkit.git", "test.lua", 15, nil)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if state.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", state.Op().LuaFile())
	}

	if state.Op().LuaLine() != 15 {
		t.Errorf("Expected Lua line 15, got %d", state.Op().LuaLine())
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestGitWithRef(t *testing.T) {
	opts := &GitOptions{
		Ref: "v0.12.0",
	}
	state := Git("https://github.com/moby/buildkit.git", "test.lua", 15, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#v0.12.0"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestGitWithKeepGitDir(t *testing.T) {
	opts := &GitOptions{
		KeepGitDir: true,
	}
	state := Git("https://github.com/moby/buildkit.git", "test.lua", 15, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	if sourceOp.Attrs["keepgitdir"] != "true" {
		t.Errorf("Expected keepgitdir attribute 'true', got '%s'", sourceOp.Attrs["keepgitdir"])
	}
}

func TestGitWithBothOptions(t *testing.T) {
	opts := &GitOptions{
		Ref:        "main",
		KeepGitDir: true,
	}
	state := Git("https://github.com/moby/buildkit.git", "test.lua", 15, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "git://https://github.com/moby/buildkit.git#main"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}

	if sourceOp.Attrs["keepgitdir"] != "true" {
		t.Errorf("Expected keepgitdir attribute 'true', got '%s'", sourceOp.Attrs["keepgitdir"])
	}
}

func TestGitEmptyURL(t *testing.T) {
	state := Git("", "test.lua", 10, nil)

	if state != nil {
		t.Error("Expected nil state for empty URL")
	}
}

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		url string
		err bool
	}{
		{"https://github.com/moby/buildkit.git", false},
		{"http://github.com/moby/buildkit.git", false},
		{"git://github.com/moby/buildkit.git", false},
		{"ssh://git@github.com/moby/buildkit.git", false},
		{"git+ssh://git@github.com/moby/buildkit.git", false},
		{"git@github.com:moby/buildkit.git", false},
		{"git@gitlab.com:group/project.git", false},
		{"https://gitlab.com/group/project.git", false},
		{"https://bitbucket.org/user/repo.git", false},
		{"", true},
		{"ftp://github.com/repo.git", true},
		{"file:///local/repo.git", true},
		{"invalid-url", true},
	}

	for _, tt := range tests {
		err := ValidateGitURL(tt.url)
		if (err != nil) != tt.err {
			t.Errorf("ValidateGitURL(%q) error = %v, want error %v", tt.url, err, tt.err)
		}
	}
}

func TestHTTP(t *testing.T) {
	state := HTTP("https://example.com/file.tar.gz", "test.lua", 20, nil)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	if state.Op() == nil {
		t.Fatal("Expected non-nil Op")
	}

	if state.Op().LuaFile() != "test.lua" {
		t.Errorf("Expected Lua file 'test.lua', got '%s'", state.Op().LuaFile())
	}

	if state.Op().LuaLine() != 20 {
		t.Errorf("Expected Lua line 20, got %d", state.Op().LuaLine())
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp == nil {
		t.Fatal("Expected SourceOp")
	}

	expectedIdentifier := "https://example.com/file.tar.gz"
	if sourceOp.Identifier != expectedIdentifier {
		t.Errorf("Expected identifier '%s', got '%s'", expectedIdentifier, sourceOp.Identifier)
	}
}

func TestHTTPWithChecksum(t *testing.T) {
	opts := &HTTPOptions{
		Checksum: "sha256:abc123def456",
	}
	state := HTTP("https://example.com/file.tar.gz", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Attrs["checksum"] != "sha256:abc123def456" {
		t.Errorf("Expected checksum 'sha256:abc123def456', got '%s'", sourceOp.Attrs["checksum"])
	}
}

func TestHTTPWithFilename(t *testing.T) {
	opts := &HTTPOptions{
		Filename: "archive.tar.gz",
	}
	state := HTTP("https://example.com/file", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Attrs["filename"] != "archive.tar.gz" {
		t.Errorf("Expected filename 'archive.tar.gz', got '%s'", sourceOp.Attrs["filename"])
	}
}

func TestHTTPWithMode(t *testing.T) {
	opts := &HTTPOptions{
		Mode: 0644,
	}
	state := HTTP("https://example.com/file.tar.gz", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Attrs["mode"] != "420" {
		t.Errorf("Expected mode '420', got '%s'", sourceOp.Attrs["mode"])
	}
}

func TestHTTPWithHeaders(t *testing.T) {
	opts := &HTTPOptions{
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"User-Agent":    "luakit/0.1.0",
		},
	}
	state := HTTP("https://example.com/file.tar.gz", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Attrs["http.header.Authorization"] != "Bearer token123" {
		t.Errorf("Expected http.header.Authorization 'Bearer token123', got '%s'", sourceOp.Attrs["http.header.Authorization"])
	}

	if sourceOp.Attrs["http.header.User-Agent"] != "luakit/0.1.0" {
		t.Errorf("Expected http.header.User-Agent 'luakit/0.1.0', got '%s'", sourceOp.Attrs["http.header.User-Agent"])
	}
}

func TestHTTPWithBasicAuth(t *testing.T) {
	opts := &HTTPOptions{
		Username: "user",
		Password: "pass",
	}
	state := HTTP("https://example.com/file.tar.gz", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()
	if sourceOp.Attrs["http.basicauth"] != "user:pass" {
		t.Errorf("Expected http.basicauth 'user:pass', got '%s'", sourceOp.Attrs["http.basicauth"])
	}
}

func TestHTTPWithAllOptions(t *testing.T) {
	opts := &HTTPOptions{
		Checksum: "sha256:abc123def456",
		Filename: "archive.tar.gz",
		Mode:     0644,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
		Username: "user",
		Password: "pass",
	}
	state := HTTP("https://example.com/file", "test.lua", 20, opts)

	if state == nil {
		t.Fatal("Expected non-nil state")
	}

	sourceOp := state.Op().Op().GetSource()

	if sourceOp.Identifier != "https://example.com/file" {
		t.Errorf("Expected identifier 'https://example.com/file', got '%s'", sourceOp.Identifier)
	}

	if sourceOp.Attrs["checksum"] != "sha256:abc123def456" {
		t.Errorf("Expected checksum 'sha256:abc123def456', got '%s'", sourceOp.Attrs["checksum"])
	}

	if sourceOp.Attrs["filename"] != "archive.tar.gz" {
		t.Errorf("Expected filename 'archive.tar.gz', got '%s'", sourceOp.Attrs["filename"])
	}

	if sourceOp.Attrs["mode"] != "420" {
		t.Errorf("Expected mode '420', got '%s'", sourceOp.Attrs["mode"])
	}

	if sourceOp.Attrs["http.header.Authorization"] != "Bearer token123" {
		t.Errorf("Expected http.header.Authorization 'Bearer token123', got '%s'", sourceOp.Attrs["http.header.Authorization"])
	}

	if sourceOp.Attrs["http.basicauth"] != "user:pass" {
		t.Errorf("Expected http.basicauth 'user:pass', got '%s'", sourceOp.Attrs["http.basicauth"])
	}
}

func TestHTTPEmptyURL(t *testing.T) {
	state := HTTP("", "test.lua", 10, nil)

	if state != nil {
		t.Error("Expected nil state for empty URL")
	}
}

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		url string
		err bool
	}{
		{"https://example.com/file.tar.gz", false},
		{"http://example.com/file.tar.gz", false},
		{"https://github.com/user/repo/archive.tar.gz", false},
		{"https://example.com", false},
		{"http://localhost:8080/file", false},
		{"https://user:pass@example.com/file", false},
		{"https://example.com/path/to/file?query=value", false},
		{"https://example.com/path#fragment", false},
		{"", true},
		{"ftp://example.com/file.tar.gz", true},
		{"file:///local/file.tar.gz", true},
		{"invalid-url", true},
		{"https://", true},
		{"://example.com/file", true},
	}

	for _, tt := range tests {
		err := ValidateHTTPURL(tt.url)
		if (err != nil) != tt.err {
			t.Errorf("ValidateHTTPURL(%q) error = %v, want error %v", tt.url, err, tt.err)
		}
	}
}
