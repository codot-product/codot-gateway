package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
	"github.com/codot-product/codot-gateway/api/openai"
)

type OpenAIClientEngine struct{}

func (e *OpenAIClientEngine) StreamRequest(ctx context.Context, req openai.ChatCompletionRequest, apiKey string) (*http.Response, error) {
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	clientReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	clientReq.Header.Set("Content-Type", "application/json")
	clientReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(clientReq)
}
