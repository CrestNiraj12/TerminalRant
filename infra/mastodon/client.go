package mastodon

import (
	"fmt"
	"io"
	"net/http"

	"terminalrant/infra/auth"
)

// Client is a thin HTTP wrapper for the Mastodon API.
// It handles base URL construction and bearer token injection.
type Client struct {
	baseURL       string
	tokenProvider auth.TokenProvider
	http          *http.Client
}

// NewClient creates a Mastodon API client.
func NewClient(baseURL string, tp auth.TokenProvider) *Client {
	return &Client{
		baseURL:       baseURL,
		tokenProvider: tp,
		http:          &http.Client{},
	}
}

// Get performs an authenticated GET request.
func (c *Client) Get(path string) ([]byte, error) {
	return c.do(http.MethodGet, path, nil)
}

// Post performs an authenticated POST request.
func (c *Client) Post(path string, body io.Reader) ([]byte, error) {
	return c.do(http.MethodPost, path, body)
}

// Put performs an authenticated PUT request.
func (c *Client) Put(path string, body io.Reader) ([]byte, error) {
	return c.do(http.MethodPut, path, body)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(path string) ([]byte, error) {
	return c.do(http.MethodDelete, path, nil)
}

func (c *Client) do(method, path string, body io.Reader) ([]byte, error) {
	token, err := c.tokenProvider.AccessToken()
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	url := c.baseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s: %w", path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API %s %s returned %d: %s", method, path, resp.StatusCode, string(data))
	}

	return data, nil
}
