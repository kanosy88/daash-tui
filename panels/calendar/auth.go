package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gcal "google.golang.org/api/calendar/v3"

	"daash/config"
)

func tokenCachePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "daash", "token.json")
}

func oauthConfig() (*oauth2.Config, error) {
	cfg := config.Load().Google
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("google.client_id and google.client_secret missing from config.yaml")
	}
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8085",
		Scopes:       []string{gcal.CalendarReadonlyScope},
	}, nil
}

// EnsureAuth must be called before the TUI starts.
// If no token is cached it runs the browser OAuth flow in the current terminal,
// then saves the token so future calls are instant.
func EnsureAuth() error {
	_, err := loadToken()
	if err == nil {
		return nil // already authenticated
	}

	cfg, err := oauthConfig()
	if err != nil {
		return err
	}

	tok, err := browserFlow(cfg)
	if err != nil {
		return err
	}
	return saveToken(tok)
}

// oauthClient returns an authenticated HTTP client using the cached token.
func oauthClient(ctx context.Context) (*http.Client, error) {
	cfg, err := oauthConfig()
	if err != nil {
		return nil, err
	}

	tok, err := loadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run EnsureAuth first: %w", err)
	}

	return cfg.Client(ctx, tok), nil
}

// browserFlow opens the Google consent page and waits for the auth code
// on a local redirect server.
func browserFlow(cfg *oauth2.Config) (*oauth2.Token, error) {
	codeCh := make(chan string, 1)
	srv := &http.Server{Addr: ":8085"}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Fprintln(w, "Auth successful — you can close this tab.")
		codeCh <- code
	})
	go srv.ListenAndServe() //nolint:errcheck

	url := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Println("\nOpening browser for Google Calendar authorization…")
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

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start() //nolint:errcheck
	case "linux":
		exec.Command("xdg-open", url).Start() //nolint:errcheck
	}
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
