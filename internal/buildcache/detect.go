package buildcache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/workspace"
)

// StaleResult describes whether a service needs rebuilding and why.
type StaleResult struct {
	NeedsRebuild bool
	Reasons      []string
}

// DetectStale checks if a service's image is stale relative to its build context.
// workDir is the workspace root path. svc.Build is the build context path relative to workDir.
// currentImageID is the current Docker image ID (optional - if empty, image ID check is skipped).
func DetectStale(cache *Cache, service string, svc workspace.Service, workDir, currentImageID string) (*StaleResult, error) {
	result := &StaleResult{}

	if svc.Build == "" {
		return result, nil // no build context, nothing to check
	}

	buildCtx := filepath.Join(workDir, svc.Build)
	cached, hasCache := cache.Services[service]

	// No cache entry = needs rebuild
	if !hasCache {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "no build cache")
		return result, nil
	}

	// Check if image was deleted (cached exists but current doesn't)
	if currentImageID == "" && cached.ImageID != "" {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "image missing")
	}

	// Check if image was rebuilt externally (IDs differ)
	if currentImageID != "" && cached.ImageID != "" && currentImageID != cached.ImageID {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "image rebuilt externally")
	}

	// Determine Dockerfile path: explicit path is relative to workDir, default "Dockerfile" is in build context
	dockerfile := "Dockerfile"
	if svc.Dockerfile != "" {
		dockerfile = svc.Dockerfile
	}
	var dockerfilePath string
	if svc.Dockerfile != "" {
		dockerfilePath = filepath.Join(workDir, dockerfile)
	} else {
		dockerfilePath = filepath.Join(buildCtx, "Dockerfile")
	}

	// Compute Dockerfile path relative to build context for skip comparison
	dockerfileRelToCtx, err := filepath.Rel(buildCtx, dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("computing Dockerfile relative path: %w", err)
	}

	// Check Dockerfile
	dockerfileHash, err := HashFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("hashing Dockerfile: %w", err)
	}
	if dockerfileHash != cached.DockerfileHash {
		result.NeedsRebuild = true
		result.Reasons = append(result.Reasons, "Dockerfile modified")
	}

	// Parse .dockerignore
	ignorePatterns, err := ParseDockerignore(buildCtx)
	if err != nil {
		return nil, fmt.Errorf("parsing .dockerignore: %w", err)
	}

	// Walk build context
	newFiles := make(map[string]FileEntry)
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

		// Skip Dockerfile - it's checked separately via DockerfileHash
		if relPath == dockerfileRelToCtx {
			return nil
		}

		// Skip .dockerignore - it's metadata, not build content
		if relPath == ".dockerignore" {
			return nil
		}

		// Check mtime first
		cachedEntry, wasCached := cached.ContextFiles[relPath]
		mtime := info.ModTime().Unix()

		if wasCached && cachedEntry.Mtime == mtime {
			// mtime unchanged, file is unchanged
			newFiles[relPath] = cachedEntry
			return nil
		}

		// mtime changed or new file, compute hash
		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("hashing %s: %w", relPath, err)
		}

		newFiles[relPath] = FileEntry{Mtime: mtime, Hash: hash}

		if !wasCached {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s added", relPath))
		} else if hash != cachedEntry.Hash {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s modified", relPath))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking build context: %w", err)
	}

	// Check for deleted files
	for relPath := range cached.ContextFiles {
		if _, exists := newFiles[relPath]; !exists {
			result.NeedsRebuild = true
			result.Reasons = append(result.Reasons, fmt.Sprintf("%s deleted", relPath))
		}
	}

	// Deduplicate and summarize reasons
	result.Reasons = summarizeReasons(result.Reasons)

	return result, nil
}

// summarizeReasons consolidates multiple file changes into a summary.
func summarizeReasons(reasons []string) []string {
	if len(reasons) <= 3 {
		return reasons
	}
	return []string{fmt.Sprintf("%d files changed", len(reasons))}
}
