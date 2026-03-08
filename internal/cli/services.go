package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/andybarilla/flock/internal/databases"
	"github.com/spf13/cobra"
)

func RenderServiceStatus(w io.Writer, services []databases.ServiceInfo, asJSON bool) {
	if asJSON {
		if services == nil {
			services = []databases.ServiceInfo{}
		}
		FormatJSON(w, services)
		return
	}

	if len(services) == 0 {
		fmt.Fprintln(w, "No services configured.")
		return
	}

	headers := []string{"SERVICE", "STATUS", "PORT"}
	rows := make([][]string, len(services))
	for i, s := range services {
		status := "stopped"
		if s.Running {
			status = "running"
		}
		port := ""
		if s.Enabled {
			port = fmt.Sprintf("%d", s.Port)
		}
		rows[i] = []string{string(s.Type), status, port}
	}
	FormatTable(w, headers, rows)
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show status of all database services",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			useJSON := jsonOutput || !IsTTY()
			RenderServiceStatus(os.Stdout, c.DatabaseServices(), useJSON)
			return nil
		},
	}
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <service>",
		Short: "Start a database service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.StartDatabase(svc); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"started": svc})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Service %q started\n", svc)
			}
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <service>",
		Short: "Stop a database service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.StopDatabase(svc); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"stopped": svc})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Service %q stopped\n", svc)
			}
			return nil
		},
	}
}
