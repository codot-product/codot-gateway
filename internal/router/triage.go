package router

import (
	"strings"
	"github.com/codot-product/codot-gateway/api/openai"
	"github.com/codot-product/codot-gateway/internal/db"
)

// EvaluatePromptDynamic queries the SQLite registry on-the-fly instead of using hardcoded arrays
func EvaluatePromptDynamic(prompt string, characterCount int) openai.SystemModel {
	promptLower := strings.ToLower(prompt)

	// Fetch all active models from the SQLite configuration table
	allModels, err := db.GetAllModels()
	
	// Safe fallback: Create an emergency model profile if the database is unpopulated
	fallback := openai.SystemModel{
		ID:             "gemini-2.5-flash",
		Provider:       "google",
		CapabilityTier: "flash",
		InputCostPerM:  0.075,
		OutputCostPerM: 0.075,
		IsActive:       1,
	}
	
	if err != nil || len(allModels) == 0 {
		return fallback
	}

	// 1. Establish Intent Targets
	var targetTier string
	
	// Basic heuristic parsing matching keyword arrays to capability strings
	isHighComplexity := containsAny(promptLower, "refactor", "race condition", "memory leak", "architecture", "optimize", "db migration", "deepseek-r1")
	isMediumComplexity := containsAny(promptLower, "unit test", "regex", "add route", "log", "export")

	if isHighComplexity || characterCount > 15000 {
		targetTier = "frontier"
	} else if isMediumComplexity || characterCount > 5000 {
		targetTier = "balanced"
	} else {
		targetTier = "flash"
	}

	// 2. Scan the database models to find a matching, active tier variant
	for _, model := range allModels {
		if model.IsActive == 1 && model.CapabilityTier == targetTier {
			return model // Matches the perfect active model profile!
		}
	}

	// 3. Smart Fallback Scan: Find any active model if the exact tier isn't configured/enabled
	for _, model := range allModels {
		if model.IsActive == 1 {
			return model
		}
	}

	return fallback
}

// Helper utility to quickly search strings for multiple keyword matches
func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

