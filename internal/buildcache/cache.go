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
