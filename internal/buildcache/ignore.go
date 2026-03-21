package buildcache

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/patternmatcher"
)

// Default exclusion patterns (always applied, even without .dockerignore).
var defaultExclusions = []string{".rook/", ".git/"}

// ParseDockerignore reads .dockerignore from the given directory.
// Returns default exclusions if file doesn't exist.
func ParseDockerignore(dir string) ([]string, error) {
	path := filepath.Join(dir, ".dockerignore")
	patterns := append([]string{}, defaultExclusions...)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, fmt.Errorf("reading .dockerignore: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading .dockerignore: %w", err)
	}
	return patterns, nil
}

// ParseGitignore reads .gitignore from the given directory.
// Returns empty slice if file doesn't exist (unlike ParseDockerignore,
// does not include default exclusions — those are added by CollectIgnorePatterns).
func ParseGitignore(dir string) ([]string, error) {
	path := filepath.Join(dir, ".gitignore")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading .gitignore: %w", err)
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading .gitignore: %w", err)
	}
	return patterns, nil
}

// MatchesPatterns checks if a file path matches any of the patterns.
// Supports negation patterns (those starting with !).
// Patterns are normalized to match Docker's .dockerignore behavior:
// patterns without "/" are treated as matching anywhere in the tree.
func MatchesPatterns(path string, patterns []string) bool {
	// Convert path to forward slashes for consistent matching
	path = filepath.ToSlash(path)

	// Normalize patterns to match Docker's .dockerignore behavior
	normalized := make([]string, len(patterns))
	for i, p := range patterns {
		normalized[i] = normalizePattern(p)
	}

	matcher, err := patternmatcher.New(normalized)
	if err != nil {
		// If we can't parse patterns, be conservative and don't exclude
		return false
	}
	matches, _ := matcher.MatchesOrParentMatches(path)
	return matches
}

// normalizePattern prepends **/ to patterns that don't contain a path separator,
// mimicking Docker's .dockerignore behavior where patterns match anywhere in the tree.
func normalizePattern(pattern string) string {
	// Handle negation patterns
	if strings.HasPrefix(pattern, "!") {
		return "!" + normalizePattern(pattern[1:])
	}
	// Patterns with path separators are used as-is
	if strings.Contains(pattern, "/") {
		return pattern
	}
	// Patterns without separators match anywhere
	return "**/" + pattern
}
