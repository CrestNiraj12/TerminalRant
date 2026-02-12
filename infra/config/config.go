package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds application-level configuration.
type Config struct {
	InstanceURL       string // e.g. "https://mastodon.social"
	OAuthTokenPath    string // Path where OAuth access token is stored
	OAuthClientPath   string // Path where OAuth client credentials are stored
	OAuthCallbackPort int    // Local callback port for OAuth login
	Hashtag           string // Hashtag to follow, without the '#'
}

// Load reads configuration from environment variables.
//
//	TERMINALRANT_INSTANCE            — Mastodon instance URL
//	TERMINALRANT_AUTH_DIR            — Directory for OAuth token/client state
//	TERMINALRANT_OAUTH_CALLBACK_PORT — Local callback port for OAuth login
//	TERMINALRANT_HASHTAG             — Hashtag to follow
func Load() (Config, error) {
	instance := os.Getenv("TERMINALRANT_INSTANCE")
	if instance == "" {
		instance = "https://mastodon.social"
	}
	parsed, err := url.Parse(instance)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Config{}, fmt.Errorf("invalid TERMINALRANT_INSTANCE: must be an absolute URL")
	}
	if parsed.Scheme != "https" {
		return Config{}, fmt.Errorf("invalid TERMINALRANT_INSTANCE: only https is allowed")
	}
	instance = strings.TrimRight(parsed.String(), "/")

	authDir := os.Getenv("TERMINALRANT_AUTH_DIR")
	if authDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("cannot determine home directory: %w", err)
		}
		authDir = filepath.Join(home, ".config", "terminalrant")
	}

	callbackPort := 45145
	if p := os.Getenv("TERMINALRANT_OAUTH_CALLBACK_PORT"); p != "" {
		parsedPort, err := strconv.Atoi(p)
		if err != nil || parsedPort < 1024 || parsedPort > 65535 {
			return Config{}, fmt.Errorf("invalid TERMINALRANT_OAUTH_CALLBACK_PORT: must be 1024-65535")
		}
		callbackPort = parsedPort
	}

	hashtag := os.Getenv("TERMINALRANT_HASHTAG")
	if hashtag == "" {
		hashtag = "terminalrant"
	}

	return Config{
		InstanceURL:       instance,
		OAuthTokenPath:    filepath.Join(authDir, "oauth_token"),
		OAuthClientPath:   filepath.Join(authDir, "oauth_client.json"),
		OAuthCallbackPort: callbackPort,
		Hashtag:           hashtag,
	}, nil
}
