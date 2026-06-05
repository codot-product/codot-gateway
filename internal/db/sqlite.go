package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	"github.com/codot-product/codot-gateway/api/openai"
)

var DB *sql.DB

type AuditLog struct {
	ID        int64     `db:"id"`
	RequestID string    `db:"request_id"`
	Timestamp time.Time `db:"timestamp"`
	FileType  string    `db:"file_type"`
	Severity  string    `db:"severity"`
	RuleName  string    `db:"rule_name"`
	Message   string    `db:"message"`
	Snippet   string    `db:"snippet"`
}

// resolveDBPath returns a stable, absolute path for the database file
// so the gateway always uses the same vault regardless of working directory.
func resolveDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to CWD if home directory can't be determined
		log.Println("[DB] Warning: could not resolve home dir, using local path")
		return "./gateway_metrics.db"
	}
	dbDir := filepath.Join(homeDir, ".codot")
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		log.Printf("[DB] Warning: could not create ~/.codot dir: %v", err)
		return "./gateway_metrics.db"
	}
	return filepath.Join(dbDir, "gateway_metrics.db")
}

func InitDB() {
	dbPath := resolveDBPath()
	log.Printf("💾 Database path: %s", dbPath)
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to local SQLite engine: %v", err)
	}

	// 1. Maintain your original logging schema table
	tokenTableQuery := `
	CREATE TABLE IF NOT EXISTS token_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		requested_model TEXT,
		routed_model TEXT,
		prompt_char_count INTEGER,
		estimated_cost REAL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// 2. NEW! Create the local credentials security configuration key table
	vaultTableQuery := `
	CREATE TABLE IF NOT EXISTS api_keys (
		provider TEXT PRIMARY KEY,
		secret_key TEXT
	);`

	_, err = DB.Exec(tokenTableQuery)
	if err != nil {
		log.Fatalf("Fatal: Failed to build metric storage tables: %v", err)
	}

	_, err = DB.Exec(vaultTableQuery)
	if err != nil {
		log.Fatalf("Fatal: Failed to build API configuration key vault: %v", err)
	}
	
	log.Println("💾 Storage engine successfully extended with Config Key Vault schemas.")

	ExtendDatabaseForDynamicModels()
}

func ExtendDatabaseForDynamicModels() {
	// Create a structural configuration dictionary table for dynamic model tracking
	query := `
	CREATE TABLE IF NOT EXISTS system_models (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		capability_tier TEXT NOT NULL,
		input_cost_per_m REAL NOT NULL,
		output_cost_per_m REAL NOT NULL,
		is_active INTEGER DEFAULT 1
	);`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("Fatal: Dynamic model schema compilation failed: %v", err)
	}
	
	log.Println("💾 SQLite extended: Dynamic Model Registry configurations are now live.")

	// Seed default models if table is empty
	var count int
	err = DB.QueryRow("SELECT COUNT(*) FROM system_models").Scan(&count)
	if err == nil && count == 0 {
		defaultModels := []struct {
			id             string
			provider       string
			capabilityTier string
			inputCost      float64
			outputCost     float64
		}{
			{"gemini-2.5-flash", "google", "flash", 0.075, 0.075},
			{"gpt-5.4", "openai", "balanced", 1.50, 1.50},
			{"claude-4.8-opus", "anthropic", "frontier", 15.00, 15.00},
		}

		for _, m := range defaultModels {
			_, _ = DB.Exec(`INSERT INTO system_models (id, provider, capability_tier, input_cost_per_m, output_cost_per_m, is_active) 
				VALUES (?, ?, ?, ?, ?, 1)`, m.id, m.provider, m.capabilityTier, m.inputCost, m.outputCost)
		}
		log.Println("💾 SQLite seeded: Loaded default provider capability tier metrics.")
	}
}

// Helper utility to pull active keys on-demand inside our proxy loop
func GetAPIKey(provider string) string {
	var key string
	err := DB.QueryRow("SELECT secret_key FROM api_keys WHERE provider = ?", provider).Scan(&key)
	if err != nil {
		return "" // Returns empty if the developer hasn't configured this provider yet
	}
	return key
}

func SaveAPIKey(provider, secretKey string) error {
	query := `INSERT OR REPLACE INTO api_keys (provider, secret_key) VALUES (?, ?)`
	_, err := DB.Exec(query, provider, secretKey)
	return err
}

func RecordMetric(reqModel, routedModel string, charCount int, cost float64) {
	query := `INSERT INTO token_logs (requested_model, routed_model, prompt_char_count, estimated_cost) VALUES (?, ?, ?, ?)`
	_, err := DB.Exec(query, reqModel, routedModel, charCount, cost)
	if err != nil {
		log.Printf("[ERROR] Database logging failure: %v", err)
	}
}

// GetAllModels queries our local registry state configuration 
func GetAllModels() ([]openai.SystemModel, error) {
	rows, err := DB.Query("SELECT id, provider, capability_tier, input_cost_per_m, output_cost_per_m, is_active FROM system_models ORDER BY provider, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []openai.SystemModel
	for rows.Next() {
		var m openai.SystemModel
		err := rows.Scan(&m.ID, &m.Provider, &m.CapabilityTier, &m.InputCostPerM, &m.OutputCostPerM, &m.IsActive)
		if err == nil {
			list = append(list, m)
		}
	}
	return list, nil
}

// UpsertModel inserts a new model structure or modifies parameters if it already exists
func UpsertModel(m openai.SystemModel) error {
	query := `
	INSERT INTO system_models (id, provider, capability_tier, input_cost_per_m, output_cost_per_m, is_active)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		provider=excluded.provider,
		capability_tier=excluded.capability_tier,
		input_cost_per_m=excluded.input_cost_per_m,
		output_cost_per_m=excluded.output_cost_per_m,
		is_active=excluded.is_active;`
		
	_, err := DB.Exec(query, m.ID, m.Provider, m.CapabilityTier, m.InputCostPerM, m.OutputCostPerM, m.IsActive)
	return err
}
