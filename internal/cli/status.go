package cli

import (
	"fmt"
	"os"
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
			// Check if we can resolve a workspace name at all.
			// If not (no arg, no rook.yaml in cwd), show all workspaces.
			_, nameErr := cctx.resolveWorkspaceName(args)
			if nameErr != nil && len(args) == 0 {
				return showAllWorkspaces(cctx)
			}
			// Name resolved — use full resolve+load flow (with auto-init prompt)
			ws, err := cctx.resolveAndLoadWorkspace(args, os.Stdin)
			if err != nil {
				return err
			}
			return showWorkspaceDetail(cctx, ws)
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
		ws, err := m.ToWorkspace(e.Path)
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
		// Count running process services
		for name, svc := range ws.Services {
			if !svc.IsProcess() {
				continue
			}
			if processStatus(ws.Root, name) == runner.StatusRunning {
				running++
			}
		}
		status := "stopped"
		if running > 0 && running >= total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		}
		fmt.Printf("%-20s %-12s %d/%d\n", e.Name, status, running, total)
	}
	return nil
}

func showWorkspaceDetail(cctx *cliContext, ws *workspace.Workspace) error {
	prefix := fmt.Sprintf("rook_%s_", ws.Name)
	fmt.Printf("%-20s %-12s %-12s %-8s\n", "SERVICE", "TYPE", "STATUS", "PORT")
	for name, svc := range ws.Services {
		svcType := "process"
		var status string
		if svc.IsContainer() {
			svcType = "container"
			containerName := fmt.Sprintf("%s%s", prefix, name)
			status = string(runner.ContainerStatus(containerName))
		} else {
			status = string(processStatus(ws.Root, name))
		}
		port := "-"
		if result := cctx.portAlloc.Get(ws.Name, name); result.OK {
			port = fmt.Sprintf("%d", result.Port)
		}
		fmt.Printf("%-20s %-12s %-12s %-8s\n", name, svcType, status, port)
	}
	return nil
}

// processStatus checks a process service's status via its PID file.
func processStatus(wsRoot, serviceName string) runner.ServiceStatus {
	pidDir := runner.PIDDirPath(wsRoot)
	info, err := runner.ReadPIDFile(pidDir, serviceName)
	if err != nil {
		return runner.StatusStopped
	}
	if runner.IsProcessAlive(info.PID) {
		return runner.StatusRunning
	}
	// Stale PID file — clean up
	runner.RemovePIDFile(pidDir, serviceName)
	return runner.StatusStopped
}
