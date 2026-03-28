package slack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nlink-jp/swrite/internal/slack"
)

// newTestServer creates an httptest.Server with the given mux.
// Returns the server; the caller must call defer srv.Close().
func newPostServer(t *testing.T, gotChannel, gotText *string) *httptest.Server {
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
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if gotChannel != nil {
			if ch, ok := body["channel"].(string); ok {
				*gotChannel = ch
			}
		}
		if gotText != nil {
			if txt, ok := body["text"].(string); ok {
				*gotText = txt
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	return httptest.NewServer(mux)
}

func TestHTTPClient_PostMessage_ByChannelName(t *testing.T) {
	var gotChannel, gotText string
	srv := newPostServer(t, &gotChannel, &gotText)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		Channel: "#general",
		Text:    "hello world",
	})
	if err != nil {
		t.Fatalf("PostMessage: %v", err)
	}
	if gotChannel != "C001" {
		t.Errorf("channel sent = %q, want %q", gotChannel, "C001")
	}
	if gotText != "hello world" {
		t.Errorf("text sent = %q, want %q", gotText, "hello world")
	}
}

func TestHTTPClient_PostMessage_DefaultChannel(t *testing.T) {
	var gotChannel string
	srv := newPostServer(t, &gotChannel, nil)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		DefaultChannel: "general",
		Text:           "from default",
	})
	if err != nil {
		t.Fatalf("PostMessage: %v", err)
	}
	if gotChannel != "C001" {
		t.Errorf("channel sent = %q, want C001", gotChannel)
	}
}

func TestHTTPClient_PostMessage_ChannelID(t *testing.T) {
	// When a channel ID is given, conversations.list should not be called.
	listCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		listCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "channels": []any{}, "response_metadata": map[string]string{"next_cursor": ""}})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		Channel: "C0123456789",
		Text:    "direct id",
	})
	if err != nil {
		t.Fatalf("PostMessage: %v", err)
	}
	if listCalled {
		t.Error("conversations.list should not be called when a channel ID is provided")
	}
}

func TestHTTPClient_PostMessage_NoDestination(t *testing.T) {
	client := slack.NewHTTPClient("xoxb-test")
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		Text: "no dest",
	})
	if err == nil {
		t.Error("expected error when no destination is set")
	}
}

func TestHTTPClient_PostMessage_DMViaUser(t *testing.T) {
	var gotChannel string
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.open", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"channel": map[string]string{"id": "D001"},
		})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if ch, ok := body["channel"].(string); ok {
			gotChannel = ch
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		UserID: "U001",
		Text:   "dm message",
	})
	if err != nil {
		t.Fatalf("PostMessage DM: %v", err)
	}
	if gotChannel != "D001" {
		t.Errorf("channel = %q, want D001", gotChannel)
	}
}

func TestHTTPClient_PostMessage_NotInChannel_AutoJoin(t *testing.T) {
	joinCalled := false
	postCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/conversations.join", func(w http.ResponseWriter, _ *http.Request) {
		joinCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
		postCount++
		w.Header().Set("Content-Type", "application/json")
		if postCount == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "not_in_channel"})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		Channel: "#general",
		Text:    "auto join test",
	})
	if err != nil {
		t.Fatalf("PostMessage: %v", err)
	}
	if !joinCalled {
		t.Error("conversations.join was not called")
	}
	if postCount != 2 {
		t.Errorf("chat.postMessage called %d times, want 2", postCount)
	}
}

func TestHTTPClient_PostFile(t *testing.T) {
	// Create a temp file to upload.
	dir := t.TempDir()
	fpath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(fpath, []byte("hello file"), 0o600); err != nil {
		t.Fatal(err)
	}

	completeCalled := false
	mux := http.NewServeMux()

	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})

	srv := httptest.NewServer(mux) // need server URL before registering upload handler

	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"upload_url": srv.URL + "/upload",
			"file_id":    "F001",
		})
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, _ *http.Request) {
		completeCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostFile(context.Background(), slack.PostFileOptions{
		Channel:  "#general",
		FilePath: fpath,
		Filename: "test.txt",
		Comment:  "a test file",
	})
	srv.Close()
	if err != nil {
		t.Fatalf("PostFile: %v", err)
	}
	if !completeCalled {
		t.Error("files.completeUploadExternal was not called")
	}
}

func TestHTTPClient_PostMessage_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat.postMessage", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "channel_not_found"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := slack.NewHTTPClient("xoxb-test").WithBaseURL(srv.URL)
	err := client.PostMessage(context.Background(), slack.PostMessageOptions{
		Channel: "C0000000000",
		Text:    "test",
	})
	if err == nil {
		t.Error("expected error for API error response")
	}
}
