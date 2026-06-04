package openai

import (
	"time"
)

// ChatMessage maps the standard message format used by both OpenAI and Anthropic translators
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest handles the incoming configuration parameters from the calling IDE
type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// UsageMetrics holds the data structures forwarded to the dashboard engine
type UsageMetrics struct {
	RequestedModel string `json:"requested_model"`
	RoutedModel    string `json:"routed_model"`
	PromptTokens   int    `json:"prompt_tokens"`
	ResponseTokens int    `json:"response_tokens"`
	EstimatedCost  float64 `json:"estimated_cost"`
}

// ============================================================================
// Internal Gateway compatibility types & aliases
// ============================================================================

// Message is an alias for ChatMessage to preserve compatibility with internal modules.
type Message = ChatMessage

// Usage represents token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Choice represents a single choice returned in a chat completion response.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatCompletionResponse represents the standard OpenAI chat completions response body.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// MessageDelta represents a partial message delta in a stream.
type MessageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ChunkChoice represents a single choice returned in a chat completion stream chunk.
type ChunkChoice struct {
	Index        int          `json:"index"`
	Delta        MessageDelta `json:"delta"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// ChatCompletionChunk represents the standard OpenAI chat completions streaming chunk.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// AnthropicMessage represents a message in the Anthropic schema.
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicThinkingConfig tracks the extended thinking settings for models like Claude 3.7/4.5
type AnthropicThinkingConfig struct {
	Type         string `json:"type"` // "enabled" or "disabled"
	BudgetTokens int    `json:"budget_tokens"`
}

// AnthropicMessageRequest maps the exact body layout expected at /v1/messages
type AnthropicMessageRequest struct {
	Model       string                   `json:"model"`
	Messages    []AnthropicMessage       `json:"messages"`
	MaxTokens   int                      `json:"max_tokens"`
	Thinking    *AnthropicThinkingConfig `json:"thinking,omitempty"`
	Stream      bool                     `json:"stream"`
}

// TranslateAnthropicToOpenAI maps Claude Code payloads into OpenAI schemas dynamically
func TranslateAnthropicToOpenAI(anthReq AnthropicMessageRequest) ChatCompletionRequest {
	var openAIReq ChatCompletionRequest
	openAIReq.Stream = anthReq.Stream
	
	// Default target choice if downgraded
	openAIReq.Model = "gemini-2.5-flash" 

	for _, msg := range anthReq.Messages {
		openAIReq.Messages = append(openAIReq.Messages, ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return openAIReq
}

// AnthropicRequest represents the standard Anthropic Messages API request body.
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature *float64           `json:"temperature,omitempty"`
	TopP        *float64           `json:"top_p,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

// AnthropicUsage represents usage in the Anthropic response.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicResponseContent represents a content block in an Anthropic response.
type AnthropicResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicResponse represents the standard Anthropic Messages API response body.
type AnthropicResponse struct {
	ID           string                     `json:"id"`
	Type         string                     `json:"type"`
	Role         string                     `json:"role"`
	Content      []AnthropicResponseContent `json:"content"`
	Model        string                     `json:"model"`
	StopReason   string                     `json:"stop_reason"`
	StopSequence *string                    `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage             `json:"usage"`
}

// TranslateOpenAIToAnthropic converts an OpenAI chat completion request to Anthropic message format
func TranslateOpenAIToAnthropic(openAIReq ChatCompletionRequest) AnthropicRequest {
	var anthReq AnthropicRequest
	anthReq.Model = openAIReq.Model
	anthReq.MaxTokens = 1024
	anthReq.Stream = openAIReq.Stream
	for _, msg := range openAIReq.Messages {
		anthReq.Messages = append(anthReq.Messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	return anthReq
}

// TranslateAnthropicResponseToOpenAI converts an Anthropic message response back to OpenAI chat completion response
func TranslateAnthropicResponseToOpenAI(anthResp AnthropicResponse, originalModel string) ChatCompletionResponseWrapper {
	var textResponse string
	for _, block := range anthResp.Content {
		if block.Type == "text" {
			textResponse += block.Text
		}
	}
	choice := WrapperChoice{
		Index: 0,
		Message: ChatMessage{
			Role:    "assistant",
			Content: textResponse,
		},
		Logprobs:     nil,
		FinishReason: "stop",
	}
	var openAIResp ChatCompletionResponseWrapper
	openAIResp.ID = "chatcmpl-" + anthResp.ID
	openAIResp.Object = "chat.completion"
	openAIResp.Created = time.Now().Unix()
	openAIResp.Model = originalModel
	openAIResp.Choices = append(openAIResp.Choices, choice)
	openAIResp.Usage = WrapperUsage{
		PromptTokens:     anthResp.Usage.InputTokens,
		CompletionTokens: anthResp.Usage.OutputTokens,
		TotalTokens:      anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
	}
	openAIResp.SystemFingerprint = "fp_codot_gateway_mesh"
	return openAIResp
}


// GeminiContent represents the basic messaging structure for Google APIs
type GeminiContent struct {
	Role  string       `json:"role"` // Maps to "user" or "model"
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerateRequest represents the incoming JSON structure Google expects
type GeminiGenerateRequest struct {
	Contents []GeminiContent `json:"contents"`
}

// GeminiResponse maps out the structured response returning from Google Cloud
type GeminiCandidate struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type GeminiGenerateResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

// TranslateOpenAIToGemini transforms standard client configurations to Google inputs
func TranslateOpenAIToGemini(openAIReq ChatCompletionRequest) GeminiGenerateRequest {
	var geminiReq GeminiGenerateRequest
	
	for _, msg := range openAIReq.Messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model" // Google API uses the term "model" instead of "assistant"
		}
		
		geminiReq.Contents = append(geminiReq.Contents, GeminiContent{
			Role: role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		})
	}
	
	return geminiReq
}

// TranslateGeminiToOpenAI adapts a Google completion response back into standard OpenAI parameters perfectly
func TranslateGeminiToOpenAI(geminiResp GeminiGenerateResponse, originalModel string) ChatCompletionResponseWrapper {
	var textResponse string
	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		textResponse = geminiResp.Candidates[0].Content.Parts[0].Text
	}

	// 1. Create fully compliant choices data structures
	choice := WrapperChoice{
		Index:        0,
		Message: ChatMessage{
			Role:    "assistant",
			Content: textResponse,
		},
		Logprobs:     nil,
		FinishReason: "stop", // CRITICAL: Tells the IDE tool that the AI has finished typing!
	}

	// 2. Build the final outer response envelope
	var openAIResp ChatCompletionResponseWrapper
	openAIResp.ID = "chatcmpl-gemini-optimized-route"
	openAIResp.Object = "chat.completion"
	openAIResp.Created = time.Now().Unix() // Inject real epoch timestamp
	openAIResp.Model = originalModel
	openAIResp.Choices = append(openAIResp.Choices, choice)
	
	// 3. Populate standard fallback usage metrics
	openAIResp.Usage = WrapperUsage{
		PromptTokens:     15,
		CompletionTokens: 35,
		TotalTokens:      50,
	}
	openAIResp.SystemFingerprint = "fp_codot_gateway_mesh"

	return openAIResp
}

// ChatCompletionResponseWrapper is the absolute standard output shape expected by Cursor/Claude Code
type ChatCompletionResponseWrapper struct {
	ID                string               `json:"id"`
	Object            string               `json:"object"`
	Created           int64                `json:"created"`
	Model             string               `json:"model"`
	Choices           []WrapperChoice      `json:"choices"`
	Usage             WrapperUsage         `json:"usage"`
	SystemFingerprint string               `json:"system_fingerprint,omitempty"`
}

type WrapperChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type WrapperUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SystemModel reflects the storage architecture inside our sqlite registry
type SystemModel struct {
	ID             string  `json:"id"`
	Provider       string  `json:"provider"`
	CapabilityTier string  `json:"capability_tier"`
	InputCostPerM  float64 `json:"input_cost_per_m"`
	OutputCostPerM float64 `json:"output_cost_per_m"`
	IsActive       int     `json:"is_active"` // 1 = true, 0 = false
}
