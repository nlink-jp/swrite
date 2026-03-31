package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlink-jp/swrite/cmd"
	"github.com/nlink-jp/swrite/internal/config"
	"github.com/nlink-jp/swrite/internal/slack"
)

// setupConfig writes a minimal config file and sets --config for the root command.
func setupConfig(t *testing.T, token, channel string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				Provider: "slack",
				Token:    token,
				Channel:  channel,
			},
		},
	}
	if err := config.Save(cfg, cfgPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return cfgPath
}

// newPostTestServer creates an httptest.Server that handles conversations.list
// and chat.postMessage. It writes the received body to gotBody.
func newPostTestServer(t *testing.T, gotBody *map[string]any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{"id": "C001", "name": "general"},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		if gotBody != nil {
			_ = json.NewDecoder(r.Body).Decode(gotBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "ts": "1234567890.123456", "channel": "C001"})
	})
	return httptest.NewServer(mux)
}

func TestPost_TextArgument(t *testing.T) {
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"--config", cfgPath, "post", "hello test"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["text"] != "hello test" {
		t.Errorf("text = %v, want %q", gotBody["text"], "hello test")
	}
	if !strings.Contains(stdout.String(), "1234567890.123456") {
		t.Errorf("stdout = %q, want to contain ts value '1234567890.123456'", stdout.String())
	}
}

func TestPost_FromFile(t *testing.T) {
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	msgFile := filepath.Join(t.TempDir(), "msg.txt")
	if err := os.WriteFile(msgFile, []byte("from file content"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["text"] != "from file content" {
		t.Errorf("text = %v, want %q", gotBody["text"], "from file content")
	}
}

func TestPost_BlocksArray(t *testing.T) {
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	blocksJSON := `[{"type":"section","text":{"type":"mrkdwn","text":"*hello*"}}]`
	msgFile := filepath.Join(t.TempDir(), "blocks.json")
	if err := os.WriteFile(msgFile, []byte(blocksJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "blocks", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, ok := gotBody["blocks"]; !ok {
		t.Error("expected 'blocks' field in payload")
	}
	if gotBody["text"] != nil && gotBody["text"] != "" {
		t.Errorf("text should be empty for blocks format, got %v", gotBody["text"])
	}
}

func TestPost_BlocksWrapper(t *testing.T) {
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	wrapperJSON := `{"blocks":[{"type":"section","text":{"type":"mrkdwn","text":"hi"}}]}`
	msgFile := filepath.Join(t.TempDir(), "wrapper.json")
	if err := os.WriteFile(msgFile, []byte(wrapperJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "blocks", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, ok := gotBody["blocks"]; !ok {
		t.Error("expected 'blocks' field in payload")
	}
}

func TestPost_InvalidFormat(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "xml", "msg"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for invalid --format")
	}
}

func TestPost_MutuallyExclusiveFlags(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--channel", "#foo", "--user", "U001", "msg"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for --channel + --user")
	}
}

func TestPost_ChannelOverride(t *testing.T) {
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{"id": "C001", "name": "general"},
				{"id": "C002", "name": "ops"},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Profile default is #general (C001), but we override with #ops (C002).
	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--channel", "#ops", "override test"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["channel"] != "C002" {
		t.Errorf("channel = %v, want C002", gotBody["channel"])
	}
}

func TestParseBlocks_Array(t *testing.T) {
	input := `[{"type":"section"}]`
	// Exercise parseBlocks indirectly through post --format blocks.
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	msgFile := filepath.Join(t.TempDir(), "b.json")
	if err := os.WriteFile(msgFile, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "blocks", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, ok := gotBody["blocks"]; !ok {
		t.Error("blocks not present in payload")
	}
}

func TestParseBlocks_InvalidJSON(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")
	msgFile := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(msgFile, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "blocks", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for invalid blocks JSON")
	}
}

// TestPost_ServerMode verifies that SWRITE_MODE=server uses env vars.
func TestPost_ServerMode(t *testing.T) {
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("SWRITE_MODE", "server")
	t.Setenv("SWRITE_TOKEN", "xoxb-env")
	t.Setenv("SWRITE_CHANNEL", "C0123456789X")

	// Provide a test client pointing at the test server.
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"post", "server mode test"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["channel"] != "C0123456789X" {
		t.Errorf("channel = %v, want C0123456789X", gotBody["channel"])
	}
	if !strings.Contains(fmt.Sprintf("%v", gotBody["text"]), "server mode test") {
		t.Errorf("text = %v, want to contain 'server mode test'", gotBody["text"])
	}
}

// TestPost_ServerMode_CacheDir verifies that SWRITE_CACHE_DIR enables caching in server mode.
func TestPost_ServerMode_CacheDir(t *testing.T) {
	var listCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		listCalls++
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
	t.Setenv("SWRITE_MODE", "server")
	t.Setenv("SWRITE_TOKEN", "xoxb-env")
	t.Setenv("SWRITE_CHANNEL", "#general")
	t.Setenv("SWRITE_CACHE_DIR", cacheDir)

	cmd.SetNewClientFuncForTest(func(token, cd string) slack.Client {
		c := slack.NewHTTPClient(token).WithBaseURL(srv.URL)
		if cd != "" {
			c.SetCacheDir(cd)
		}
		return c
	})
	defer cmd.ResetClientFunc()

	// First invocation: populates cache.
	root := cmd.RootCmd()
	root.SetArgs([]string{"post", "first"})
	root.SetErr(new(bytes.Buffer))
	if err := root.Execute(); err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	// Second invocation: should hit cache, not call conversations.list again.
	root2 := cmd.RootCmd()
	root2.SetArgs([]string{"post", "second"})
	root2.SetErr(new(bytes.Buffer))
	if err := root2.Execute(); err != nil {
		t.Fatalf("second Execute: %v", err)
	}

	if listCalls != 1 {
		t.Errorf("conversations.list called %d times, want 1 (cache hit on second call)", listCalls)
	}
}


func TestPost_PayloadFormat(t *testing.T) {
	var gotBody map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "C0123456789X")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	payloadJSON := `{"text":"fallback","attachments":[{"color":"#e01e5a","blocks":[{"type":"section","text":{"type":"mrkdwn","text":"hello"}}]}]}`
	msgFile := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(msgFile, []byte(payloadJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "payload", "--from-file", msgFile})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["text"] != "fallback" {
		t.Errorf("text = %v, want 'fallback'", gotBody["text"])
	}
	attachments, ok := gotBody["attachments"]
	if !ok {
		t.Fatal("expected attachments in payload")
	}
	arr, ok := attachments.([]any)
	if !ok || len(arr) == 0 {
		t.Errorf("attachments should be a non-empty array, got %v", attachments)
	}
}

func TestPost_NoUnfurl(t *testing.T) {
	var gotBody map[string]any
	srv := newPostTestServer(t, &gotBody)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--no-unfurl", "check https://example.com"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotBody["unfurl_links"] != false {
		t.Errorf("unfurl_links = %v, want false", gotBody["unfurl_links"])
	}
	if gotBody["unfurl_media"] != false {
		t.Errorf("unfurl_media = %v, want false", gotBody["unfurl_media"])
	}
}

func TestPost_PayloadFormatInvalid(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "post", "--format", "payload", "{}"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for empty payload")
	}
}
