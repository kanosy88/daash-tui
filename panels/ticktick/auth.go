package ticktick

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kanosy88/daash-tui/config"
	"golang.org/x/oauth2"
)

func tokenCachePath() string {
	return filepath.Join(config.ConfigDir(), "ticktick_token.json")
}

// EnsureAuth checks for a cached token. If none exists, runs the browser
// OAuth2 flow and saves the token. Must be called before the TUI starts.
func EnsureAuth() error {
	cfg := config.Load().TickTick
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return fmt.Errorf("ticktick client_id and client_secret missing from config.yaml")
	}

	_, err := loadToken()
	if err == nil {
		return nil // already authenticated
	}

	tok, err := browserFlow(cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		return err
	}
	return saveToken(tok)
}

func oauthConfig(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://ticktick.com/oauth/authorize",
			TokenURL: "https://ticktick.com/oauth/token",
		},
		RedirectURL: "http://localhost:8086",
		Scopes:      []string{"tasks:read"},
	}
}

func browserFlow(clientID, clientSecret string) (*oauth2.Token, error) {
	cfg := oauthConfig(clientID, clientSecret)

	codeCh := make(chan string, 1)
	srv := &http.Server{Addr: ":8086"}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprintln(w, "TickTick auth successful — you can close this tab.")
		codeCh <- code
	})
	go srv.ListenAndServe() //nolint:errcheck

	url := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Println("\nOpening browser for TickTick authorization…")
	fmt.Printf("If it doesn't open, visit:\n%s\n\n", url)
	openBrowser(url)

	code := <-codeCh
	srv.Close()

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return tok, nil
}

// httpClient returns an authenticated HTTP client using the cached token.
func httpClient(ctx context.Context) (*http.Client, error) {
	cfg := config.Load().TickTick
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("ticktick client_id / client_secret missing from config.yaml")
	}

	tok, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run EnsureAuth first: %w", err)
	}

	oc := oauthConfig(cfg.ClientID, cfg.ClientSecret)
	return oc.Client(ctx, tok), nil
}

func loadToken() (*oauth2.Token, error) {
	f, err := os.Open(tokenCachePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	return tok, json.NewDecoder(f).Decode(tok)
}

func saveToken(tok *oauth2.Token) error {
	path := tokenCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start() //nolint:errcheck
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:errcheck
	}
}
