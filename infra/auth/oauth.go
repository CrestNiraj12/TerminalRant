package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"terminalrant/domain"
	"time"
)

type oauthClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// EnsureOAuthLogin guarantees a valid OAuth token exists at tokenPath.
// It validates an existing token and falls back to browser OAuth login if needed.
func EnsureOAuthLogin(ctx context.Context, instanceURL, tokenPath, clientPath string, callbackPort int) error {
	token, err := readToken(tokenPath)
	if err == nil && token != "" {
		valid, err := validateToken(ctx, instanceURL, token)
		if err != nil {
			return err
		}
		if valid {
			return nil
		}
	}

	creds, err := loadOrCreateOAuthClient(ctx, instanceURL, clientPath, callbackPort)
	if err != nil {
		return err
	}

	token, err = runOAuthAuthorization(ctx, instanceURL, creds, callbackPort)
	if err != nil {
		return err
	}

	return writeToken(tokenPath, token)
}

func validateToken(ctx context.Context, instanceURL, token string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, instanceURL+"/api/v1/accounts/verify_credentials", nil)
	if err != nil {
		return false, fmt.Errorf("creating token validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return false, fmt.Errorf("validating oauth token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("token validation failed: %d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return true, nil
}

func loadOrCreateOAuthClient(ctx context.Context, instanceURL, clientPath string, callbackPort int) (oauthClientCredentials, error) {
	if data, err := os.ReadFile(clientPath); err == nil {
		var creds oauthClientCredentials
		if err := json.Unmarshal(data, &creds); err == nil && creds.ClientID != "" && creds.ClientSecret != "" {
			return creds, nil
		}
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", callbackPort)
	form := url.Values{}
	form.Set("client_name", "TerminalRant")
	form.Set("redirect_uris", redirectURI)
	form.Set("scopes", "read write")
	form.Set("website", "https://github.com/CrestNiraj12")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, instanceURL+"/api/v1/apps", strings.NewReader(form.Encode()))
	if err != nil {
		return oauthClientCredentials{}, fmt.Errorf("creating oauth app registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return oauthClientCredentials{}, fmt.Errorf("registering oauth app: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthClientCredentials{}, fmt.Errorf("reading oauth app registration response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthClientCredentials{}, fmt.Errorf("oauth app registration failed: %d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var creds oauthClientCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return oauthClientCredentials{}, fmt.Errorf("parsing oauth app registration response: %w", err)
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return oauthClientCredentials{}, errors.New("oauth app registration returned empty client credentials")
	}

	if err := os.MkdirAll(filepath.Dir(clientPath), 0o700); err != nil {
		return oauthClientCredentials{}, fmt.Errorf("creating auth directory: %w", err)
	}
	serialized, err := json.Marshal(creds)
	if err != nil {
		return oauthClientCredentials{}, fmt.Errorf("serializing oauth client credentials: %w", err)
	}
	if err := os.WriteFile(clientPath, serialized, 0o600); err != nil {
		return oauthClientCredentials{}, fmt.Errorf("writing oauth client credentials: %w", err)
	}

	return creds, nil
}

func runOAuthAuthorization(ctx context.Context, instanceURL string, creds oauthClientCredentials, callbackPort int) (string, error) {
	state, err := randomState()
	if err != nil {
		return "", fmt.Errorf("generating oauth state: %w", err)
	}
	codeVerifier, err := randomCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("generating oauth code verifier: %w", err)
	}
	codeChallenge := codeChallengeS256(codeVerifier)
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", callbackPort)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", callbackPort)}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			select {
			case errCh <- errors.New("oauth state mismatch"):
			default:
			}
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			http.Error(w, "authorization denied", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("oauth authorization error: %s", e):
			default:
			}
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing oauth code", http.StatusBadRequest)
			select {
			case errCh <- errors.New("oauth callback missing code"):
			default:
			}
			return
		}
		_, _ = io.WriteString(w, domain.AppTitle+" login complete. You can return to the terminal.")
		select {
		case codeCh <- code:
		default:
		}
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- fmt.Errorf("oauth callback server: %w", err):
			default:
			}
		}
	}()

	authURL := instanceURL + "/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {creds.ClientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {"read write"},
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {codeChallenge},
	}.Encode()

	fmt.Printf("Opening browser for OAuth login...\nIf it does not open, visit:\n%s\n\n", authURL)
	_ = exec.Command("open", authURL).Start()

	timeout := time.NewTimer(2 * time.Minute)
	defer timeout.Stop()

	var code string
	select {
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return "", ctx.Err()
	case err := <-errCh:
		_ = srv.Shutdown(context.Background())
		return "", err
	case code = <-codeCh:
		_ = srv.Shutdown(context.Background())
	case <-timeout.C:
		_ = srv.Shutdown(context.Background())
		return "", errors.New("oauth login timed out")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("redirect_uri", redirectURI)
	form.Set("scope", "read write")
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, instanceURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating oauth token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("exchanging oauth code: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading oauth token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oauth token exchange failed: %d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var tr oauthTokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return "", fmt.Errorf("parsing oauth token response: %w", err)
	}
	if strings.TrimSpace(tr.AccessToken) == "" {
		return "", errors.New("oauth token response missing access token")
	}
	return strings.TrimSpace(tr.AccessToken), nil
}

func randomState() (string, error) {
	return randomToken(24)
}

func randomCodeVerifier() (string, error) {
	// 32 random bytes -> 43 chars with RawURLEncoding, valid PKCE verifier length.
	return randomToken(32)
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func readToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating auth directory: %w", err)
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)), 0o600)
}
