package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andybarilla/rook/internal/envgen"
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
	EnvFile     any      `yaml:"env_file"`
}

var composeFileNames = []string{
	".devcontainer/docker-compose.yml",
	".devcontainer/docker-compose.yaml",
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

	// Compose file dir — paths in the file are relative to this
	composeDir := filepath.Dir(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, err
	}

	sourceName := "docker-compose"
	if strings.Contains(path, ".devcontainer") {
		sourceName = "devcontainer-compose"
	}

	result := &DiscoveryResult{
		Source:   sourceName,
		Services: make(map[string]workspace.Service),
	}

	for name, cs := range cf.Services {
		svc := workspace.Service{
			Image: cs.Image,
		}

		// Resolve volume paths relative to compose file directory
		for _, v := range cs.Volumes {
			parts := strings.SplitN(v, ":", 2)
			if len(parts) == 2 && (strings.HasPrefix(parts[0], "./") || strings.HasPrefix(parts[0], "..")) {
				// Resolve relative path, then make it relative to project root
				absPath := filepath.Join(composeDir, parts[0])
				relPath, err := filepath.Rel(dir, absPath)
				if err == nil {
					svc.Volumes = append(svc.Volumes, "./"+relPath+":"+parts[1])
					continue
				}
			}
			svc.Volumes = append(svc.Volumes, v)
		}

		for _, p := range cs.Ports {
			// Expand shell vars like ${POSTGRES_PORT:-5432}
			p = envgen.ExpandShellVars(p)
			parts := strings.Split(p, ":")
			portStr := strings.Split(parts[len(parts)-1], "/")[0]
			if port, err := strconv.Atoi(portStr); err == nil {
				svc.Ports = append(svc.Ports, port)
			}
		}

		// Infer default ports for well-known images when no ports are defined
		if len(svc.Ports) == 0 && svc.Image != "" {
			if port := defaultPortForImage(svc.Image); port > 0 {
				svc.Ports = append(svc.Ports, port)
			}
		}

		// Extract build context (string or object form)
		if cs.Build != nil {
			switch v := cs.Build.(type) {
			case string:
				// Resolve relative to compose file dir, then make relative to project root
				absCtx := filepath.Join(composeDir, v)
				relCtx, err := filepath.Rel(dir, absCtx)
				if err == nil {
					svc.Build = relCtx
				} else {
					svc.Build = v
				}
			case map[string]any:
				// Object form: { context: .., dockerfile: .devcontainer/Dockerfile }
				if ctx, ok := v["context"].(string); ok {
					absCtx := filepath.Join(composeDir, ctx)
					relCtx, err := filepath.Rel(dir, absCtx)
					if err == nil {
						svc.Build = relCtx
					} else {
						svc.Build = ctx
					}
				}
				if df, ok := v["dockerfile"].(string); ok {
					// Dockerfile path is relative to the build context, not the compose dir.
					// Since we've already resolved build context to be relative to project root,
					// resolve dockerfile relative to the build context.
					if svc.Build != "" {
						svc.Dockerfile = filepath.Join(svc.Build, df)
					} else {
						svc.Dockerfile = df
					}
				}
			}
		}

		// Extract command
		if cs.Command != nil {
			switch v := cs.Command.(type) {
			case string:
				svc.Command = v
			case []any:
				parts := make([]string, len(v))
				for i, p := range v {
					parts[i] = fmt.Sprintf("%v", p)
				}
				svc.Command = strings.Join(parts, " ")
			}
		}

		// Extract env_file (simple string form only), resolve relative to compose dir
		if cs.EnvFile != nil {
			if envStr, ok := cs.EnvFile.(string); ok {
				absEnv := filepath.Join(composeDir, envStr)
				relEnv, err := filepath.Rel(dir, absEnv)
				if err == nil {
					svc.EnvFile = relEnv
				} else {
					svc.EnvFile = envStr
				}
			}
		}

		svc.Environment = parseEnvironment(cs.Environment)
		svc.DependsOn = parseDependsOn(cs.DependsOn)
		result.Services[name] = svc
	}

	// When using devcontainer compose, merge depends_on from root compose
	// (devcontainer compose typically omits dependency declarations)
	if strings.Contains(path, ".devcontainer") {
		mergeRootDependsOn(dir, result)
	}

	return result, nil
}

