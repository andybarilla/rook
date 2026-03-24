package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/envgen"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func newEnvRewriteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rewrite <var> <service> [workspace]",
		Short: "Rewrite an env var with rook template tags",
		Long:  "Reads a variable from a service's .env file, replaces host/port with {{.Host.x}}/{{.Port.x}} template tags, and adds it to rook.yaml's environment block.",
		Args:  cobra.RangeArgs(2, 3),
		RunE:  runEnvRewrite,
	}
}

func runEnvRewrite(cmd *cobra.Command, args []string) error {
	varName := args[0]
	serviceName := args[1]
	wsArgs := args[2:]

	cctx, err := newCLIContext()
	if err != nil {
		return err
	}

	ws, err := cctx.resolveAndLoadWorkspace(wsArgs, os.Stdin)
	if err != nil {
		return err
	}

	if _, ok := ws.Services[serviceName]; !ok {
		return fmt.Errorf("service %q not found in workspace %q", serviceName, ws.Name)
	}

	type envFileMatch struct {
		svcName string
		value   string
	}
	var matches []envFileMatch
	hasEnvFile := false

	for name, svc := range ws.Services {
		if svc.EnvFile == "" {
			continue
		}
		hasEnvFile = true
		envPath := filepath.Join(ws.Root, svc.EnvFile)
		vars, err := envgen.ParseEnvFile(envPath)
		if err != nil {
			return fmt.Errorf("parsing env file for %s: %w", name, err)
		}
		if val, ok := vars[varName]; ok {
			matches = append(matches, envFileMatch{svcName: name, value: val})
		}
	}

	if !hasEnvFile {
		return fmt.Errorf("no services in workspace %q have an env_file", ws.Name)
	}

	if len(matches) == 0 {
		return fmt.Errorf("%q not found in any service's env_file", varName)
	}

	entry, regErr := cctx.registry.Get(ws.Name)
	if regErr != nil {
		return regErr
	}
	manifestPath := filepath.Join(entry.Path, "rook.yaml")
	manifest, err := workspace.ParseManifest(manifestPath)
	if err != nil {
		return err
	}

	for _, m := range matches {
		rewritten, err := envgen.Rewrite(m.value, serviceName)
		if err != nil {
			return err
		}
		svc := manifest.Services[m.svcName]
		if svc.Environment == nil {
			svc.Environment = make(map[string]string)
		}
		svc.Environment[varName] = rewritten
		manifest.Services[m.svcName] = svc
		fmt.Printf("%s: %s = %s\n", m.svcName, varName, rewritten)
	}

	if err := workspace.WriteManifest(manifestPath, manifest); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}
