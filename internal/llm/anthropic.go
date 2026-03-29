package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	defaultModel    = "claude-haiku-4-5-20251001"
	defaultEndpoint = "https://api.anthropic.com/v1/messages"
	maxTokens       = 4096
)

// AnthropicProvider calls the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey   string
	model    string
	endpoint string
	client   *http.Client
}

// NewAnthropicProvider creates a provider using the ANTHROPIC_API_KEY env var.
// Returns an error if the key is not set.
func NewAnthropicProvider() (*AnthropicProvider, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set — LLM features require an API key")
	}
	return &AnthropicProvider{
		apiKey:   key,
		model:    defaultModel,
		endpoint: defaultEndpoint,
		client:   http.DefaultClient,
	}, nil
}

// newAnthropicProviderWithEndpoint creates a provider pointing at a custom endpoint (for testing).
func newAnthropicProviderWithEndpoint(apiKey, endpoint string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:   apiKey,
		model:    defaultModel,
		endpoint: endpoint,
		client:   http.DefaultClient,
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends a request to the Anthropic API and returns the response.
func (p *AnthropicProvider) Complete(ctx context.Context, req Request) (Response, error) {
	body := anthropicRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    req.System,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.Prompt},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return Response{}, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return Response{}, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return Response{}, fmt.Errorf("parsing response: %w", err)
	}

	if apiResp.Error != nil {
		return Response{}, fmt.Errorf("API error: %s: %s", apiResp.Error.Type, apiResp.Error.Message)
	}

	for _, block := range apiResp.Content {
		if block.Type == "text" {
			return Response{Content: block.Text}, nil
		}
	}

	return Response{}, fmt.Errorf("no text content in response")
}
