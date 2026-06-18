package commands

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/rootlyhq/rootly-catalog-sync/client"
	"github.com/rootlyhq/rootly-catalog-sync/oauth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Rootly via browser-based OAuth2",
	Long: `Opens your browser to authenticate with Rootly using OAuth2 Authorization Code + PKCE.

No configuration is needed — just run "rootly-catalog-sync login" and follow the browser prompts.
By default, connects to api.rootly.com. Set ROOTLY_API_URL to target a different environment.`,
	Example: `  # Login to Rootly (production)
  rootly-catalog-sync login

  # Login to a local dev server
  ROOTLY_API_URL=http://localhost:22166 rootly-catalog-sync login`,
	RunE: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	apiURL := os.Getenv("ROOTLY_API_URL")
	if apiURL == "" {
		apiURL = client.DefaultBaseURL
	}

	authBaseURL := oauth.DeriveAuthBaseURL(apiURL)

	// Register the OAuth client (or use cached registration)
	clientID, scopes := oauth.LoadCachedRegistration()
	if clientID == "" {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Registering OAuth client...\n")
		var err error
		clientID, scopes, err = oauth.RegisterClient(ctx, authBaseURL)
		if err != nil {
			return err
		}
		if saveErr := oauth.SaveRegistration(clientID, scopes); saveErr != nil {
			return fmt.Errorf("failed to cache registration: %w", saveErr)
		}
	}

	tok, err := doOAuthFlow(ctx, cmd, authBaseURL, clientID, scopes)
	if err != nil {
		return err
	}

	if err := oauth.SaveOAuth2Token(tok); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Login successful! Tokens saved.\n")
	return nil
}

func doOAuthFlow(ctx context.Context, cmd *cobra.Command, authBaseURL, clientID string, scopes []string) (*oauth2.Token, error) {
	cfg := oauth.NewConfig(authBaseURL, clientID, scopes)

	verifier := oauth2.GenerateVerifier()

	state, err := oauth.GenerateState()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Build authorization URL with PKCE (S256 challenge derived from verifier)
	authURL := cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	// Channel to receive the authorization code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local callback server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("authorization error: %s — %s", errMsg, desc)
			_, _ = fmt.Fprintf(w, "<html><body><h1>Authorization Failed</h1><p>%s</p><p>You can close this window.</p></body></html>", html.EscapeString(desc))
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		_, _ = fmt.Fprint(w, "<html><body><h1>Login Successful!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	lc := net.ListenConfig{}
	listener, err := lc.Listen(ctx, "tcp", "localhost:"+oauth.CallbackPort)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %s: %w", oauth.CallbackPort, err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Opening browser for authentication...\n")
	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "If the browser doesn't open, visit:\n%s\n\n", authURL)

	if err := openBrowser(ctx, authURL); err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Failed to open browser: %v\n", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStderr(), "Waiting for authorization...\n")

	// Wait for callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("login timed out after 5 minutes")
	}

	// Exchange code for tokens
	tok, err := oauth.ExchangeCode(ctx, cfg, code, verifier)
	if err != nil {
		return nil, err
	}

	return tok, nil
}

func openBrowser(ctx context.Context, url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "open", url).Start()
	case "linux":
		return exec.CommandContext(ctx, "xdg-open", url).Start()
	case "windows":
		return exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
