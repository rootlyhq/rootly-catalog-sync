package authconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSave(t *testing.T) {
	// Use a temp directory to avoid touching real config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	expiry := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := &Config{
		OAuth: &OAuthData{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresAt:    expiry,
			TokenType:    "Bearer",
		},
		ClientID: "test-client-id",
		Scopes:   []string{"read", "write"},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was created in the right place
	expectedPath := filepath.Join(tmpDir, configDir, configFile)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("config file not created at %s: %v", expectedPath, err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ClientID != cfg.ClientID {
		t.Errorf("ClientID = %q, want %q", loaded.ClientID, cfg.ClientID)
	}
	if len(loaded.Scopes) != len(cfg.Scopes) {
		t.Fatalf("Scopes length = %d, want %d", len(loaded.Scopes), len(cfg.Scopes))
	}
	for i, s := range loaded.Scopes {
		if s != cfg.Scopes[i] {
			t.Errorf("Scopes[%d] = %q, want %q", i, s, cfg.Scopes[i])
		}
	}
	if loaded.OAuth == nil {
		t.Fatal("OAuth is nil")
	}
	if loaded.OAuth.AccessToken != cfg.OAuth.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.OAuth.AccessToken, cfg.OAuth.AccessToken)
	}
	if loaded.OAuth.RefreshToken != cfg.OAuth.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.OAuth.RefreshToken, cfg.OAuth.RefreshToken)
	}
	if !loaded.OAuth.ExpiresAt.Equal(cfg.OAuth.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.OAuth.ExpiresAt, cfg.OAuth.ExpiresAt)
	}
	if loaded.OAuth.TokenType != cfg.OAuth.TokenType {
		t.Errorf("TokenType = %q, want %q", loaded.OAuth.TokenType, cfg.OAuth.TokenType)
	}
}

func TestLoadNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on nonexistent file should not error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load on nonexistent file should return non-nil config")
	}
	if cfg.OAuth != nil {
		t.Error("OAuth should be nil for empty config")
	}
	if cfg.ClientID != "" {
		t.Errorf("ClientID should be empty, got %q", cfg.ClientID)
	}
}
