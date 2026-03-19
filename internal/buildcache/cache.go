package buildcache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileEntry stores mtime and content hash for a single file.
type FileEntry struct {
	Mtime int64  `json:"mtime"`
	Hash  string `json:"hash"`
}

// ServiceCache stores build metadata for a single service.
type ServiceCache struct {
	ImageID        string               `json:"image_id"`
	DockerfileHash string               `json:"dockerfile_hash"`
	ContextFiles   map[string]FileEntry `json:"context_files"`
}

// Cache stores build metadata for all services in a workspace.
type Cache struct {
	Version  int                     `json:"version"`
	Services map[string]ServiceCache `json:"services"`
}

// Load reads the cache from disk. Returns empty cache if file doesn't exist.
func Load(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Cache{Version: 1, Services: make(map[string]ServiceCache)}, nil
		}
		return nil, fmt.Errorf("reading build cache: %w", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing build cache: %w", err)
	}
	if cache.Services == nil {
		cache.Services = make(map[string]ServiceCache)
	}
	return &cache, nil
}

// Save writes the cache to disk, creating parent directories if needed.
func (c *Cache) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding build cache: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// UpdateAfterBuild refreshes the cache entry for a service after a successful build.
// workDir is the workspace root path.
// buildCtx is the build context directory path (can be relative or absolute).
// dockerfile is the relative path to the Dockerfile from workDir (or "Dockerfile" if default).
// imageID is the Docker image ID of the built image.
func (c *Cache) UpdateAfterBuild(service, workDir, buildCtx, dockerfile, imageID string) error {
	// Hash Dockerfile: explicit path is relative to workDir, default "Dockerfile" is in build context
	var dockerfilePath string
	if dockerfile == "Dockerfile" {
		// Resolve build context first for default Dockerfile lookup
		absBuildCtx := buildCtx
		if !filepath.IsAbs(absBuildCtx) {
			absBuildCtx = filepath.Join(workDir, absBuildCtx)
		}
		dockerfilePath = filepath.Join(absBuildCtx, "Dockerfile")
	} else {
		dockerfilePath = filepath.Join(workDir, dockerfile)
	}
	dockerfileHash, err := HashFile(dockerfilePath)
	if err != nil {
		return fmt.Errorf("hashing Dockerfile: %w", err)
	}

	// Resolve build context to absolute path
	if !filepath.IsAbs(buildCtx) {
		buildCtx = filepath.Join(workDir, buildCtx)
	}

	// Compute Dockerfile path relative to build context for skip comparison
	dockerfileRelToCtx, err := filepath.Rel(buildCtx, dockerfilePath)
	if err != nil {
		return fmt.Errorf("computing Dockerfile relative path: %w", err)
	}

	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return fmt.Errorf("parsing .dockerignore: %w", err)
	}

	// Walk build context
	contextFiles := make(map[string]FileEntry)
	err = filepath.Walk(buildCtx, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip symlinks to directories (Walk uses Lstat, so IsDir is false for dir symlinks)
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Stat(path)
			if err != nil {
				return nil // skip unresolvable symlinks
			}
			if target.IsDir() {
				return nil
			}
		}

		relPath, err := filepath.Rel(buildCtx, path)
		if err != nil {
			return err
		}

		// Skip .dockerignore patterns
		if MatchesPatterns(relPath, ignorePatterns) {
			return nil
		}

		// Skip Dockerfile - it's tracked separately via DockerfileHash
		if relPath == dockerfileRelToCtx {
			return nil
		}

		// Skip .dockerignore - it's metadata, not build content
		if relPath == ".dockerignore" {
			return nil
		}

		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", relPath, err)
		}

		contextFiles[relPath] = FileEntry{
			Mtime: info.ModTime().Unix(),
			Hash:  hash,
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking build context: %w", err)
	}

	c.Services[service] = ServiceCache{
		ImageID:        imageID,
		DockerfileHash: dockerfileHash,
		ContextFiles:   contextFiles,
	}

	return nil
}
