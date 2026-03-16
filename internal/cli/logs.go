package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/andybarilla/rook/internal/runner"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs [workspace] [service]",
		Short: "Tail logs (all or specific service)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			if len(args) > 1 {
				containerName := fmt.Sprintf("rook_%s_%s", wsName, args[1])
				return streamSingleContainer(containerName)
			}

			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return err
			}
			if len(containers) == 0 {
				fmt.Printf("No running containers found for %s.\n", wsName)
				return nil
			}

			mux := newLogMux(os.Stdout)
			var wg sync.WaitGroup

			for i, containerName := range containers {
				svcName := strings.TrimPrefix(containerName, prefix)
				idx := i
				cName := containerName

				logCmd := exec.Command(runner.ContainerRuntime, "logs", "-f", "--follow", cName)
				stdout, err := logCmd.StdoutPipe()
				if err != nil {
					continue
				}
				logCmd.Stderr = logCmd.Stdout
				if err := logCmd.Start(); err != nil {
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					mux.addStream(svcName, stdout, idx)
					logCmd.Wait()
				}()
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			fmt.Println("\nStopped tailing logs.")
			return nil
		},
	}
}

func streamSingleContainer(containerName string) error {
	cmd := exec.Command(runner.ContainerRuntime, "logs", "-f", "--follow", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("tailing logs for %s: %w", containerName, err)
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cmd.Process.Kill()
	return nil
}
