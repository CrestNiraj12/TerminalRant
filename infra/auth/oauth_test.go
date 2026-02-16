package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func withMockDefaultTransport(t *testing.T, rt roundTripFunc) {
	t.Helper()
	prev := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() { http.DefaultTransport = prev })
}

func response(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestValidateToken_StatusHandling(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantValid bool
		wantErr   bool
	}{
		{name: "ok", status: http.StatusOK, body: "{}", wantValid: true, wantErr: false},
		{name: "unauthorized", status: http.StatusUnauthorized, body: "{}", wantValid: false, wantErr: false},
		{name: "server error", status: http.StatusInternalServerError, body: "boom", wantValid: false, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var authHeader string
			withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/v1/accounts/verify_credentials" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				authHeader = r.Header.Get("Authorization")
				return response(r, tc.status, tc.body), nil
			}))

			valid, err := validateToken(context.Background(), "http://example.test", "tok123")
			if authHeader != "Bearer tok123" {
				t.Fatalf("missing bearer token header: %q", authHeader)
			}
			if valid != tc.wantValid {
				t.Fatalf("valid mismatch got=%v want=%v", valid, tc.wantValid)
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("err mismatch got=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestLoadOrCreateOAuthClient_ReadsCachedCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oauth_client.json")
	cached := oauthClientCredentials{ClientID: "cid", ClientSecret: "sec"}
	raw, _ := json.Marshal(cached)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write cached creds failed: %v", err)
	}

	creds, err := loadOrCreateOAuthClient(context.Background(), "https://example.invalid", path, 45145)
	if err != nil {
		t.Fatalf("load cached creds failed: %v", err)
	}
	if creds != cached {
		t.Fatalf("cached creds mismatch got=%#v want=%#v", creds, cached)
	}
}

func TestLoadOrCreateOAuthClient_RegistersAndPersists(t *testing.T) {
	var gotContentType string
	var gotValues url.Values

	withMockDefaultTransport(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/apps" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotValues, _ = url.ParseQuery(string(body))
		resp, _ := json.Marshal(oauthClientCredentials{ClientID: "newid", ClientSecret: "newsecret"})
		return response(r, http.StatusOK, string(resp)), nil
	}))

	path := filepath.Join(t.TempDir(), "auth", "oauth_client.json")
	creds, err := loadOrCreateOAuthClient(context.Background(), "http://example.test", path, 45145)
	if err != nil {
		t.Fatalf("register oauth app failed: %v", err)
	}
	if creds.ClientID != "newid" || creds.ClientSecret != "newsecret" {
		t.Fatalf("unexpected creds: %#v", creds)
	}
	if !strings.Contains(gotContentType, "application/x-www-form-urlencoded") {
		t.Fatalf("expected form content-type, got %q", gotContentType)
	}
	if gotValues.Get("client_name") != "TerminalRant" || gotValues.Get("redirect_uris") == "" || gotValues.Get("scopes") != "read write" {
		t.Fatalf("unexpected registration form values: %#v", gotValues)
	}

	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected persisted credentials file: %v", err)
	}
	var parsed oauthClientCredentials
	if err := json.Unmarshal(persisted, &parsed); err != nil {
		t.Fatalf("parse persisted creds failed: %v", err)
	}
	if parsed != creds {
		t.Fatalf("persisted creds mismatch got=%#v want=%#v", parsed, creds)
	}
}

func TestStateAndTokenHelpers(t *testing.T) {
	state, err := randomState()
	if err != nil {
		t.Fatalf("randomState failed: %v", err)
	}
	if len(state) < 20 {
		t.Fatalf("unexpectedly short state: %q", state)
	}
	if strings.ContainsAny(state, " \n\t") {
		t.Fatalf("state should not include whitespace: %q", state)
	}

	path := filepath.Join(t.TempDir(), "auth", "token")
	if err := writeToken(path, "  my-token \n"); err != nil {
		t.Fatalf("writeToken failed: %v", err)
	}
	got, err := readToken(path)
	if err != nil {
		t.Fatalf("readToken failed: %v", err)
	}
	if got != "my-token" {
		t.Fatalf("unexpected read token: %q", got)
	}
}

func TestPKCEHelpers(t *testing.T) {
	verifier, err := randomCodeVerifier()
	if err != nil {
		t.Fatalf("randomCodeVerifier failed: %v", err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("invalid code verifier length: %d", len(verifier))
	}
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Fatalf("verifier is not base64url: %v", err)
	}

	// RFC 7636 Appendix B example.
	challenge := codeChallengeS256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if challenge != want {
		t.Fatalf("unexpected code challenge: got %q want %q", challenge, want)
	}
}
