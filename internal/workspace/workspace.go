package workspace

import "sort"

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
	Build       string            `yaml:"build,omitempty"`
	ForceBuild  bool              `yaml:"-"`
}

func (s Service) IsContainer() bool { return s.Image != "" || s.Build != "" }
func (s Service) IsProcess() bool   { return s.Command != "" && s.Image == "" && s.Build == "" }

type Manifest struct {
	Name     string              `yaml:"name"`
	Type     WorkspaceType       `yaml:"type"`
	Root     string              `yaml:"root,omitempty"`
	Services map[string]Service  `yaml:"services"`
	Groups   map[string][]string `yaml:"groups,omitempty"`
	Profiles map[string][]string `yaml:"profiles,omitempty"`
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
