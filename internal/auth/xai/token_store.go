package xai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrTokenNotFound = errors.New("xai oauth token not found")
	ErrTokenExpired  = errors.New("xai oauth token expired")
)

const tokenExpiryLeeway = 5 * time.Minute

type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

func (t Token) expired(now time.Time) bool {
	return !t.ExpiresAt.IsZero() && !t.ExpiresAt.After(now.Add(tokenExpiryLeeway))
}

type TokenStore interface {
	Load() (Token, error)
	Save(Token) error
}

type FileTokenStore struct {
	path string
}

func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: expandHome(path)}
}

func (s *FileTokenStore) Load() (Token, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Token{}, ErrTokenNotFound
		}
		return Token{}, err
	}

	token, err := parseToken(data)
	if err != nil {
		return Token{}, err
	}
	if token.AccessToken == "" {
		return Token{}, fmt.Errorf("%w: missing access_token", ErrTokenNotFound)
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	return token, nil
}

func (s *FileTokenStore) Save(token Token) error {
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func parseToken(data []byte) (Token, error) {
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return Token{}, err
	}
	if token.AccessToken != "" {
		return token, nil
	}

	// Hermes stores xAI OAuth credentials in auth.json. Prefer the credential
	// pool because it records per-token health; fall back to provider tokens for
	// older Hermes files.
	var hermes struct {
		ActiveProvider string `json:"active_provider"`
		CredentialPool map[string][]struct {
			Token
			LastStatus string `json:"last_status"`
		} `json:"credential_pool"`
		Providers map[string]struct {
			Token
			Tokens Token `json:"tokens"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(data, &hermes); err != nil {
		return Token{}, err
	}
	for _, key := range []string{hermes.ActiveProvider, "xai-oauth", "xai"} {
		if key == "" {
			continue
		}
		if poolToken := selectHermesPoolToken(hermes.CredentialPool[key]); poolToken.AccessToken != "" {
			return poolToken, nil
		}
		if provider, ok := hermes.Providers[key]; ok {
			if provider.Token.AccessToken != "" {
				return provider.Token, nil
			}
			if provider.Tokens.AccessToken != "" {
				return provider.Tokens, nil
			}
		}
	}
	return Token{}, nil
}

func selectHermesPoolToken(credentials []struct {
	Token
	LastStatus string `json:"last_status"`
}) Token {
	var fallback Token
	for _, credential := range credentials {
		if credential.AccessToken == "" {
			continue
		}
		if credential.LastStatus == "ok" {
			return credential.Token
		}
		if credential.LastStatus == "" && fallback.AccessToken == "" {
			fallback = credential.Token
		}
	}
	return fallback
}

type FileTokenSource struct {
	store TokenStore
	now   func() time.Time
}

func NewFileTokenSource(store TokenStore) *FileTokenSource {
	return &FileTokenSource{
		store: store,
		now:   time.Now,
	}
}

func (s *FileTokenSource) BearerToken(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	token, err := s.store.Load()
	if err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", ErrTokenNotFound
	}
	if token.expired(s.now()) {
		return "", ErrTokenExpired
	}
	return token.AccessToken, nil
}

func DefaultTokenPath() string {
	return "~/.agenticdub/auth/xai.json"
}

func expandHome(path string) string {
	if path == "" || path == "~" {
		return path
	}
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/"))
}
