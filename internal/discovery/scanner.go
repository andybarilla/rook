package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFileInfo summarizes a compose file found during scanning.
type ComposeFileInfo struct {
	Path         string   // absolute path
	RelPath      string   // relative to project dir
	ServiceNames []string // service names found in the file
}

// scanCandidates are the fixed filenames to check (relative to project root).
var scanCandidates = []string{
	".devcontainer/docker-compose.yml",
	".devcontainer/docker-compose.yaml",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// ScanComposeFiles finds all compose files in dir and returns a summary of each.
// Results are sorted: .devcontainer/ files first, then root files alphabetically.
func ScanComposeFiles(dir string) []ComposeFileInfo {
	seen := make(map[string]bool)
	var results []ComposeFileInfo

	// Check fixed candidates
	for _, rel := range scanCandidates {
		abs := filepath.Join(dir, rel)
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		info := parseComposeFileInfo(abs, rel)
		if info != nil {
			results = append(results, *info)
			seen[rel] = true
		}
	}

	// Glob for docker-compose.*.yml and docker-compose.*.yaml variants
	for _, pattern := range []string{"docker-compose.*.yml", "docker-compose.*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		for _, abs := range matches {
			rel, err := filepath.Rel(dir, abs)
			if err != nil {
				continue
			}
			if seen[rel] {
				continue
			}
			info := parseComposeFileInfo(abs, rel)
			if info != nil {
				results = append(results, *info)
				seen[rel] = true
			}
		}
	}

	// Sort: .devcontainer/ first, then alphabetically by RelPath
	sort.Slice(results, func(i, j int) bool {
		iDev := strings.HasPrefix(results[i].RelPath, ".devcontainer/")
		jDev := strings.HasPrefix(results[j].RelPath, ".devcontainer/")
		if iDev != jDev {
			return iDev
		}
		return results[i].RelPath < results[j].RelPath
	})

	return results
}

// parseComposeFileInfo reads a compose file and extracts service names.
// Returns nil if the file can't be parsed.
func parseComposeFileInfo(absPath, relPath string) *ComposeFileInfo {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	var cf struct {
		Services map[string]any `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil
	}

	names := make([]string, 0, len(cf.Services))
	for name := range cf.Services {
		names = append(names, name)
	}
	sort.Strings(names)

	return &ComposeFileInfo{
		Path:         absPath,
		RelPath:      relPath,
		ServiceNames: names,
	}
}
