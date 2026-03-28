package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nlink-jp/swrite/cmd"
	"github.com/nlink-jp/swrite/internal/slack"
)

func newUploadTestServer(t *testing.T) (*httptest.Server, *bool) {
	t.Helper()
	completeCalled := false
	var srv *httptest.Server

	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"upload_url": srv.URL + "/upload-target",
			"file_id":    "F001",
		})
	})
	mux.HandleFunc("/upload-target", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, _ *http.Request) {
		completeCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &completeCalled
}

func TestUpload_FileFlag(t *testing.T) {
	srv, completeCalled := newUploadTestServer(t)

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	fpath := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(fpath, []byte("a,b\n1,2"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "upload", "--file", fpath})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !*completeCalled {
		t.Error("files.completeUploadExternal was not called")
	}
}

func TestUpload_WithComment(t *testing.T) {
	var gotComplete map[string]any
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.list", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":                true,
			"channels":          []map[string]any{{"id": "C001", "name": "general"}},
			"response_metadata": map[string]string{"next_cursor": ""},
		})
	})
	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true, "upload_url": srv.URL + "/up", "file_id": "F002",
		})
	})
	mux.HandleFunc("/up", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotComplete)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	cfgPath := setupConfig(t, "xoxb-test", "#general")
	cmd.SetNewClientFuncForTest(func(token, _ string) slack.Client {
		return slack.NewHTTPClient(token).WithBaseURL(srv.URL)
	})
	defer cmd.ResetClientFunc()

	fpath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(fpath, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "upload",
		"--file", fpath,
		"--comment", "weekly report",
	})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotComplete["initial_comment"] != "weekly report" {
		t.Errorf("initial_comment = %v, want %q", gotComplete["initial_comment"], "weekly report")
	}
}

func TestUpload_MissingFile(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "upload", "--file", "/nonexistent/file.txt"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestUpload_MutuallyExclusiveFlags(t *testing.T) {
	cfgPath := setupConfig(t, "xoxb-test", "#general")

	fpath := filepath.Join(t.TempDir(), "f.txt")
	_ = os.WriteFile(fpath, []byte("x"), 0o600)

	root := cmd.RootCmd()
	root.SetArgs([]string{"--config", cfgPath, "upload",
		"--file", fpath, "--channel", "#foo", "--user", "U001",
	})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for --channel + --user")
	}
}
