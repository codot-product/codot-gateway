package proxy

import (
	"context"
	"net/http"
	"github.com/codot-product/codot-gateway/api/openai"
)

// ProviderClient abstracts away vendor-specific schema anomalies completely
type ProviderClient interface {
	StreamRequest(ctx context.Context, req openai.ChatCompletionRequest, apiKey string) (*http.Response, error)
}

// ClientRegistry maps your active structural network handlers
type ClientRegistry struct {
	Pool map[string]ProviderClient
}

// NewClientRegistry initializes the client engine mapping registry pool
func NewClientRegistry() *ClientRegistry {
	reg := &ClientRegistry{
		Pool: make(map[string]ProviderClient),
	}
	
	// Registry seeding targets
	reg.Pool["openai"] = &OpenAIClientEngine{}
	reg.Pool["google"] = &GeminiClientEngine{}
	reg.Pool["anthropic"] = &AnthropicClientEngine{}
	
	return reg
}
