package workspace

import (
	"fmt"
	"sort"
)

type WorkspaceType string

const (
	TypeSingle WorkspaceType = "single"
	TypeMulti  WorkspaceType = "multi"
)

type HealthcheckConfig struct {
	Test     string `yaml:"test"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
	Retries  int    `yaml:"retries"`
}

type Service struct {
	Image       string            `yaml:"image,omitempty"`
	Command     string            `yaml:"command,omitempty"`
	Path        string            `yaml:"path,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Ports       []int             `yaml:"ports,omitempty"`
	PinPort     int               `yaml:"pin_port,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Healthcheck any               `yaml:"healthcheck,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	EnvFile     string            `yaml:"env_file,omitempty"`
	Build       string            `yaml:"build,omitempty"`
	Dockerfile  string            `yaml:"dockerfile,omitempty"`
	BuildFrom   string            `yaml:"build_from,omitempty"`
	ForceBuild     bool   `yaml:"-"`
	ResolvedEnvFile string `yaml:"-"`
}

func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" || s.BuildFrom != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" && s.Build == "" && s.BuildFrom == "" }

type Manifest struct {
	Name     string              `yaml:"name"`
	Type     WorkspaceType       `yaml:"type"`
	Root     string              `yaml:"root,omitempty"`
	Services map[string]Service  `yaml:"services"`
	Groups   map[string][]string `yaml:"groups,omitempty"`
	Profiles map[string][]string `yaml:"profiles,omitempty"`
}

func (m *Manifest) Validate() error {
	for name, svc := range m.Services {
		if svc.BuildFrom == "" {
			continue
		}
		if svc.Build != "" {
			return fmt.Errorf("service %q: build_from is mutually exclusive with build", name)
		}
		if svc.Image != "" {
			return fmt.Errorf("service %q: build_from is mutually exclusive with image", name)
		}
		target, ok := m.Services[svc.BuildFrom]
		if !ok {
			return fmt.Errorf("service %q: build_from references unknown service %q", name, svc.BuildFrom)
		}
		if target.Build == "" {
			return fmt.Errorf("service %q: build_from target %q has no build context", name, svc.BuildFrom)
		}
		if target.BuildFrom != "" {
			return fmt.Errorf("service %q: build_from target %q is itself a build_from (chaining not allowed)", name, svc.BuildFrom)
		}
	}
	return nil
}

type Workspace struct {
	Name     string
	Type     WorkspaceType
	Root     string
	Services map[string]Service
	Groups   map[string][]string
	Profiles map[string][]string
}

func (w Workspace) ServiceNames() []string {
	names := make([]string, 0, len(w.Services))
	for name := range w.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
