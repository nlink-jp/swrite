package slack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/nlink-jp/swrite/internal/slack"
)

// TestListChannels_CacheHit verifies that a second resolveChannelID call does
// not hit the network when a valid disk cache exists.
func TestListChannels_CacheHit(t *testing.T) {
	var listCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		listCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cacheDir := t.TempDir()
	c := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	c.SetCacheDir(cacheDir)

	ctx := context.Background()

	// First call: should hit the network to build the cache.
	if err := c.PostMessage(ctx, slack.PostMessageOptions{
		Channel: "#general",
	}); err != nil {
		t.Fatalf("first PostMessage: %v", err)
	}
	if listCalls.Load() != 1 {
		t.Fatalf("expected 1 conversations.list call after first post, got %d", listCalls.Load())
	}

	// Second call: cache should be warm — no additional list call.
	if err := c.PostMessage(ctx, slack.PostMessageOptions{
		Channel: "#general",
	}); err != nil {
		t.Fatalf("second PostMessage: %v", err)
	}
	if listCalls.Load() != 1 {
		t.Fatalf("expected still 1 conversations.list call (cache hit), got %d", listCalls.Load())
	}

	// Verify cache file was created.
	if _, err := os.Stat(filepath.Join(cacheDir, "channels.json")); err != nil {
		t.Errorf("cache file not found: %v", err)
	}
}

// TestListChannels_NoCacheDir verifies that without a cache dir the list API
// is called on every invocation.
func TestListChannels_NoCacheDir(t *testing.T) {
	var listCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		listCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	// No SetCacheDir call — cache is disabled.

	ctx := context.Background()
	for i := range 3 {
		if err := c.PostMessage(ctx, slack.PostMessageOptions{Channel: "#general"}); err != nil {
			t.Fatalf("PostMessage[%d]: %v", i, err)
		}
	}
	if listCalls.Load() != 3 {
		t.Fatalf("expected 3 conversations.list calls (no cache), got %d", listCalls.Load())
	}
}
