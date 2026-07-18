package oauth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/rootlyhq/rootly-catalog-sync/authconfig"
)

func TestSaveAndLoadTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	td := &TokenData{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresAt:    expiry,
		TokenType:    "Bearer",
	}

	if err := SaveTokens(td); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	loaded, err := LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens: %v", err)
	}

	if loaded.AccessToken != "access-abc" {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, "access-abc")
	}
	if loaded.RefreshToken != "refresh-xyz" {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, "refresh-xyz")
	}
	if loaded.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", loaded.TokenType, "Bearer")
	}
	if !loaded.ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", loaded.ExpiresAt, expiry)
	}
}

func TestHasTokens_Empty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if HasTokens() {
		t.Error("HasTokens() = true, want false when no config file exists")
	}
}

func TestHasTokens_WithTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	td := &TokenData{
		AccessToken:  "some-token",
		RefreshToken: "some-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
	}
	if err := SaveTokens(td); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	if !HasTokens() {
		t.Error("HasTokens() = false, want true after saving tokens")
	}
}

func TestClearTokens(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	td := &TokenData{
		AccessToken:  "clear-me",
		RefreshToken: "clear-ref",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
	}
	if err := SaveTokens(td); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	if !HasTokens() {
		t.Fatal("HasTokens() = false before clear, expected true")
	}

	if err := ClearTokens(); err != nil {
		t.Fatalf("ClearTokens: %v", err)
	}

	if HasTokens() {
		t.Error("HasTokens() = true after ClearTokens, want false")
	}
}

func TestClearTokens_NoFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Clearing when no config file exists should not error
	if err := ClearTokens(); err != nil {
		t.Fatalf("ClearTokens on missing file: %v", err)
	}
}

func TestIsExpired_Future(t *testing.T) {
	td := &TokenData{
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if IsExpired(td) {
		t.Error("IsExpired() = true for token expiring in 1 hour, want false")
	}
}

func TestIsExpired_Past(t *testing.T) {
	td := &TokenData{
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	if !IsExpired(td) {
		t.Error("IsExpired() = false for token expired 1 hour ago, want true")
	}
}

func TestIsExpired_WithinGracePeriod(t *testing.T) {
	// Token expiring in 20 seconds is within the 30-second grace period
	td := &TokenData{
		ExpiresAt: time.Now().Add(20 * time.Second),
	}
	if !IsExpired(td) {
		t.Error("IsExpired() = false for token expiring in 20s (within 30s grace), want true")
	}
}

func TestTokenDataConversion(t *testing.T) {
	expiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)

	original := &oauth2.Token{
		AccessToken:  "oauth-access",
		RefreshToken: "oauth-refresh",
		TokenType:    "Bearer",
		Expiry:       expiry,
	}

	td := TokenDataFromOAuth2(original)

	if td.AccessToken != "oauth-access" {
		t.Errorf("TokenData.AccessToken = %q, want %q", td.AccessToken, "oauth-access")
	}
	if td.RefreshToken != "oauth-refresh" {
		t.Errorf("TokenData.RefreshToken = %q, want %q", td.RefreshToken, "oauth-refresh")
	}
	if td.TokenType != "Bearer" {
		t.Errorf("TokenData.TokenType = %q, want %q", td.TokenType, "Bearer")
	}
	if !td.ExpiresAt.Equal(expiry) {
		t.Errorf("TokenData.ExpiresAt = %v, want %v", td.ExpiresAt, expiry)
	}

	roundTripped := ToOAuth2Token(td)

	if roundTripped.AccessToken != original.AccessToken {
		t.Errorf("roundTripped.AccessToken = %q, want %q", roundTripped.AccessToken, original.AccessToken)
	}
	if roundTripped.RefreshToken != original.RefreshToken {
		t.Errorf("roundTripped.RefreshToken = %q, want %q", roundTripped.RefreshToken, original.RefreshToken)
	}
	if roundTripped.TokenType != original.TokenType {
		t.Errorf("roundTripped.TokenType = %q, want %q", roundTripped.TokenType, original.TokenType)
	}
	if !roundTripped.Expiry.Equal(original.Expiry) {
		t.Errorf("roundTripped.Expiry = %v, want %v", roundTripped.Expiry, original.Expiry)
	}
}

func TestSaveOAuth2Token(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	tok := &oauth2.Token{
		AccessToken:  "save-oauth2",
		RefreshToken: "save-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(1 * time.Hour).Truncate(time.Second),
	}

	if err := SaveOAuth2Token(tok); err != nil {
		t.Fatalf("SaveOAuth2Token: %v", err)
	}

	loaded, err := LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens: %v", err)
	}
	if loaded.AccessToken != "save-oauth2" {
		t.Errorf("loaded.AccessToken = %q, want %q", loaded.AccessToken, "save-oauth2")
	}
}

func TestSaveTokensPreservesOtherFields(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Save an initial config with extra fields
	initialCfg := &authconfig.Config{
		ClientID: "my-client-id",
		Scopes:   []string{"read", "write"},
	}
	if err := authconfig.Save(initialCfg); err != nil {
		t.Fatalf("authconfig.Save: %v", err)
	}

	// Now save tokens over it
	td := &TokenData{
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
	}
	if err := SaveTokens(td); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	// Verify other fields are preserved
	cfg, err := authconfig.Load()
	if err != nil {
		t.Fatalf("authconfig.Load: %v", err)
	}
	if cfg.ClientID != "my-client-id" {
		t.Errorf("ClientID = %q, want %q (was overwritten)", cfg.ClientID, "my-client-id")
	}
}
