package cache

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Cache represents a SQLite-backed cache
type Cache struct {
	db *sql.DB
}

// CacheEntry represents a cached prediction
type CacheEntry struct {
	ContextHash string
	Command     string
	CreatedAt   time.Time
	HitCount    int
	LastUsed    time.Time
}

// Open opens or creates a SQLite cache database
func Open(path string) (*Cache, error) {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Create table schema
	schema := `
	CREATE TABLE IF NOT EXISTS predictions (
		context_hash TEXT PRIMARY KEY,
		command TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		hit_count INTEGER DEFAULT 0,
		last_used INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_last_used ON predictions(last_used);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Cache{db: db}, nil
}

// Get retrieves a cached entry by context hash
func (c *Cache) Get(contextHash string) *CacheEntry {
	var entry CacheEntry
	var createdAt, lastUsed int64

	err := c.db.QueryRow(`
		SELECT context_hash, command, created_at, hit_count, last_used
		FROM predictions
		WHERE context_hash = ?
	`, contextHash).Scan(
		&entry.ContextHash,
		&entry.Command,
		&createdAt,
		&entry.HitCount,
		&lastUsed,
	)

	if err != nil {
		return nil
	}

	// Update hit count and last used time
	c.db.Exec(`
		UPDATE predictions
		SET hit_count = hit_count + 1, last_used = ?
		WHERE context_hash = ?
	`, time.Now().Unix(), contextHash)

	entry.CreatedAt = time.Unix(createdAt, 0)
	entry.LastUsed = time.Unix(lastUsed, 0)

	return &entry
}

// Set stores or updates a cache entry
func (c *Cache) Set(contextHash, command string) error {
	now := time.Now().Unix()

	_, err := c.db.Exec(`
		INSERT OR REPLACE INTO predictions
		(context_hash, command, created_at, hit_count, last_used)
		VALUES (?, ?, ?, 0, ?)
	`, contextHash, command, now, now)

	return err
}

// Cleanup removes old entries based on TTL and max entries limit
func (c *Cache) Cleanup(maxAge time.Duration, maxEntries int) error {
	// Delete expired entries
	cutoff := time.Now().Add(-maxAge).Unix()
	if _, err := c.db.Exec("DELETE FROM predictions WHERE last_used < ?", cutoff); err != nil {
		return err
	}

	// Keep only the most recently used maxEntries
	if maxEntries > 0 {
		_, err := c.db.Exec(`
			DELETE FROM predictions
			WHERE context_hash NOT IN (
				SELECT context_hash FROM predictions
				ORDER BY last_used DESC
				LIMIT ?
			)
		`, maxEntries)
		if err != nil {
			return err
		}
	}

	// Vacuum to reclaim space
	_, err := c.db.Exec("VACUUM")
	return err
}

// Stats returns cache statistics
func (c *Cache) Stats() (total int, totalHits int64, err error) {
	err = c.db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(hit_count), 0)
		FROM predictions
	`).Scan(&total, &totalHits)
	return
}

// Close closes the database connection
func (c *Cache) Close() error {
	return c.db.Close()
}
