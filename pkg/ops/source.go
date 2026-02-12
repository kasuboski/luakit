package ops

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/distribution/reference"
	pb "github.com/moby/buildkit/solver/pb"

	"github.com/kasuboski/luakit/pkg/dag"
)

type LocalOptions struct {
	IncludePatterns []string
	ExcludePatterns []string
	SharedKeyHint   string
}

type GitOptions struct {
	Ref        string
	KeepGitDir bool
}

type HTTPOptions struct {
	Checksum string
	Filename string
	Mode     int32
	Headers  map[string]string
	Username string
	Password string // #nosec G117 -- Password field for HTTP basic auth, not a hard-coded secret
}

type ImageOptions struct {
	Platform      string
	ResolveDigest bool
}

const (
	dockerImagePrefix = "docker-image://"
	localPrefix       = "local://"
	gitPrefix         = "git://"
	scratchIdentifier = "scratch"
)

func NewSourceOp(identifier string, attrs map[string]string) *pb.SourceOp {
	return &pb.SourceOp{
		Identifier: identifier,
		Attrs:      attrs,
	}
}

func NewSourceState(op *pb.SourceOp, luaFile string, luaLine int) *dag.State {
	pbOp := &pb.Op{
		Op: &pb.Op_Source{
			Source: op,
		},
	}
	node := dag.NewOpNode(pbOp, luaFile, luaLine)
	return dag.NewState(node)
}

func Image(ref string, luaFile string, luaLine int, platform *pb.Platform, opts *ImageOptions) *dag.State {
	if ref == "" {
		return nil
	}

	if err := ValidateImageRef(ref); err != nil {
		log.Printf("validation failed: %v", err)
		return nil
	}

	identifier := ref
	if !hasPrefix(ref, dockerImagePrefix) {
		normalizedRef, err := reference.ParseNormalizedNamed(ref)
		if err != nil {
			identifier = dockerImagePrefix + ref
		} else {
			identifier = dockerImagePrefix + normalizedRef.String()
		}
	}

	op := NewSourceOp(identifier, nil)
	state := NewSourceState(op, luaFile, luaLine)

	if platform != nil {
		state = state.WithPlatform(platform)
	}

	// Default to resolving digest (true if nil, or user-specified value)
	resolveDigest := opts == nil || opts.ResolveDigest
	if resolveDigest {
		// Set resolveConfig on the OpNode itself
		opNode := state.Op()
		opNode.SetResolveConfig(true)
	}

	return state
}

func Scratch() *dag.State {
	op := NewSourceOp(scratchIdentifier, nil)
	return NewSourceState(op, "", 0)
}

func Local(name string, luaFile string, luaLine int, opts *LocalOptions) *dag.State {
	if name == "" {
		return nil
	}

	if err := ValidateLocalName(name); err != nil {
		log.Printf("validation failed: %v", err)
		return nil
	}

	identifier := localPrefix + name
	attrs := make(map[string]string)

	if opts != nil {
		if len(opts.IncludePatterns) > 0 {
			for i, pattern := range opts.IncludePatterns {
				attrs[fmt.Sprintf("includepattern%d", i)] = pattern
			}
		}
		if len(opts.ExcludePatterns) > 0 {
			for i, pattern := range opts.ExcludePatterns {
				attrs[fmt.Sprintf("excludepattern%d", i)] = pattern
			}
		}
		if opts.SharedKeyHint != "" {
			attrs["sharedkeyhint"] = opts.SharedKeyHint
		}
	}

	op := NewSourceOp(identifier, attrs)
	return NewSourceState(op, luaFile, luaLine)
}

func Git(url string, luaFile string, luaLine int, opts *GitOptions) *dag.State {
	if url == "" {
		return nil
	}

	if err := ValidateGitURL(url); err != nil {
		log.Printf("validation failed: %v", err)
		return nil
	}

	ref := ""
	keepGitDir := false
	if opts != nil {
		ref = opts.Ref
		keepGitDir = opts.KeepGitDir
	}

	identifier := GitIdentifier(url, ref)
	attrs := make(map[string]string)

	if keepGitDir {
		attrs["keepgitdir"] = "true"
	}

	op := NewSourceOp(identifier, attrs)
	return NewSourceState(op, luaFile, luaLine)
}

