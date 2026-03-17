package api

import (
	"github.com/andybarilla/rook/internal/ports"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
)

// ServiceInfo is a summary of a single service for list views.
type ServiceInfo struct {
	Name      string               `json:"name"`
	Image     string               `json:"image,omitempty"`
	Command   string               `json:"command,omitempty"`
	Status    runner.ServiceStatus `json:"status"`
	Port      int                  `json:"port,omitempty"`
	DependsOn []string             `json:"dependsOn,omitempty"`
}

// WorkspaceInfo is a summary of a workspace for list views.
type WorkspaceInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	ServiceCount  int    `json:"serviceCount"`
	RunningCount  int    `json:"runningCount"`
	ActiveProfile string `json:"activeProfile,omitempty"`
}

// WorkspaceDetail is the full detail of a workspace including services.
type WorkspaceDetail struct {
	Name          string              `json:"name"`
	Path          string              `json:"path"`
	Services      []ServiceInfo       `json:"services"`
	Profiles      map[string][]string `json:"profiles,omitempty"`
	Groups        map[string][]string `json:"groups,omitempty"`
	ActiveProfile string              `json:"activeProfile,omitempty"`
}

// StatusEvent is emitted when a service status changes.
type StatusEvent struct {
	Workspace string               `json:"workspace"`
	Service   string               `json:"service"`
	Status    runner.ServiceStatus `json:"status"`
}

// LogEvent is emitted when a new log line is received.
type LogEvent struct {
	Workspace string `json:"workspace"`
	Service   string `json:"service"`
	Line      string `json:"line"`
}

// WorkspaceChangedEvent is emitted when the workspace list or detail changes.
type WorkspaceChangedEvent struct {
	Workspace string `json:"workspace"`
}

// DiscoverResult wraps discovery output for the API layer.
type DiscoverResult struct {
	Source   string                       `json:"source"`
	Services map[string]workspace.Service `json:"services"`
	Groups   map[string][]string          `json:"groups,omitempty"`
}

// EnvVar represents a single environment variable with its template and resolved value.
type EnvVar struct {
	Key      string `json:"key"`
	Template string `json:"template"`
	Resolved string `json:"resolved"`
}

// PortEntry is a type alias for the ports package PortEntry.
type PortEntry = ports.PortEntry

// Manifest is a type alias for the workspace package Manifest.
type Manifest = workspace.Manifest

// Settings holds user preferences for the GUI.
type Settings struct {
	AutoRebuild bool `json:"autoRebuild"`
}

// BuildStatus describes the build state of a single service.
type BuildStatus struct {
	Name     string   `json:"name"`
	HasBuild bool     `json:"hasBuild"`
	Status   string   `json:"status"` // "up_to_date", "needs_rebuild", "no_build_context"
	Reasons  []string `json:"reasons,omitempty"`
}

// BuildCheckResult contains build status for all services in a workspace.
type BuildCheckResult struct {
	Services []BuildStatus `json:"services"`
	HasStale bool          `json:"hasStale"`
}
