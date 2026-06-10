package xai

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileTokenStore_SaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth", "xai.json")
	store := NewFileTokenStore(path)
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	token := Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		IDToken:      "id-token",
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
	}

	if err := store.Save(token); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected token file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("expected token file mode 0600, got %o", got)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if got.AccessToken != token.AccessToken {
		t.Fatalf("expected access token %q, got %q", token.AccessToken, got.AccessToken)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expires_at %s, got %s", expiresAt, got.ExpiresAt)
	}
}

func TestFileTokenStore_LoadMissing(t *testing.T) {
	store := NewFileTokenStore(filepath.Join(t.TempDir(), "missing.json"))

	_, err := store.Load()
	if !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestFileTokenStore_LoadLegacyHermesAuthJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	data := []byte(`{
		"active_provider": "xai-oauth",
		"providers": {
			"xai-oauth": {
				"access_token": "hermes-access",
				"refresh_token": "hermes-refresh",
				"id_token": "hermes-id",
				"token_type": "Bearer"
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	token, err := NewFileTokenStore(path).Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if token.AccessToken != "hermes-access" {
		t.Fatalf("expected Hermes access token, got %q", token.AccessToken)
	}
}

func TestFileTokenStore_LoadHermesCredentialPoolPrefersOKEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	data := []byte(`{
		"active_provider": "xai-oauth",
		"credential_pool": {
			"xai-oauth": [
				{
					"id": "exhausted",
					"access_token": "exhausted-access",
					"refresh_token": "exhausted-refresh",
					"last_status": "exhausted",
					"token_type": "Bearer"
				},
				{
					"id": "ok",
					"access_token": "pool-access",
					"refresh_token": "pool-refresh",
					"last_status": "ok",
					"base_url": "https://api.x.ai/v1",
					"token_type": "Bearer"
				}
			]
		},
		"providers": {
			"xai-oauth": {
				"tokens": {
					"access_token": "provider-access",
					"refresh_token": "provider-refresh",
					"token_type": "Bearer"
				}
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	token, err := NewFileTokenStore(path).Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if token.AccessToken != "pool-access" {
		t.Fatalf("expected ok pool access token, got %q", token.AccessToken)
	}
	if token.RefreshToken != "pool-refresh" {
		t.Fatalf("expected ok pool refresh token, got %q", token.RefreshToken)
	}
}

func TestFileTokenStore_LoadHermesProviderTokensFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	data := []byte(`{
		"active_provider": "xai-oauth",
		"credential_pool": {
			"xai-oauth": [
				{
					"id": "exhausted",
					"access_token": "exhausted-access",
					"last_status": "exhausted",
					"token_type": "Bearer"
				}
			]
		},
		"providers": {
			"xai-oauth": {
				"auth_mode": "oauth_pkce",
				"tokens": {
					"access_token": "nested-provider-access",
					"refresh_token": "nested-provider-refresh",
					"id_token": "nested-provider-id",
					"token_type": "Bearer"
				}
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	token, err := NewFileTokenStore(path).Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if token.AccessToken != "nested-provider-access" {
		t.Fatalf("expected nested provider access token, got %q", token.AccessToken)
	}
	if token.IDToken != "nested-provider-id" {
		t.Fatalf("expected nested provider id token, got %q", token.IDToken)
	}
}

func TestFileTokenSource_BearerToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xai.json")
	store := NewFileTokenStore(path)
	if err := store.Save(Token{AccessToken: "access-token", TokenType: "Bearer", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	source := NewFileTokenSource(store)
	token, err := source.BearerToken(t.Context())
	if err != nil {
		t.Fatalf("BearerToken failed: %v", err)
	}
	if token != "access-token" {
		t.Fatalf("expected access-token, got %q", token)
	}
}

func TestFileTokenSource_ExpiredToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xai.json")
	store := NewFileTokenStore(path)
	if err := store.Save(Token{AccessToken: "access-token", ExpiresAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	_, err := NewFileTokenSource(store).BearerToken(t.Context())
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}
