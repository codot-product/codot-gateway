package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"github.com/codot-product/codot-gateway/api/openai"
)

type AnthropicClientEngine struct{}

func (e *AnthropicClientEngine) StreamRequest(ctx context.Context, req openai.ChatCompletionRequest, apiKey string) (*http.Response, error) {
	// 1. Translate OpenAI request to Anthropic schema format
	anthReq := openai.TranslateOpenAIToAnthropic(req)
	payloadBytes, err := json.Marshal(anthReq)
	if err != nil {
		return nil, err
	}

	// 2. Build target request
	targetURL := "https://api.anthropic.com/v1/messages"
	clientReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	clientReq.Header.Set("Content-Type", "application/json")
	clientReq.Header.Set("x-api-key", apiKey)
	clientReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(clientReq)
	if err != nil {
		return nil, err
	}

	// 3. Translate response body back to OpenAI layout format
	if resp.StatusCode == http.StatusOK {
		respBodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body.Close()
			var anthResp openai.AnthropicResponse
			if err := json.Unmarshal(respBodyBytes, &anthResp); err == nil {
				openAIResp := openai.TranslateAnthropicResponseToOpenAI(anthResp, req.Model)
				openAIBytes, _ := json.Marshal(openAIResp)
				
				resp.Body = io.NopCloser(bytes.NewBuffer(openAIBytes))
				resp.ContentLength = int64(len(openAIBytes))
				resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(openAIBytes)))
				resp.Header.Set("Content-Type", "application/json")
			} else {
				resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
			}
		}
	}

	return resp, nil
}