func HTTP(url string, luaFile string, luaLine int, opts *HTTPOptions) *dag.State {
	if url == "" {
		return nil
	}

	if err := ValidateHTTPURL(url); err != nil {
		log.Printf("validation failed: %v", err)
		return nil
	}

	identifier := url
	attrs := make(map[string]string)

	if opts != nil {
		if opts.Checksum != "" {
			attrs["checksum"] = opts.Checksum
		}
		if opts.Filename != "" {
			attrs["filename"] = opts.Filename
		}
		if opts.Mode != 0 {
			attrs["mode"] = fmt.Sprintf("%d", opts.Mode)
		}
		for key, value := range opts.Headers {
			attrs["http.header."+key] = value
		}
		if opts.Username != "" && opts.Password != "" {
			if strings.Contains(opts.Username, ":") {
				log.Printf("validation failed: username must not contain colon")
				return nil
			}
			attrs["http.basicauth"] = opts.Username + ":" + opts.Password
		}
	}

	op := NewSourceOp(identifier, attrs)
	return NewSourceState(op, luaFile, luaLine)
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func ImageIdentifier(ref string) string {
	if hasPrefix(ref, dockerImagePrefix) {
		return ref
	}
	return dockerImagePrefix + ref
}

func LocalIdentifier(name string) string {
	return localPrefix + name
}

func GitIdentifier(url string, ref string) string {
	identifier := gitPrefix + url
	if ref != "" {
		identifier += "#" + ref
	}
	return identifier
}

func ValidateImageRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("image reference must not be empty")
	}

	actualRef := ref
	if hasPrefix(ref, dockerImagePrefix) {
		actualRef = ref[len(dockerImagePrefix):]
	}

	if err := validateNormalizedReference(actualRef); err == nil {
		return nil
	}

	if err := validateDigestFormat(actualRef, ref); err == nil {
		return nil
	}

	if err := validateTagFormat(actualRef); err == nil {
		return nil
	}

	if err := validateSimpleName(actualRef); err == nil {
		return nil
	}

	return fmt.Errorf("invalid image reference %q", ref)
}

func validateNormalizedReference(actualRef string) error {
	parsed, err := reference.ParseNormalizedNamed(actualRef)
	if err != nil {
		return err
	}

	if _, ok := parsed.(reference.Digested); ok {
		return nil
	}

	if parsed.Name() != "" {
		return nil
	}

	return fmt.Errorf("invalid normalized reference")
}

func validateDigestFormat(actualRef, originalRef string) error {
	if strings.Count(actualRef, "@") != 1 {
		return fmt.Errorf("not a digest format")
	}

	parts := strings.SplitN(actualRef, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid digest format")
	}

	if strings.Contains(parts[0], ":") || strings.Contains(parts[0], "@") {
		return fmt.Errorf("invalid image reference %q: image name should not contain : or @ before digest", originalRef)
	}

	return nil
}

func validateTagFormat(actualRef string) error {
	if strings.Count(actualRef, ":") != 1 || strings.Contains(actualRef, "@") {
		return fmt.Errorf("not a tag format")
	}

	parts := strings.SplitN(actualRef, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid tag format")
	}

	return nil
}

func validateSimpleName(actualRef string) error {
	if actualRef == "" || strings.Contains(actualRef, ":") || strings.Contains(actualRef, "@") {
		return fmt.Errorf("not a simple name")
	}

	return nil
}

func ValidateLocalName(name string) error {
	if name == "" {
		return fmt.Errorf("local name must not be empty")
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("local name must not contain path traversal sequences: %q", name)
	}

	if strings.ContainsAny(name, "/\\:") {
		return fmt.Errorf("local name must not contain path separators or colons: %q", name)
	}

	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("local name must not start with a dot: %q", name)
	}

	if len(name) > 256 {
		return fmt.Errorf("local name must not exceed 256 characters: %q", name)
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("local name must contain only alphanumeric characters, hyphens, and underscores: %q", name)
	}

	return nil
}

func ValidateGitURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("git URL must not be empty")
	}

	var u *url.URL
	var err error

	if strings.HasPrefix(rawURL, "git@") {
		sshURL := strings.Replace(rawURL, ":", "/", 1)
		sshURL = "ssh://" + sshURL
		u, err = url.Parse(sshURL)
		if err != nil {
			return fmt.Errorf("invalid git SSH URL: %w", err)
		}
	} else {
		u, err = url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("invalid git URL: %w", err)
		}
	}

	if u.Scheme != "" {
		supportedSchemes := map[string]bool{
			"http":    true,
			"https":   true,
			"git":     true,
			"ssh":     true,
			"git+ssh": true,
		}
		if !supportedSchemes[u.Scheme] {
			return fmt.Errorf("unsupported git URL scheme %q, supported schemes are: http, https, git, ssh, git+ssh", u.Scheme)
		}
	}

	if u.Host == "" {
		return fmt.Errorf("git URL must have a host: %q", rawURL)
	}

	return nil
}

func ValidateHTTPURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("http URL must not be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got %q: %s", u.Scheme, rawURL)
	}

	if u.Host == "" {
		return fmt.Errorf("URL must have a host: %s", rawURL)
	}

	return nil
}
