package endee

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// listIndexesResponse is the response body for the list indexes endpoint.
type listIndexesResponse struct {
	Indexes []IndexInfo `json:"indexes"`
}

// createIndexRequest is the request body for creating a new index.
type createIndexRequest struct {
	Name      string `json:"index_name"`
	Dimension int    `json:"dim"`
	SpaceType string `json:"space_type"`
	M         int    `json:"M"`
	EfCon     int    `json:"ef_con"`
	Checksum  int    `json:"checksum"`
	Precision string `json:"precision"`
	Version   int    `json:"version"`
	SparseDim int    `json:"sparse_dim"`
}

// getIndexResponse is the response body for the get index info endpoint.
type getIndexResponse struct {
	LibToken      string `json:"lib_token"`
	TotalElements int    `json:"total_elements"`
	SpaceType     string `json:"space_type"`
	Dimension     int    `json:"dimension"`
	Precision     string `json:"precision"`
	M             int    `json:"M"`
	EfCon         int    `json:"ef_con"`
	CreatedAt     int64  `json:"created_at"`
	Name          string `json:"name"`
	SparseDim     int    `json:"sparse_dim"`
}

// CreateIndex creates a new vector index with the given configuration.
func (nd *Endee) CreateIndex(name string, dimension int, spaceType string, m int, efCon int, precision string, version *int, sparseDim int) error {
	return nd.CreateIndexWithContext(context.Background(), name, dimension, spaceType, m, efCon, precision, version, sparseDim)
}

// CreateIndexWithContext creates a new vector index with context support for cancellation.
func (nd *Endee) CreateIndexWithContext(ctx context.Context, name string, dimension int, spaceType string, m int, efCon int, precision string, version *int, sparseDim int) error {
	if !isValidIndexName(name) {
		return errors.New("invalid index name. Index name must be alphanumeric and can contain underscores and less than 48 characters")
	}

	if precision == "" {
		precision = PrecisionInt16
	}

	if dimension <= 0 || dimension > MaxDimensionAllowed {
		return fmt.Errorf("dimension must be between 1 and %d", MaxDimensionAllowed)
	}

	if m <= 0 {
		return fmt.Errorf("M must be greater than 0")
	}

	if efCon <= 0 {
		return fmt.Errorf("ef_con must be greater than 0")
	}

	spaceType = strings.ToLower(spaceType)
	if !validSpaceTypes[spaceType] {
		return fmt.Errorf("invalid space type: %s", spaceType)
	}

	validPrecision := false

	for _, p := range PrecisionTypesSupported {
		if p == precision {
			validPrecision = true

			break
		}
	}

	if !validPrecision {
		return fmt.Errorf("invalid precision: %s. Must be one of: %v", precision, PrecisionTypesSupported)
	}

	if sparseDim < 0 {
		return fmt.Errorf("sparse_dim must be non-negative")
	}

	finalVersion := 1
	if version != nil {
		finalVersion = *version
	}

	requestData := createIndexRequest{
		Name:      name,
		Dimension: dimension,
		SpaceType: spaceType,
		M:         m,
		EfCon:     efCon,
		Checksum:  Checksum,
		Precision: precision,
		Version:   finalVersion,
		SparseDim: sparseDim,
	}

	jsonData, err := fastJSONMarshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", nd.buildURL("/index/create"), bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	return checkError(resp)
}

// ListIndexes returns all indexes in the database.
func (nd *Endee) ListIndexes() ([]IndexInfo, error) {
	return nd.ListIndexesWithContext(context.Background())
}

// ListIndexesWithContext returns all indexes with context support for cancellation.
func (nd *Endee) ListIndexesWithContext(ctx context.Context) ([]IndexInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", nd.buildURL("/index/list"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %d - %s", resp.StatusCode, resp.Status)
	}

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response listIndexesResponse

	if err := fastJSONUnmarshal(buf.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Indexes == nil {
		return []IndexInfo{}, nil
	}

	return response.Indexes, nil
}

// DeleteIndex deletes the named index.
func (nd *Endee) DeleteIndex(name string) error {
	return nd.DeleteIndexWithContext(context.Background(), name)
}

// DeleteIndexWithContext deletes the named index with context support for cancellation.
func (nd *Endee) DeleteIndexWithContext(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", nd.buildURL(fmt.Sprintf("/index/%s/delete", name)), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	return checkError(resp)
}

// GetIndex retrieves an index by name and returns a handle for vector operations.
func (nd *Endee) GetIndex(name string) (*Index, error) {
	return nd.GetIndexWithContext(context.Background(), name)
}

// GetIndexWithContext retrieves an index by name with context support for cancellation.
func (nd *Endee) GetIndexWithContext(ctx context.Context, name string) (*Index, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", nd.buildURL(fmt.Sprintf("/index/%s/info", name)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := nd.executeRequestWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkError(resp); err != nil {
		return nil, err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var data getIndexResponse

	if err := fastJSONUnmarshal(buf.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	params := &IndexParams{
		LibToken:      data.LibToken,
		TotalElements: data.TotalElements,
		SpaceType:     data.SpaceType,
		Dimension:     data.Dimension,
		SparseDim:     data.SparseDim,
		Precision:     data.Precision,
		M:             data.M,
		EfCon:         data.EfCon,
	}

	return NewIndex(name, nd.token, nd.baseURL, 1, params), nil
}