// mergeRootDependsOn looks for a root docker-compose.yml and merges any
// depends_on declarations into the discovered services. This is needed because
// devcontainer compose files typically omit depends_on since the devcontainer
// runtime handles service ordering.
func mergeRootDependsOn(dir string, result *DiscoveryResult) {
	rootNames := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	var rootPath string
	for _, name := range rootNames {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			rootPath = p
			break
		}
	}
	if rootPath == "" {
		return
	}

	data, err := os.ReadFile(rootPath)
	if err != nil {
		return
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return
	}

	// Direct name match: merge depends_on for services with the same name
	for name, cs := range cf.Services {
		svc, exists := result.Services[name]
		if !exists || len(svc.DependsOn) > 0 {
			continue
		}
		deps := parseDependsOn(cs.DependsOn)
		if len(deps) > 0 {
			svc.DependsOn = deps
			result.Services[name] = svc
		}
	}

	// Primary service match: devcontainer typically names the build service "app"
	// while root compose uses the actual name (e.g., "api"). If the devcontainer
	// has a build service with no depends_on, find the root's build service and
	// carry over its dependencies.
	for devName, devSvc := range result.Services {
		if devSvc.Build == "" || len(devSvc.DependsOn) > 0 {
			continue
		}
		// Find a root build service with depends_on (skip if names match — already handled)
		for rootName, rootCs := range cf.Services {
			if rootName == devName {
				continue
			}
			// Check if this root service has a build context
			hasBuild := false
			if rootCs.Build != nil {
				switch rootCs.Build.(type) {
				case string:
					hasBuild = true
				case map[string]any:
					hasBuild = true
				}
			}
			if !hasBuild {
				continue
			}
			deps := parseDependsOn(rootCs.DependsOn)
			if len(deps) == 0 {
				continue
			}
			// Filter to only deps that exist in the devcontainer result
			var validDeps []string
			for _, dep := range deps {
				if _, exists := result.Services[dep]; exists {
					validDeps = append(validDeps, dep)
				}
			}
			if len(validDeps) > 0 {
				devSvc.DependsOn = validDeps
				result.Services[devName] = devSvc
				break // only match one root build service
			}
		}
	}
}

func parseEnvironment(env any) map[string]string {
	if env == nil {
		return nil
	}
	result := make(map[string]string)
	switch v := env.(type) {
	case map[string]any:
		for k, val := range v {
			result[k] = envgen.ExpandShellVars(fmt.Sprintf("%v", val))
		}
	case []any:
		for _, item := range v {
			parts := strings.SplitN(fmt.Sprintf("%v", item), "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = envgen.ExpandShellVars(parts[1])
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

// defaultPortForImage returns the well-known default port for common Docker images.
// Returns 0 if the image is not recognized.
func defaultPortForImage(image string) int {
	// Strip tag (e.g., "postgres:16-alpine" → "postgres")
	name := strings.Split(image, ":")[0]
	// Strip registry prefix (e.g., "docker.io/library/postgres" → "postgres")
	parts := strings.Split(name, "/")
	name = parts[len(parts)-1]

	defaults := map[string]int{
		"postgres":  5432,
		"pgvector":  5432,
		"mysql":     3306,
		"mariadb":   3306,
		"redis":     6379,
		"mongo":     27017,
		"rabbitmq":  5672,
		"nats":      4222,
		"memcached": 11211,
		"minio":     9000,
		"nginx":     80,
		"caddy":     80,
		"httpd":     80,
		"traefik":   80,
	}
	return defaults[name]
}
