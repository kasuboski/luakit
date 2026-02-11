package resolver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// ImageConfig holds resolved image configuration
type ImageConfig struct {
	Ref      string
	Digest   string
	Config   *ocispec.Image
	Platform ocispec.Platform
}

// Resolver resolves image configurations from registries
type Resolver struct {
	cache *Cache
}

// NewResolver creates a new image resolver
func NewResolver() *Resolver {
	return &Resolver{
		cache: NewCache(),
	}
}

// getAuthCreds reads Docker config and returns auth credentials for a host
func getAuthCreds(host string) (string, string, error) {
	logrus.Debugf(" Looking up auth for host: %s\n", host)

	// Try to read Docker config from standard locations
	configPaths := []string{}

	// Add standard Docker config location
	if home, err := os.UserHomeDir(); err == nil {
		configPaths = append(configPaths, filepath.Join(home, ".docker", "config.json"))
	}

	// Add Lima Docker config location if it exists
	if limaHome := os.Getenv("LIMA_HOME"); limaHome != "" {
		configPaths = append(configPaths, filepath.Join(limaHome, "docker", ".docker", "config.json"))
	}

	// Add DOCKER_CONFIG if set
	if dockerConfig := os.Getenv("DOCKER_CONFIG"); dockerConfig != "" {
		configPaths = append(configPaths, filepath.Join(dockerConfig, "config.json"))
		configPaths = append(configPaths, dockerConfig)
	}

	// Generate list of possible host keys to try (Docker config formats vary)
	hostKeys := generateHostKeys(host)

	for _, configPath := range configPaths {
		logrus.Debugf(" Checking config path: %s\n", configPath)
		config, err := readDockerConfig(configPath)
		if err != nil {
			logrus.Debugf(" Failed to read config: %v\n", err)
			continue
		}

		logrus.Debugf(" Found auths: %v\n", getKeys(config.Auths))

		// Check for auth in the config
		if config.Auths != nil {
			// Try each possible host key format
			for _, hostKey := range hostKeys {
				if auth, ok := config.Auths[hostKey]; ok {
					logrus.Debugf(" Found auth for host %s using key %s\n", host, hostKey)
					username, password, err := decodeAuth(auth.Auth)
					if err != nil {
						continue
					}
					return username, password, nil
				}
			}
		}
	}

	// No auth found - return empty credentials (may still work for public images)
	logrus.Debugf(" No auth found for host %s\n", host)
	return "", "", nil
}

func generateHostKeys(host string) []string {
	keys := []string{host}

	// Docker Hub special cases - different config formats use different keys
	if strings.HasSuffix(host, ".docker.io") || host == "docker.io" {
		keys = append(keys,
			"https://index.docker.io/v1/",
			"index.docker.io",
			"registry-1.docker.io",
		)
	}

	// Try with protocol prefixes
	keys = append(keys, "https://"+host)

	return keys
}

func getKeys(m map[string]authEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// dockerConfig represents the Docker config.json structure
type dockerConfig struct {
	Auths       map[string]authEntry `json:"auths"`
	CredsStore  string               `json:"credsStore"`
	CredHelpers map[string]string    `json:"credHelpers"`
}

type authEntry struct {
	Auth string `json:"auth"`
}

// readDockerConfig reads and parses a Docker config file
func readDockerConfig(path string) (*dockerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	logrus.Debugf(" Config file content (%d bytes): %s\n", len(data), string(data))

	var config dockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logrus.Debugf(" Failed to unmarshal config: %v\n", err)
		return nil, err
	}

	logrus.Debugf(" Parsed config with %d auths, credsStore=%s, credHelpers=%v\n",
		len(config.Auths), config.CredsStore, config.CredHelpers)

	return &config, nil
}

// decodeAuth decodes a base64-encoded auth string
func decodeAuth(auth string) (string, string, error) {
	if auth == "" {
		return "", "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid auth format")
	}

	return parts[0], parts[1], nil
}

