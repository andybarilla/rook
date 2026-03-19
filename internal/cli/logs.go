package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cctx, err := newCLIContext()
			if err != nil {
				return err
			}
			wsName, err := cctx.resolveWorkspaceName(args)
			if err != nil {
				return err
			}

			ws, err := cctx.loadWorkspace(wsName)
			if err != nil {
				return err
			}

			// Single-service mode
			if len(args) > 1 {
				svcName := args[1]
				if svc, ok := ws.Services[svcName]; ok && svc.IsProcess() {
					logPath := filepath.Join(logDirPath(ws.Root), svcName+".log")
					return streamSingleProcessLog(logPath, ctx)
				}
				containerName := fmt.Sprintf("rook_%s_%s", wsName, svcName)
				return streamSingleContainer(containerName)
			}

			// Multi-service mode
			prefix := fmt.Sprintf("rook_%s_", wsName)
			containers, err := runner.FindContainers(prefix)
			if err != nil {
				return err
			}

			// Find process services with log files
			type processLog struct {
				name string
				path string
			}
			var processLogs []processLog
			logDir := logDirPath(ws.Root)
			for name, svc := range ws.Services {
				if !svc.IsProcess() {
					continue
				}
				logPath := filepath.Join(logDir, name+".log")
				if _, err := os.Stat(logPath); err == nil {
					processLogs = append(processLogs, processLog{name: name, path: logPath})
				}
			}

			if len(containers) == 0 && len(processLogs) == 0 {
				fmt.Printf("No running services found for %s.\n", wsName)
				return nil
			}

			mux := newLogMux(os.Stdout)
			var wg sync.WaitGroup
			colorIdx := 0

			// Stream container logs
			for _, containerName := range containers {
				svcName := strings.TrimPrefix(containerName, prefix)
				idx := colorIdx
				colorIdx++
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

			// Stream process log files
			for _, pl := range processLogs {
				idx := colorIdx
				colorIdx++
				name := pl.name

				reader, err := tailFile(ctx, pl.path)
				if err != nil {
					continue
				}

				wg.Add(1)
				go func() {
					defer wg.Done()
					mux.addStream(name, reader, idx)
				}()
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh

			cancel()
			wg.Wait()
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

func streamSingleProcessLog(logPath string, ctx context.Context) error {
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("no log file found at %s", logPath)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	reader, err := tailFile(ctx, logPath)
	if err != nil {
		return fmt.Errorf("tailing log file: %w", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cancel()
	<-done
	return nil
}
