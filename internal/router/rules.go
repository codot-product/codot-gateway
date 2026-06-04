package router

// Define our standardized capability tiers
type ModelTier int

const (
	TierFlash   ModelTier = iota // Ultra-low-cost, high-speed documentation, simple regex, unit test structures
	TierBalanced                 // General programming, debugging isolated functions, everyday refactoring
	TierFrontier                 // Massive reasoning loops, complex architecture changes, race conditions, multi-file updates
)

type ModelDetails struct {
	ID       string
	Provider string
	CostPerM float64 // Abstracted financial tracker for FinOps mapping
}

// Global registry mapping our target pool of models
var ModelRegistry = map[ModelTier]ModelDetails{
	TierFlash:    {ID: "gemini-2.5-flash", Provider: "google", CostPerM: 0.075},
	TierBalanced: {ID: "gpt-5.4", Provider: "openai", CostPerM: 1.50},
	TierFrontier: {ID: "claude-4.8-opus", Provider: "anthropic", CostPerM: 15.00},
}

// Categorized Intent Keywords
var LowComplexityKeywords = []string{"docstring", "comment", "readme", "format", "typo"}
var MediumComplexityKeywords = []string{"unit test", "regex", "add route", "log", "export"}
var HighComplexityKeywords = []string{"refactor", "race condition", "memory leak", "architecture", "optimize", "db migration"}
