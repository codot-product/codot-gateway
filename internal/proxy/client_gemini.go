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

type GeminiClientEngine struct{}

func (e *GeminiClientEngine) StreamRequest(ctx context.Context, req openai.ChatCompletionRequest, apiKey string) (*http.Response, error) {
	// 1. Map OpenAI parameters directly into Google Gemini schemas
	geminiSchemaPayload := openai.TranslateOpenAIToGemini(req)
	payloadBytes, err := json.Marshal(geminiSchemaPayload)
	if err != nil {
		return nil, err
	}

	// 2. Build target destination URL utilizing key path queries
	targetURL := "https://generativelanguage.googleapis.com/v1beta/models/" + req.Model + ":generateContent?key=" + apiKey

	clientReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}
	clientReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	
	// 3. Dispatch out across the web
	resp, err := client.Do(clientReq)
	if err != nil {
		return nil, err
	}

	// 4. Translate response body back to OpenAI layout format
	if resp.StatusCode == http.StatusOK {
		respBodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body.Close()
			var geminiResp openai.GeminiGenerateResponse
			if err := json.Unmarshal(respBodyBytes, &geminiResp); err == nil {
				openAIResp := openai.TranslateGeminiToOpenAI(geminiResp, req.Model)
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
