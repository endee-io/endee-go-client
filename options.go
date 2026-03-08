package endee

import "net/http"

// Option is a functional option for configuring an Endee client.
type Option func(*clientConfig)

type clientConfig struct {
	token      string
	baseURL    string
	baseURLSet bool
	httpClient *http.Client
}

// WithToken sets the authentication token for the client.
// Cloud tokens have the format "prefix:secret:region" — the region is
// automatically used to derive the correct base URL.
func WithToken(token string) Option {
	return func(c *clientConfig) {
		c.token = token
	}
}

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) Option {
	return func(c *clientConfig) {
		c.baseURL = url
		c.baseURLSet = true
	}
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}
