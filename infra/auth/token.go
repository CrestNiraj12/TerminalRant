package auth

import (
	"fmt"
	"os"
	"strings"
)

// TokenProvider supplies an access token for API authentication.
type TokenProvider interface {
	AccessToken() (string, error)
}

// FileTokenProvider reads a bearer token from a file on disk.
type FileTokenProvider struct {
	path string
}

// NewFileTokenProvider creates a TokenProvider that reads from the given file path.
func NewFileTokenProvider(path string) *FileTokenProvider {
	return &FileTokenProvider{path: path}
}

// AccessToken reads and returns the token, trimming whitespace.
func (f *FileTokenProvider) AccessToken() (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return "", fmt.Errorf("reading token from %s: %w", f.path, err)
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty", f.path)
	}

	return token, nil
}
