package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/andybarilla/rook/internal/buildcache"
	"github.com/andybarilla/rook/internal/runner"
	"github.com/andybarilla/rook/internal/workspace"
	"github.com/spf13/cobra"
)

func NewCheckBuildsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check-builds [workspace]",
		Short: "Check which services need rebuilding",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			cachePath := filepath.Join(ws.Root, ".rook", "build-cache.json")
			cache, err := buildcache.Load(cachePath)
			if err != nil {
				return fmt.Errorf("loading build cache: %w", err)
			}

			results := make(map[string]buildcache.StaleResult)
			hasStale := false

			docker := runner.NewDockerRunner(fmt.Sprintf("rook_%s", wsName))

			for name, svc := range ws.Services {
				if svc.Build == "" {
					results[name] = buildcache.StaleResult{}
					continue
				}

				// Get current image ID (optional - may not exist)
				currentImageID, _ := docker.GetImageID(name)

				result, err := buildcache.DetectStale(cache, name, svc, ws.Root, currentImageID)
				if err != nil {
					return fmt.Errorf("checking %s: %w", name, err)
				}
				results[name] = *result
				if result.NeedsRebuild {
					hasStale = true
				}
			}

			if jsonOutput {
				return printCheckBuildsJSON(results, ws.Services, hasStale)
			}
			return printCheckBuildsText(results, ws.Services, hasStale)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func printCheckBuildsText(results map[string]buildcache.StaleResult, services map[string]workspace.Service, hasStale bool) error {
	for name, svc := range services {
		result := results[name]
		if svc.Build == "" {
			fmt.Printf("%s: no build context (uses image)\n", name)
		} else if result.NeedsRebuild {
			if len(result.Reasons) > 0 {
				fmt.Printf("%s: needs rebuild (%s)\n", name, result.Reasons[0])
			} else {
				fmt.Printf("%s: needs rebuild\n", name)
			}
		} else {
			fmt.Printf("%s: up to date\n", name)
		}
	}

	if hasStale {
		os.Exit(1)
	}
	return nil
}

type checkBuildsJSONOutput struct {
	Services map[string]checkBuildsServiceStatus `json:"services"`
}

type checkBuildsServiceStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func printCheckBuildsJSON(results map[string]buildcache.StaleResult, services map[string]workspace.Service, hasStale bool) error {
	output := checkBuildsJSONOutput{Services: make(map[string]checkBuildsServiceStatus)}

	for name, svc := range services {
		result := results[name]
		status := checkBuildsServiceStatus{}

		if svc.Build == "" {
			status.Status = "no_build_context"
		} else if result.NeedsRebuild {
			status.Status = "needs_rebuild"
			if len(result.Reasons) > 0 {
				status.Reason = result.Reasons[0]
			}
		} else {
			status.Status = "up_to_date"
		}

		output.Services[name] = status
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return err
	}

	if hasStale {
		os.Exit(1)
	}
	return nil
}
