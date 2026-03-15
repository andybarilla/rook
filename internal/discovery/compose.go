package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andybarilla/rook/internal/workspace"
	"gopkg.in/yaml.v3"
)

type composeFile struct {
	Services map[string]composeService `yaml:"services"`
}

type composeService struct {
	Image       string   `yaml:"image"`
	Build       any      `yaml:"build"`
	Ports       []string `yaml:"ports"`
	Environment any      `yaml:"environment"`
	Volumes     []string `yaml:"volumes"`
	DependsOn   any      `yaml:"depends_on"`
	Command     any      `yaml:"command"`
}

var composeFileNames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// ComposeDiscoverer detects and parses Docker Compose files.
type ComposeDiscoverer struct{}

// NewComposeDiscoverer creates a new ComposeDiscoverer.
func NewComposeDiscoverer() *ComposeDiscoverer { return &ComposeDiscoverer{} }

// Name returns the discoverer name.
func (d *ComposeDiscoverer) Name() string { return "docker-compose" }

// Detect returns true if a Docker Compose file exists in the directory.
func (d *ComposeDiscoverer) Detect(dir string) bool {
	for _, name := range composeFileNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// Discover parses the Docker Compose file and returns discovered services.
func (d *ComposeDiscoverer) Discover(dir string) (*DiscoveryResult, error) {
	var path string
	for _, name := range composeFileNames {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			path = p
			break
		}
	}
	if path == "" {
		return nil, fmt.Errorf("no compose file found")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, err
	}

	result := &DiscoveryResult{
		Source:   "docker-compose",
		Services: make(map[string]workspace.Service),
	}

	for name, cs := range cf.Services {
		svc := workspace.Service{
			Image:   cs.Image,
			Volumes: cs.Volumes,
		}

		for _, p := range cs.Ports {
			parts := strings.Split(p, ":")
			portStr := strings.Split(parts[len(parts)-1], "/")[0]
			if port, err := strconv.Atoi(portStr); err == nil {
				svc.Ports = append(svc.Ports, port)
			}
		}

		svc.Environment = parseEnvironment(cs.Environment)
		svc.DependsOn = parseDependsOn(cs.DependsOn)
		result.Services[name] = svc
	}

	return result, nil
}

func parseEnvironment(env any) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string)
	switch v := env.(type) {
	case map[string]any:
		for k, val := range v {
			result[k] = fmt.Sprintf("%v", val)
		}
	case []any:
		for _, item := range v {
			parts := strings.SplitN(fmt.Sprintf("%v", item), "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseDependsOn(deps any) []string {
	if deps == nil {
		return nil
	}
	switch v := deps.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case map[string]any:
		result := make([]string, 0, len(v))
		for k := range v {
			result = append(result, k)
		}
		return result
	}
	return nil
}
