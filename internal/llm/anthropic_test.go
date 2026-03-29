package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicProvider_Complete(t *testing.T) {
	t.Run("sends_correct_request_and_parses_response", func(t *testing.T) {
		var receivedReq anthropicRequest

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-api-key") != "test-key" {
				t.Error("expected x-api-key header")
			}
			if r.Header.Get("anthropic-version") != "2023-06-01" {
				t.Error("expected anthropic-version header")
			}

			json.NewDecoder(r.Body).Decode(&receivedReq)

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(anthropicResponse{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "text", Text: "Hello from the API"},
				},
			})
		}))
		defer server.Close()

		p := newAnthropicProviderWithEndpoint("test-key", server.URL)
		resp, err := p.Complete(context.Background(), Request{
			System: "You are helpful.",
			Prompt: "Say hello.",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.Content != "Hello from the API" {
			t.Errorf("expected 'Hello from the API', got %q", resp.Content)
		}
		if receivedReq.Model != defaultModel {
			t.Errorf("expected model %q, got %q", defaultModel, receivedReq.Model)
		}
		if receivedReq.System != "You are helpful." {
			t.Errorf("expected system prompt, got %q", receivedReq.System)
		}
		if len(receivedReq.Messages) != 1 || receivedReq.Messages[0].Content != "Say hello." {
			t.Errorf("unexpected messages: %v", receivedReq.Messages)
		}
	})

	t.Run("handles_api_error_status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"type":"invalid_request","message":"bad request"}}`))
		}))
		defer server.Close()

		p := newAnthropicProviderWithEndpoint("test-key", server.URL)
		_, err := p.Complete(context.Background(), Request{Prompt: "test"})
		if err == nil {
			t.Fatal("expected error for 400 status")
		}
	})

	t.Run("handles_empty_content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(anthropicResponse{})
		}))
		defer server.Close()

		p := newAnthropicProviderWithEndpoint("test-key", server.URL)
		_, err := p.Complete(context.Background(), Request{Prompt: "test"})
		if err == nil {
			t.Fatal("expected error for empty content")
		}
	})
}

func TestNewAnthropicProvider_RequiresKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := NewAnthropicProvider()
	if err == nil {
		t.Fatal("expected error when ANTHROPIC_API_KEY not set")
	}
}
