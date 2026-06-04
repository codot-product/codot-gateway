package metrics

import (
	"strings"
)

// ModelPricing defines the cost of a model per 1,000,000 tokens.
type ModelPricing struct {
	InputCostPerM  float64 // Cost per 1M input tokens
	OutputCostPerM float64 // Cost per 1M output tokens
}

// FinOpsTracker calculates token consumption and API costs.
type FinOpsTracker struct {
	pricing map[string]ModelPricing
}

// NewFinOpsTracker creates a new FinOpsTracker.
func NewFinOpsTracker() *FinOpsTracker {
	// Standard enterprise pricing as of 2026
	pricing := map[string]ModelPricing{
		"gpt-4o": {
			InputCostPerM:  2.50,
			OutputCostPerM: 10.00,
		},
		"gpt-4o-mini": {
			InputCostPerM:  0.15,
			OutputCostPerM: 0.60,
		},
		"claude-3-5-sonnet": {
			InputCostPerM:  3.00,
			OutputCostPerM: 15.00,
		},
		"claude-3-5-sonnet-20241022": {
			InputCostPerM:  3.00,
			OutputCostPerM: 15.00,
		},
		"claude-3-5-haiku": {
			InputCostPerM:  0.80,
			OutputCostPerM: 4.00,
		},
		"claude-3-5-haiku-20241022": {
			InputCostPerM:  0.80,
			OutputCostPerM: 4.00,
		},
		"deepseek-coder": {
			InputCostPerM:  0.14,
			OutputCostPerM: 0.28,
		},
		"default": {
			InputCostPerM:  1.50,
			OutputCostPerM: 6.00,
		},
	}

	return &FinOpsTracker{pricing: pricing}
}

// GetPricing retrieves pricing for a model (with fallback to default).
func (f *FinOpsTracker) GetPricing(model string) ModelPricing {
	model = strings.ToLower(model)
	// Try prefix matches first, e.g. "gpt-4o-mini-2024-07-18" matches "gpt-4o-mini"
	for k, v := range f.pricing {
		if k != "default" && strings.HasPrefix(model, k) {
			return v
		}
	}
	return f.pricing["default"]
}

// CalculateCost computes cost in USD for a given model, input tokens, and output tokens.
func (f *FinOpsTracker) CalculateCost(model string, inputTokens, outputTokens int) float64 {
	p := f.GetPricing(model)
	inputCost := (float64(inputTokens) / 1000000.0) * p.InputCostPerM
	outputCost := (float64(outputTokens) / 1000000.0) * p.OutputCostPerM
	return inputCost + outputCost
}

// CalculateSavings computes the money saved by:
// 1. Pruning tokens (not sending them to the model)
// 2. Routing a request to a cheaper model (originalModel vs chosenModel)
func (f *FinOpsTracker) CalculateSavings(originalModel, chosenModel string, originalInputTokens, chosenInputTokens, prunedTokens, outputTokens int) float64 {
	origPrice := f.GetPricing(originalModel)
	chosenPrice := f.GetPricing(chosenModel)

	// What it would have cost: original model with all original tokens
	originalTotalInputTokens := originalInputTokens + prunedTokens
	wouldHaveCost := (float64(originalTotalInputTokens) / 1000000.0)*origPrice.InputCostPerM +
		(float64(outputTokens) / 1000000.0)*origPrice.OutputCostPerM

	// What it actually cost: chosen model with chosen (pruned) input tokens
	actualCost := (float64(chosenInputTokens) / 1000000.0)*chosenPrice.InputCostPerM +
		(float64(outputTokens) / 1000000.0)*chosenPrice.OutputCostPerM

	savings := wouldHaveCost - actualCost
	if savings < 0 {
		return 0
	}
	return savings
}

// EstimateTokens provides a simple token estimation for text (approx. 4 chars per token).
func EstimateTokens(text string) int {
	words := len(strings.Fields(text))
	chars := len(text)
	
	tokenEstimate := int(float64(words) / 0.75)
	if tokenEstimate < chars/4 {
		tokenEstimate = chars / 4
	}
	return tokenEstimate
}
