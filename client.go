package endee

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// Endee is the main client for the Endee vector database.
type Endee struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient creates a new Endee client with the given options.
// If no WithBaseURL option is provided, the base URL is derived from the token.
func NewClient(opts ...Option) *Endee {
	cfg := &clientConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.token != "" {
		parsedToken, parsedBaseURL := parseToken(cfg.token)
		cfg.token = parsedToken

		if !cfg.baseURLSet {
			cfg.baseURL = parsedBaseURL
		}
	} else if cfg.baseURL == "" {
		cfg.baseURL = LocalBaseURL
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout:   DefaultTimeout,
			Transport: defaultTransport(),
		}
	}

	return &Endee{
		baseURL: cfg.baseURL,
		token:   cfg.token,
		http:    httpClient,
	}
}

// EndeeClient creates an Endee client with an optional authentication token.
//
// Deprecated: Use NewClient with WithToken instead.
//
//nolint:revive // intentional legacy name kept for backward compatibility
func EndeeClient(token ...string) *Endee {
	if len(token) > 0 && token[0] != "" {
		return NewClient(WithToken(token[0]))
	}

	return NewClient()
}

// parseToken extracts the clean token and derives the API base URL.
// Cloud tokens have the format "prefix:secret:region".
func parseToken(raw string) (token, baseURL string) {
	parts := strings.Split(raw, ":")
	if len(parts) > 2 {
		baseURL = fmt.Sprintf(CloudURLTemplate, parts[2])
		token = fmt.Sprintf("%s:%s", parts[0], parts[1])
	} else {
		token = raw
		baseURL = LocalBaseURL
	}

	return token, baseURL
}

// defaultTransport returns a high-performance HTTP transport.
func defaultTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        runtime.NumCPU() * 20,
		MaxIdleConnsPerHost: runtime.NumCPU() * 4,
		MaxConnsPerHost:     runtime.NumCPU() * 10,
		IdleConnTimeout:     120 * time.Second,

		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,

		ForceAttemptHTTP2:     true,
		WriteBufferSize:       32 * 1024,
		ReadBufferSize:        32 * 1024,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
}

// buildURL efficiently builds an absolute API URL from the given path.
func (nd *Endee) buildURL(path string) string {
	var builder strings.Builder

	builder.Grow(len(nd.baseURL) + len(path) + 1)
	builder.WriteString(nd.baseURL)

	if !strings.HasSuffix(nd.baseURL, "/") && !strings.HasPrefix(path, "/") {
		builder.WriteString("/")
	}

	builder.WriteString(path)

	return builder.String()
}

// executeRequestWithContext executes an HTTP request with the given context,
// attaching the client's auth token.
func (nd *Endee) executeRequestWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", nd.token)

	resp, err := nd.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// fastJSONMarshal serializes v to JSON using a pooled streaming encoder.
func fastJSONMarshal(v interface{}) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	enc := getJSONEncoder(buf)
	defer putJSONEncoder(enc)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	// Remove trailing newline added by json.Encoder.
	data := buf.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}

// fastJSONUnmarshal deserializes JSON data into v using a pooled streaming decoder.
func fastJSONUnmarshal(data []byte, v interface{}) error {
	reader := bytes.NewReader(data)
	dec := getJSONDecoder(reader)
	defer putJSONDecoder(dec)

	return dec.Decode(v)
}
