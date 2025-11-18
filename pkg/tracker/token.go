package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Record represents a single token usage record
type Record struct {
	ID                  string    `json:"id"`
	Timestamp           time.Time `json:"timestamp"`
	Method              string    `json:"method"`
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	InputTokens         int       `json:"input_tokens"`
	OutputTokens        int       `json:"output_tokens"`
	CacheCreationTokens int       `json:"cache_creation_tokens"`
	CacheReadTokens     int       `json:"cache_read_tokens"`
}

// Storage represents the JSON storage structure
type Storage struct {
	Version string   `json:"version"`
	Records []Record `json:"records"`
}

var (
	mu     sync.Mutex
	dbPath string
)

func init() {
	home, _ := os.UserHomeDir()
	dbPath = filepath.Join(home, ".llmsh", "tokens.json")
}

// SetDBPath sets a custom database path (useful for testing)
func SetDBPath(path string) {
	mu.Lock()
	defer mu.Unlock()
	dbPath = path
}

// RecordUsage records a new token usage entry
func RecordUsage(r *Record) error {
	mu.Lock()
	defer mu.Unlock()

	r.ID = generateID()
	r.Timestamp = time.Now()

	// Read existing data
	storage := &Storage{Version: "1.0"}
	if data, err := os.ReadFile(dbPath); err == nil {
		json.Unmarshal(data, storage)
	}

	// Add new record
	storage.Records = append(storage.Records, *r)

	// Write back to file
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(dbPath, data, 0644)
}

// LoadRecords loads all token usage records
func LoadRecords() (*Storage, error) {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Storage{Version: "1.0", Records: []Record{}}, nil
		}
		return nil, err
	}

	var storage Storage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, err
	}

	return &storage, nil
}

// DayStats represents aggregated statistics for a single day
type DayStats struct {
	Day             string
	Count           int
	InputTokens     int
	OutputTokens    int
	CacheReadTokens int
}

// AggregateByDay aggregates records by day
func AggregateByDay(records []Record) map[string]*DayStats {
	stats := make(map[string]*DayStats)

	for _, r := range records {
		day := r.Timestamp.Format("2006-01-02")

		if stats[day] == nil {
			stats[day] = &DayStats{Day: day}
		}

		stats[day].Count++
		stats[day].InputTokens += r.InputTokens
		stats[day].OutputTokens += r.OutputTokens
		stats[day].CacheReadTokens += r.CacheReadTokens
	}

	return stats
}

// MethodStats represents aggregated statistics by method
type MethodStats struct {
	Method       string
	Count        int
	InputTokens  int
	OutputTokens int
}

// AggregateByMethod aggregates records by method
func AggregateByMethod(records []Record) map[string]*MethodStats {
	stats := make(map[string]*MethodStats)

	for _, r := range records {
		if stats[r.Method] == nil {
			stats[r.Method] = &MethodStats{Method: r.Method}
		}

		stats[r.Method].Count++
		stats[r.Method].InputTokens += r.InputTokens
		stats[r.Method].OutputTokens += r.OutputTokens
	}

	return stats
}

// ProviderModelStats represents aggregated statistics by provider and model
type ProviderModelStats struct {
	Provider     string
	Model        string
	Count        int
	InputTokens  int
	OutputTokens int
	CacheReadTokens int
}

// AggregateByProviderModel aggregates records by provider and model
func AggregateByProviderModel(records []Record) []*ProviderModelStats {
	statsMap := make(map[string]*ProviderModelStats)

	for _, r := range records {
		key := r.Provider + ":" + r.Model
		if statsMap[key] == nil {
			statsMap[key] = &ProviderModelStats{
				Provider: r.Provider,
				Model:    r.Model,
			}
		}

		statsMap[key].Count++
		statsMap[key].InputTokens += r.InputTokens
		statsMap[key].OutputTokens += r.OutputTokens
		statsMap[key].CacheReadTokens += r.CacheReadTokens
	}

	// Convert map to slice for easier sorting
	stats := make([]*ProviderModelStats, 0, len(statsMap))
	for _, stat := range statsMap {
		stats = append(stats, stat)
	}

	return stats
}

// generateID generates a unique ID based on timestamp
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
