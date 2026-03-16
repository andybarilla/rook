package cli

import (
	"fmt"
	"path/filepath"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [workspace]",
		Short: "Show all workspaces and running services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return showAllWorkspaces(cctx)
			}
			return showWorkspaceDetail(cctx, args[0])
		},
	}
}

func showAllWorkspaces(cctx *cliContext) error {
	entries := cctx.registry.List()
	if len(entries) == 0 {
		fmt.Println("No workspaces registered.")
		return nil
	}
	fmt.Printf("%-20s %-12s %-12s\n", "WORKSPACE", "STATUS", "SERVICES")
	for _, e := range entries {
		m, err := workspace.ParseManifest(filepath.Join(e.Path, "rook.yaml"))
		if err != nil {
			fmt.Printf("%-20s %-12s %-12s\n", e.Name, "error", "-")
			continue
		}
		total := len(m.Services)
		prefix := fmt.Sprintf("rook_%s_", e.Name)
		containers, _ := runner.FindContainers(prefix)
		running := 0
		for _, c := range containers {
			if runner.ContainerStatus(c) == runner.StatusRunning {
				running++
			}
		}
		hasProcessOnly := true
		for _, svc := range m.Services {
			if svc.IsContainer() {
				hasProcessOnly = false
				break
			}
		}
		status := "stopped"
		if running > 0 && running >= total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		} else if hasProcessOnly && len(containers) == 0 {
			status = "unknown"
		}
		fmt.Printf("%-20s %-12s %d/%d\n", e.Name, status, running, total)
	}
	return nil
}

func showWorkspaceDetail(cctx *cliContext, wsName string) error {
	ws, err := cctx.loadWorkspace(wsName)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("rook_%s_", wsName)
	fmt.Printf("%-20s %-12s %-12s %-8s\n", "SERVICE", "TYPE", "STATUS", "PORT")
	for name, svc := range ws.Services {
		svcType := "process"
		status := "unknown"
		if svc.IsContainer() {
			svcType = "container"
			containerName := fmt.Sprintf("%s%s", prefix, name)
			status = string(runner.ContainerStatus(containerName))
		}
		port := "-"
		if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
			port = fmt.Sprintf("%d", result.Port)
		}
		fmt.Printf("%-20s %-12s %-12s %-8s\n", name, svcType, status, port)
	}
	return nil
}
