package endee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Valid space types
var validSpaceTypes = map[string]bool{
	"cosine": true,
	"l2":     true,
	"ip":     true,
}

// Advanced memory pools for various objects
var (
	// Buffer pool for reusing byte buffers
	bufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 1024) // Pre-allocate 1KB
			return bytes.NewBuffer(buf)
		},
	}

	// Slice pools for vector data
	float32SlicePool = sync.Pool{
		New: func() interface{} {
			return make([]float32, 0, 1024) // Pre-allocate for typical vector sizes
		},
	}

	// Interface slice pool for msgpack operations
	interfaceSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]interface{}, 0, 100)
		},
	}

	// String slice pool for batch operations
	stringSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 100)
		},
	}

	// Map pool for metadata and filters
	mapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 10)
		},
	}

	// JSON encoder & decoder pool for streaming operations
	jsonEncoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewEncoder(&bytes.Buffer{})
		},
	}

	jsonDecoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewDecoder(strings.NewReader(""))
		},
	}
)

type Endee struct {
	BaseUrl string
	Token   string
	HTTP    *http.Client
}

type ListIndexesResponse struct {
	Indexes []interface{} `json:"indixes"`
}

type CreateIndexRequest struct {
	IndexName string `json:"index_name"`
	Dim       int    `json:"dim"`
	SpaceType string `json:"space_type"`
	M         int    `json:"M"`
	EfCon     int    `json:"ef_con"`
	SparseDim int    `json:"sparse_dim,omitempty"`
	Checksum  int    `json:"checksum"`
	UseInt8d  bool   `json:"use_int8d"`
	Version   *int   `json:"version,omitempty"`
}

// isValidIndexName validates that the index name is alphanumeric with underscores and less than 48 characters
func isValidIndexName(name string) bool {
	if len(name) == 0 || len(name) >= MaxIndexNameLenAllowed {
		return false
	}
	return NameRegex.MatchString(name)
}

// Advanced pool management functions

// getBuffer gets a buffer from the pool
func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

// putBuffer returns a buffer to the pool after resetting it
func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// getFloat32Slice gets a float32 slice from the pool
func getFloat32Slice() []float32 {
	return float32SlicePool.Get().([]float32)[:0]
}

// putFloat32Slice returns a float32 slice to the pool
func putFloat32Slice(slice []float32) {
	if cap(slice) > 0 {
		float32SlicePool.Put(slice[:0])
	}
}

// getInterfaceSlice gets an interface slice from the pool
func getInterfaceSlice() []interface{} {
	return interfaceSlicePool.Get().([]interface{})[:0]
}

// putInterfaceSlice returns an interface slice to the pool
func putInterfaceSlice(slice []interface{}) {
	if cap(slice) > 0 {
		interfaceSlicePool.Put(slice[:0])
	}
}

// getStringSlice gets a string slice from the pool
func getStringSlice() []string {
	return stringSlicePool.Get().([]string)[:0]
}

// putStringSlice returns a string slice to the pool
func putStringSlice(slice []string) {
	if cap(slice) > 0 {
		stringSlicePool.Put(slice[:0])
	}
}

// getMap gets a map from the pool
func getMap() map[string]interface{} {
	m := mapPool.Get().(map[string]interface{})
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// putMap returns a map to the pool
func putMap(m map[string]interface{}) {
	if m != nil && len(m) < 100 {
		mapPool.Put(m)
	}
}

// getJSONEncoder gets a JSON encoder from the pool
func getJSONEncoder(w *bytes.Buffer) *json.Encoder {
	enc := jsonEncoderPool.Get().(*json.Encoder)
	// Reset the encoder's writer
	enc = json.NewEncoder(w)
	return enc
}

// putJSONEncoder returns a JSON encoder to the pool
func putJSONEncoder(enc *json.Encoder) {
	jsonEncoderPool.Put(enc)
}

// getJSONDecoder gets a JSON decoder from the pool
func getJSONDecoder(r *bytes.Reader) *json.Decoder {
	dec := jsonDecoderPool.Get().(*json.Decoder)
	// Reset the decoder's reader
	dec = json.NewDecoder(r)
	return dec
}

// putJSONDecoder returns a JSON decoder to the pool
func putJSONDecoder(dec *json.Decoder) {
	jsonDecoderPool.Put(dec)
}

// buildURL efficiently builds API URLs
func (nd *Endee) buildURL(path string) string {
	var builder strings.Builder
	builder.Grow(len(nd.BaseUrl) + len(path) + 1)
	builder.WriteString(nd.BaseUrl)
	if !strings.HasSuffix(nd.BaseUrl, "/") && !strings.HasPrefix(path, "/") {
		builder.WriteString("/")
	}
	builder.WriteString(path)
	return builder.String()
}

// EndeeClient creates an optimized client. token is optional.
func EndeeClient(token ...string) *Endee {
	baseUrl := LocalBaseURL
	var finalToken string

	// Handle optional token logic
	if len(token) > 0 && token[0] != "" {
		t := token[0]
		tokenParts := strings.Split(t, ":")

		if len(tokenParts) > 2 {
			// Extract region from 3rd part of token for Cloud URL
			baseUrl = fmt.Sprintf(CloudURLTemplate, tokenParts[2])
			finalToken = fmt.Sprintf("%s:%s", tokenParts[0], tokenParts[1])
		} else {
			finalToken = t
		}
	}

	// High-performance transport configuration
	transport := &http.Transport{
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
		DisableCompression:    true, // Optimized for Msgpack/Binary
	}

	return &Endee{
		BaseUrl: baseUrl,
		Token:   finalToken,
		HTTP: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
		},
	}
}

