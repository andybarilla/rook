package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/andybarilla/flock/internal/registry"
	"github.com/spf13/cobra"
)

func RenderSiteList(w io.Writer, sites []registry.Site, asJSON bool) {
	if asJSON {
		if sites == nil {
			sites = []registry.Site{}
		}
		FormatJSON(w, sites)
		return
	}

	if len(sites) == 0 {
		fmt.Fprintln(w, "No sites registered.")
		return
	}

	headers := []string{"DOMAIN", "PATH", "PHP", "NODE", "TLS"}
	rows := make([][]string, len(sites))
	for i, s := range sites {
		tlsStr := ""
		if s.TLS {
			tlsStr = "✓"
		}
		rows[i] = []string{s.Domain, s.Path, s.PHPVersion, s.NodeVersion, tlsStr}
	}
	FormatTable(w, headers, rows)
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered sites",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			useJSON := jsonOutput || !IsTTY()
			RenderSiteList(os.Stdout, c.Sites(), useJSON)
			return nil
		},
	}
}

func newAddCmd() *cobra.Command {
	var domain, phpVersion, nodeVersion string
	var tls bool

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a new site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			if domain == "" {
				domain = registry.InferDomain(path)
			}

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			site := registry.Site{
				Path:        path,
				Domain:      domain,
				PHPVersion:  phpVersion,
				NodeVersion: nodeVersion,
				TLS:         tls,
			}

			if err := c.AddSite(site); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, site)
			} else {
				fmt.Fprintf(os.Stdout, "✓ Site %q added (path: %s)\n", domain, path)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&domain, "domain", "", "Domain name (default: inferred from path)")
	cmd.Flags().StringVar(&phpVersion, "php", "", "PHP version")
	cmd.Flags().StringVar(&nodeVersion, "node", "", "Node version")
	cmd.Flags().BoolVar(&tls, "tls", false, "Enable TLS")

	return cmd
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <domain>",
		Short: "Remove a registered site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			c, cleanup, err := NewCore()
			if err != nil {
				return err
			}
			defer cleanup()

			if err := c.RemoveSite(domain); err != nil {
				return err
			}

			useJSON := jsonOutput || !IsTTY()
			if useJSON {
				FormatJSON(os.Stdout, map[string]string{"removed": domain})
			} else {
				fmt.Fprintf(os.Stdout, "✓ Site %q removed\n", domain)
			}
			return nil
		},
	}
}