// Resolve resolves the image configuration for given reference and platform
func (r *Resolver) Resolve(ctx context.Context, refStr string, platform ocispec.Platform) (*ImageConfig, error) {
	// Normalize reference - strip docker-image:// prefix
	ref := strings.TrimPrefix(refStr, "docker-image://")

	// Add default tag if none specified (required for some registries like GCR)
	if !strings.Contains(ref, ":") && !strings.Contains(ref, "@") {
		ref = ref + ":latest"
	}

	logrus.Debugf("Resolving image: %s for platform %+v", ref, platform)

	// Check cache first
	if cached, err := r.cache.Get(ref); cached != nil {
		if cachedConfig, ok := cached.(*ImageConfig); ok {
			return cachedConfig, err
		}
		return nil, fmt.Errorf("cached value has wrong type")
	}

	// Create Docker resolver with auth support
	resolver := docker.NewResolver(docker.ResolverOptions{
		Credentials: func(host string) (string, string, error) {
			return getAuthCreds(host)
		},
	})

	// Resolve the reference to get the descriptor
	name, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", ref, err)
	}

	// Create a fetcher to fetch the manifest
	fetcher, err := resolver.Fetcher(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetcher for %s: %w", name, err)
	}

	// Read the manifest
	reader, err := fetcher.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest for %s: %w", name, err)
	}
	defer func() { _ = reader.Close() }()

	// Read all bytes to allow multiple decode attempts
	manifestBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest for %s: %w", name, err)
	}

	logrus.Debugf(" Descriptor media type: %s\n", desc.MediaType)
	logrus.Debugf(" Manifest bytes length: %d\n", len(manifestBytes))

	// Parse the manifest based on media type
	var manifest ocispec.Manifest
	//TODO: should this not use constants for all media types?
	isIndex := desc.MediaType == ocispec.MediaTypeImageIndex || desc.MediaType == "application/vnd.oci.image.index.v1+json" || desc.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json"

	if isIndex {
		// Parse as index (multi-platform image)
		logrus.Debugf(" Parsing as multi-platform index\n")
		var index ocispec.Index
		if err := json.Unmarshal(manifestBytes, &index); err != nil {
			return nil, fmt.Errorf("failed to parse index manifest for %s: %w", name, err)
		}
		logrus.Debugf(" Parsed index with %d manifests\n", len(index.Manifests))

		// Find matching platform in index
		matcher := platforms.Only(platform)
		for i, mani := range index.Manifests {
			logrus.Debugf(" Checking manifest %d: %+v\n", i, mani.Platform)
			if mani.Platform != nil && matcher.Match(*mani.Platform) {
				// Fetch the actual manifest for this platform
				logrus.Debugf(" Found matching platform: %+v, digest: %s\n", mani.Platform, mani.Digest.String())
				platformReader, err := fetcher.Fetch(ctx, mani)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch platform manifest for %s: %w", name, err)
				}
				defer func() { _ = platformReader.Close() }()

				platformBytes, err := io.ReadAll(platformReader)
				if err != nil {
					return nil, fmt.Errorf("failed to read platform manifest for %s: %w", name, err)
				}

				logrus.Debugf(" Platform manifest bytes: %s\n", string(platformBytes))

				if err := json.Unmarshal(platformBytes, &manifest); err != nil {
					return nil, fmt.Errorf("failed to decode platform manifest for %s: %w", name, err)
				}
				logrus.Debugf(" Decoded manifest config: %+v\n", manifest.Config)
				desc = mani
				break
			}
		}

		if manifest.Config.MediaType == "" {
			return nil, fmt.Errorf("no matching platform found for %s with platform %+v", name, platform)
		}
	} else {
		// Parse as single-platform manifest
		logrus.Debugf(" Parsing as single-platform manifest\n")
		if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest for %s: %w", name, err)
		}
		logrus.Debugf(" Parsed manifest config: %+v\n", manifest.Config)
	}

	// Fetch the config
	logrus.Debugf(" Fetching config: %s\n", manifest.Config.Digest.String())
	configReader, err := fetcher.Fetch(ctx, manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config for %s: %w", name, err)
	}
	defer func() { _ = configReader.Close() }()

	// Read all bytes first to debug
	configBytes, err := io.ReadAll(configReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for %s: %w", name, err)
	}
	logrus.Debugf(" Config bytes length: %d\n", len(configBytes))

	// Parse the image config
	var imageConfig ocispec.Image
	if err := json.Unmarshal(configBytes, &imageConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config for %s: %w (first 100 bytes: %s)", name, err, string(configBytes[:min(100, len(configBytes))]))
	}

	// Set the platform if not set
	if imageConfig.OS == "" {
		imageConfig.OS = platform.OS
		imageConfig.Architecture = platform.Architecture
		imageConfig.Variant = platform.Variant
		imageConfig.OSVersion = platform.OSVersion
		imageConfig.OSFeatures = platform.OSFeatures
	}

	config := &ImageConfig{
		Ref:      ref,
		Digest:   desc.Digest.String(),
		Config:   &imageConfig,
		Platform: platform,
	}

	// Cache the result
	r.cache.Set(ref, config, nil)

	return config, nil
}

// DefaultPlatform returns the default platform for builds (always Linux)
func DefaultPlatform() ocispec.Platform {
	spec := platforms.DefaultSpec()
	// Force Linux OS since buildkit runs in containers
	spec.OS = "linux"
	return spec
}
