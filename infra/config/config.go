package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config holds application-level configuration.
type Config struct {
	InstanceURL string // e.g. "https://mastodon.social"
	TokenPath   string // Path to file containing the access token
	Hashtag     string // Hashtag to follow, without the '#'
}

// Load reads configuration from environment variables.
//
//	TERMINALRANT_INSTANCE  — Mastodon instance URL (required)
//	TERMINALRANT_TOKEN     — Path to token file (default: ~/.config/terminalrant/token)
//	TERMINALRANT_HASHTAG   — Hashtag to follow (default: "devrant")
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

	tokenPath := os.Getenv("TERMINALRANT_TOKEN")
	if tokenPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("cannot determine home directory: %w", err)
		}
		tokenPath = filepath.Join(home, ".config", "terminalrant", "token")
	}

	hashtag := os.Getenv("TERMINALRANT_HASHTAG")
	if hashtag == "" {
		hashtag = "terminalrant"
	}

	return Config{
		InstanceURL: instance,
		TokenPath:   tokenPath,
		Hashtag:     hashtag,
	}, nil
}