// executeRequestWithContext executes HTTP requests with context for cancellation and timeout
func (nd *Endee) executeRequestWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", nd.Token)

	resp, err := nd.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// fastJSONMarshal uses streaming JSON encoder for better performance
func fastJSONMarshal(v interface{}) ([]byte, error) {
	buf := getBuffer()
	defer putBuffer(buf)

	enc := getJSONEncoder(buf)
	defer putJSONEncoder(enc)

	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	// Remove trailing newline that json.Encoder adds
	data := buf.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	// Copy data since we're returning the buffer to pool
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// fastJSONUnmarshal uses streaming JSON decoder for better performance
func fastJSONUnmarshal(data []byte, v interface{}) error {
	reader := bytes.NewReader(data)
	dec := getJSONDecoder(reader)
	defer putJSONDecoder(dec)

	return dec.Decode(v)
}

// readResponseBody reads the response body and handles errors
func readResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if err := checkError(resp); err != nil {
		return nil, err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	buf.ReadFrom(resp.Body)
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func (nd *Endee) CreateIndex(name string, dimension int, spaceType string, M int, efCon int, useFp16 bool, version *int, sparseDim int) error {
	return nd.CreateIndexWithContext(context.Background(), name, dimension, spaceType, M, efCon, useFp16, version, sparseDim)
}

// CreateIndexWithContext creates an index with context support for cancellation
func (nd *Endee) CreateIndexWithContext(ctx context.Context, name string, dimension int, spaceType string, M int, efCon int, useFp16 bool, version *int, sparseDim int) error {
	// Validate index name
	if !isValidIndexName(name) {
		return errors.New("invalid index name. Index name must be alphanumeric and can contain underscores and less than 48 characters")
	}

	// Validate dimension
	if dimension > MaxDimensionAllowed {
		return fmt.Errorf("dimension cannot be greater than %d", MaxDimensionAllowed)
	}

	// Validate and normalize space type
	spaceType = strings.ToLower(spaceType)
	if !validSpaceTypes[spaceType] {
		return fmt.Errorf("invalid space type: %s", spaceType)
	}

	// Create request payload
	requestData := CreateIndexRequest{
		IndexName: name,
		Dim:       dimension,
		SpaceType: spaceType,
		M:         M,
		EfCon:     efCon,
		Checksum:  Checksum,
		UseInt8d:  !useFp16,
		Version:   version,
		SparseDim: sparseDim,
	}

	// Marshal JSON using fast streaming encoder
	jsonData, err := fastJSONMarshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", nd.buildURL("/index/create"), bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Execute request with context
	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	_, err = readResponseBody(resp)
	return err
}

func (nd *Endee) ListIndexes() ([]interface{}, error) {
	return nd.ListIndexesWithContext(context.Background())
}

// ListIndexesWithContext lists indexes with context support for cancellation
func (nd *Endee) ListIndexesWithContext(ctx context.Context) ([]interface{}, error) {
	req, err := http.NewRequest("GET", nd.buildURL("/index/list"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %d - %s", resp.StatusCode, resp.Status)
	}

	// Use buffer pool for reading response
	buf := getBuffer()
	defer putBuffer(buf)

	buf.ReadFrom(resp.Body)

	var response ListIndexesResponse
	if err := fastJSONUnmarshal(buf.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Ensure we never return nil slice, return empty slice instead
	if response.Indexes == nil {
		return []interface{}{}, nil
	}

	return response.Indexes, nil
}

func (nd *Endee) DeleteIndex(name string) error {
	return nd.DeleteIndexWithContext(context.Background(), name)
}

// DeleteIndexWithContext deletes an index with context support for cancellation
func (nd *Endee) DeleteIndexWithContext(ctx context.Context, name string) error {
	req, err := http.NewRequest("DELETE", nd.buildURL(fmt.Sprintf("/index/%s/delete", name)), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	_, err = readResponseBody(resp)
	return err
}

// GetIndexResponse represents the response from the /index/{name}/info endpoint
type GetIndexResponse struct {
	LibToken      string `json:"lib_token"`
	TotalElements int    `json:"total_elements"`
	SpaceType     string `json:"space_type"`
	Dimension     int    `json:"dimension"`
	UseFp16       bool   `json:"use_fp16"`
	M             int    `json:"M"`
	Checksum      int    `json:"checksum"`
	CreatedAt     int64  `json:"created_at"`
	Name          string `json:"name"`
}

func (nd *Endee) GetIndex(name string) (*Index, error) {
	return nd.GetIndexWithContext(context.Background(), name)
}

// GetIndexWithContext gets an index with context support for cancellation
func (nd *Endee) GetIndexWithContext(ctx context.Context, name string) (*Index, error) {
	req, err := http.NewRequest("GET", nd.buildURL(fmt.Sprintf("/index/%s/info", name)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}

	var data GetIndexResponse
	if err := fastJSONUnmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create IndexParams from response data
	params := &IndexParams{
		LibToken:      data.LibToken,
		TotalElements: data.TotalElements,
		SpaceType:     data.SpaceType,
		Dimension:     data.Dimension,
		Precision:     PrecisionInt8D, // Default fallback
		M:             data.M,
	}
	if data.UseFp16 {
		params.Precision = PrecisionFloat16
	}

	// Create and return Index object
	index := NewIndex(name, nd.Token, nd.BaseUrl, 1, params)
	return index, nil
}
