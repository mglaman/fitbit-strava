package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type Authenticator struct {
	Store *TokenStore
}

func NewAuthenticator(store *TokenStore) *Authenticator {
	return &Authenticator{Store: store}
}

// GetClient returns an authenticated HTTP client for the given provider.
// It handles token retrieval, refreshing, and initial authorization if needed.
func (a *Authenticator) GetClient(ctx context.Context, provider string, config *oauth2.Config) *http.Client {
	token := a.Store.GetToken(provider)

	// If no token exists, or if it's invalid (nil), start the auth flow
	if token == nil {
		fmt.Printf("No existing token for %s. Starting authentication flow...\n", provider)
		token = a.startAuthFlow(ctx, config)
		if err := a.Store.SetToken(provider, token); err != nil {
			log.Fatalf("Failed to save new token for %s: %v", provider, err)
		}
	} else {
		// Token exists, let's see if it needs refresh.
		// oauth2.Config.Client() automatically handles refreshing if a Transport is used,
		// but we want to persist the refreshed token.
		// So we use TokenSource and wrap it to save on refresh.
	}

	// Create a TokenSource that persists the token when it changes
	ts := config.TokenSource(ctx, token)
	persistingTS := &persistingTokenSource{
		src:      ts,
		store:    a.Store,
		provider: provider,
	}

	return oauth2.NewClient(ctx, persistingTS)
}

func (a *Authenticator) startAuthFlow(ctx context.Context, config *oauth2.Config) *oauth2.Token {
	// 1. Start local server
	codeChan := make(chan string)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Code not found", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "Authorized! You can close this tab/window now.")
		codeChan <- code
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start local auth server: %v", err)
		}
	}()

	// 2. Open browser
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\n----------------------------------------------------------------\n")
	fmt.Printf("Please authenticate by visiting this URL:\n%v\n", authURL)
	fmt.Printf("----------------------------------------------------------------\n")

	// 3. Wait for code
	code := <-codeChan

	// Shutdown server gracefully
	go func() {
		// give it a second to flush response
		time.Sleep(1 * time.Second)
		server.Shutdown(context.Background())
	}()

	// 4. Exchange code for token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Failed to exchange token: %v", err)
	}

	return token
}

// persistingTokenSource wraps an oauth2.TokenSource to save the token whenever it's refreshed.
type persistingTokenSource struct {
	src      oauth2.TokenSource
	store    *TokenStore
	provider string
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.src.Token()
	if err != nil {
		return nil, err
	}
	// We could optimize this to only save if it changed, but saving is cheap here.
	// However, we should be careful about locking. The store handles its own locking.
	// Ideally we only save if the access token or refresh token changed.
	// For now, let's just save it. logic is safe.
	if err := s.store.SetToken(s.provider, token); err != nil {
		fmt.Printf("Warning: Failed to persist refreshed token for %s: %v\n", s.provider, err)
	}
	return token, nil
}
