package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nlink-jp/swrite/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.CurrentProfile != "default" {
		t.Errorf("CurrentProfile = %q, want %q", cfg.CurrentProfile, "default")
	}
	p, ok := cfg.Profiles["default"]
	if !ok {
		t.Fatal("default profile missing")
	}
	if p.Provider != config.ProviderSlack {
		t.Errorf("Provider = %q, want %q", p.Provider, config.ProviderSlack)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.DefaultConfig()
	cfg.Profiles["work"] = config.Profile{
		Provider: config.ProviderSlack,
		Token:    "xoxb-test",
		Channel:  "#ops",
		Username: "bot",
	}

	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Check file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.CurrentProfile != cfg.CurrentProfile {
		t.Errorf("CurrentProfile = %q, want %q", loaded.CurrentProfile, cfg.CurrentProfile)
	}
	wp, ok := loaded.Profiles["work"]
	if !ok {
		t.Fatal("work profile missing after load")
	}
	if wp.Token != "xoxb-test" {
		t.Errorf("Token = %q, want %q", wp.Token, "xoxb-test")
	}
	if wp.Channel != "#ops" {
		t.Errorf("Channel = %q, want %q", wp.Channel, "#ops")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestGetProfile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Profiles["staging"] = config.Profile{Token: "xoxb-staging"}

	p, err := cfg.GetProfile("staging")
	if err != nil {
		t.Fatalf("GetProfile staging: %v", err)
	}
	if p.Token != "xoxb-staging" {
		t.Errorf("Token = %q, want %q", p.Token, "xoxb-staging")
	}

	// Empty name → current profile.
	p, err = cfg.GetProfile("")
	if err != nil {
		t.Fatalf("GetProfile empty: %v", err)
	}
	if p.Provider != config.ProviderSlack {
		t.Errorf("Provider = %q, want %q", p.Provider, config.ProviderSlack)
	}

	// Non-existent profile.
	if _, err := cfg.GetProfile("nope"); err == nil {
		t.Error("expected error for missing profile, got nil")
	}
}

func TestDetectServerMode(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		t.Setenv("SWRITE_MODE", "")
		ok, err := config.DetectServerMode()
		if err != nil || ok {
			t.Errorf("DetectServerMode() = %v, %v; want false, nil", ok, err)
		}
	})

	t.Run("server", func(t *testing.T) {
		t.Setenv("SWRITE_MODE", "server")
		ok, err := config.DetectServerMode()
		if err != nil || !ok {
			t.Errorf("DetectServerMode() = %v, %v; want true, nil", ok, err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv("SWRITE_MODE", "bad")
		_, err := config.DetectServerMode()
		if err == nil {
			t.Error("expected error for invalid mode, got nil")
		}
	})
}

func TestBuildConfigFromEnv(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		t.Setenv("SWRITE_TOKEN", "")
		if _, err := config.BuildConfigFromEnv(); err == nil {
			t.Error("expected error when SWRITE_TOKEN is missing")
		}
	})

	t.Run("full", func(t *testing.T) {
		t.Setenv("SWRITE_TOKEN", "xoxb-env")
		t.Setenv("SWRITE_CHANNEL", "#alerts")
		t.Setenv("SWRITE_USERNAME", "envbot")
		cfg, err := config.BuildConfigFromEnv()
		if err != nil {
			t.Fatalf("BuildConfigFromEnv: %v", err)
		}
		p, err := cfg.GetProfile("")
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if p.Token != "xoxb-env" {
			t.Errorf("Token = %q", p.Token)
		}
		if p.Channel != "#alerts" {
			t.Errorf("Channel = %q", p.Channel)
		}
		if p.Username != "envbot" {
			t.Errorf("Username = %q", p.Username)
		}
	})
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "config.json")

	if err := config.Save(config.DefaultConfig(), path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSave_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := config.Save(config.DefaultConfig(), path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}
}
