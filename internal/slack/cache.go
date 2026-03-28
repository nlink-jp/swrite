package slack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// cacheEnvelope wraps cached data with a timestamp for TTL checks.
type cacheEnvelope[T any] struct {
	CachedAt time.Time `json:"cached_at"`
	Data     T         `json:"data"`
}

// loadCache reads a JSON cache file and returns the data if it exists and is
// younger than ttl. Returns the zero value and false on any error or expiry.
func loadCache[T any](path string, ttl time.Duration) (T, bool) {
	var zero T
	b, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return zero, false
	}
	var env cacheEnvelope[T]
	if err := json.Unmarshal(b, &env); err != nil {
		return zero, false
	}
	if time.Since(env.CachedAt) > ttl {
		return zero, false
	}
	return env.Data, true
}

// saveCache writes data to a JSON cache file. Errors are silently ignored
// because a failed cache write must never break the main operation.
func saveCache[T any](path string, data T) {
	env := cacheEnvelope[T]{CachedAt: time.Now(), Data: data}
	b, err := json.Marshal(env)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
}
