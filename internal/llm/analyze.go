package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ServiceSuggestion is a single service suggested by LLM analysis.
type ServiceSuggestion struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"` // "container" or "process"
	Image     string   `json:"image,omitempty"`
	Command   string   `json:"command,omitempty"`
	Ports     []int    `json:"ports,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Reasoning string   `json:"reasoning"`
}

const systemPrompt = `You are analyzing a software project to help configure a local development workspace manager called rook.

Rook manages two types of services:
- "container" services: run as Docker/Podman containers (have an "image" field like "postgres:16")
- "process" services: run as local processes (have a "command" field like "go run ./cmd/api")

Infrastructure (databases, caches, message queues) should be container services.
Application code the developer is actively working on should be process services.

Respond with ONLY a JSON array of service suggestions. No markdown, no explanation outside the JSON.`

// AnalyzeDockerfile asks the LLM to analyze a Dockerfile and suggest services.
func AnalyzeDockerfile(ctx context.Context, p Provider, dockerfile string, startScript string, fileTree string) ([]ServiceSuggestion, error) {
	var prompt strings.Builder
	prompt.WriteString("Analyze this Dockerfile and suggest what rook services it represents.\n\n")
	prompt.WriteString("## Dockerfile\n```\n")
	prompt.WriteString(dockerfile)
	prompt.WriteString("\n```\n\n")

	if startScript != "" {
		prompt.WriteString("## Start script\n```bash\n")
		prompt.WriteString(startScript)
		prompt.WriteString("\n```\n\n")
	}

	if fileTree != "" {
		prompt.WriteString("## Repository file tree (top 2 levels)\n```\n")
		prompt.WriteString(fileTree)
		prompt.WriteString("\n```\n\n")
	}

	prompt.WriteString("Return a JSON array of ServiceSuggestion objects with fields: name, type (\"container\" or \"process\"), image (for containers), command (for processes), ports (array of ints), depends_on (array of service names), reasoning.")

	return complete(ctx, p, prompt.String())
}

// AnalyzeRepo asks the LLM to suggest local services based on repo structure.
func AnalyzeRepo(ctx context.Context, p Provider, fileTree string, configFiles map[string]string) ([]ServiceSuggestion, error) {
	var prompt strings.Builder
	prompt.WriteString("Analyze this repository and suggest what local development services it needs.\n\n")

	if fileTree != "" {
		prompt.WriteString("## Repository file tree (top 2 levels)\n```\n")
		prompt.WriteString(fileTree)
		prompt.WriteString("\n```\n\n")
	}

	for name, content := range configFiles {
		prompt.WriteString(fmt.Sprintf("## %s\n```\n%s\n```\n\n", name, content))
	}

	prompt.WriteString("Return a JSON array of ServiceSuggestion objects with fields: name, type (\"container\" or \"process\"), image (for containers), command (for processes), ports (array of ints), depends_on (array of service names), reasoning.")

	return complete(ctx, p, prompt.String())
}

func complete(ctx context.Context, p Provider, userPrompt string) ([]ServiceSuggestion, error) {
	resp, err := p.Complete(ctx, Request{
		System: systemPrompt,
		Prompt: userPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	return parseSuggestions(resp.Content)
}

// parseSuggestions extracts a JSON array of ServiceSuggestion from LLM output.
// Handles cases where the JSON is wrapped in markdown code fences.
func parseSuggestions(content string) ([]ServiceSuggestion, error) {
	content = strings.TrimSpace(content)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		// Remove first and last lines (fences)
		if len(lines) >= 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
			content = strings.TrimSpace(content)
		}
	}

	var suggestions []ServiceSuggestion
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w\nRaw response:\n%s", err, content)
	}
	return suggestions, nil
}
