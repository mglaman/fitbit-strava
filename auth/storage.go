package auth

import (
	"encoding/json"
	"os"
	"sync"

	"golang.org/x/oauth2"
)

const CredentialsFile = "credentials.json"

type TokenStore struct {
	mu     sync.Mutex
	Tokens map[string]*oauth2.Token `json:"tokens"`
}

// LoadTokens reads tokens from credentials.json
func LoadTokens() (*TokenStore, error) {
	store := &TokenStore{
		Tokens: make(map[string]*oauth2.Token),
	}

	file, err := os.ReadFile(CredentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil // Return empty store if file doesn't exist
		}
		return nil, err
	}

	if err := json.Unmarshal(file, store); err != nil {
		return nil, err
	}

	return store, nil
}

// SaveTokens writes tokens to credentials.json
func (s *TokenStore) SaveTokens() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(CredentialsFile, data, 0600)
}

// GetToken retrieves a token by provider name (e.g., "fitbit", "strava")
func (s *TokenStore) GetToken(provider string) *oauth2.Token {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Tokens[provider]
}

// SetToken saves a token for a provider
func (s *TokenStore) SetToken(provider string, token *oauth2.Token) error {
	s.mu.Lock()
	s.Tokens[provider] = token
	s.mu.Unlock()

	return s.SaveTokens()
}
