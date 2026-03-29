package llm

import "context"

// Provider sends prompts to an LLM and returns responses.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Request is a prompt to send to the LLM.
type Request struct {
	System string
	Prompt string
}

// Response is the LLM's reply.
type Response struct {
	Content string
}
