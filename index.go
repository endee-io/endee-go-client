package endee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"strings"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

// Index represents a Endee index with its properties and configuration
type Index struct {
	Name      string
	Token     string
	URL       string
	Version   int
	Checksum  int
	LibToken  string
	Count     int
	SpaceType string
	Dimension int
	SparseDim int
	Precision string
	M         int
}

// IndexParams represents the parameters passed to create an Index
type IndexParams struct {
	LibToken      string `json:"lib_token"`
	TotalElements int    `json:"total_elements"`
	SpaceType     string `json:"space_type"`
	Dimension     int    `json:"dimension"`
	SparseDim     int    `json:"sparse_dim"`
	Precision     string `json:"precision"`
	M             int    `json:"M"`
}

// VectorItem represents a vector with metadata for upserting
type VectorItem struct {
	ID            string                 `json:"id"`
	Vector        []float32              `json:"vector"`
	SparseIndices []int                  `json:"sparse_indices,omitempty"`
	SparseValues  []float32              `json:"sparse_values,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Filter        map[string]interface{} `json:"filter,omitempty"`
}

// VectorObject represents the internal structure for API submission
type VectorObject struct {
	ID     string    `json:"id"`
	Meta   string    `json:"meta"`
	Filter string    `json:"filter"`
	Norm   float32   `json:"norm"`
	Vector []float32 `json:"vector"`
}

// QueryResult represents a single search result
type QueryResult struct {
	ID         string                 `json:"id"`
	Similarity float32                `json:"similarity"`
	Distance   float32                `json:"distance"`
	Meta       map[string]interface{} `json:"meta"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Norm       float32                `json:"norm"`
	Vector     []float32              `json:"vector,omitempty"`
}

// QueryRequest represents the search request payload
type QueryRequest struct {
	Vector         []float32 `json:"vector,omitempty"`
	SparseIndices  []int     `json:"sparse_indices,omitempty"`
	SparseValues   []float32 `json:"sparse_values,omitempty"`
	TopK           int       `json:"k"`
	Ef             int       `json:"ef"`
	IncludeVectors bool      `json:"include_vectors"`
	Filter         string    `json:"filter,omitempty"`
}

// NewIndex creates a new Index instance similar to Python's __init__
func NewIndex(name string, token string, url string, version int, params *IndexParams) *Index {
	if version == 0 {
		version = 1 // Default version
	}

	precision := PrecisionInt8D
	if params != nil && params.Precision != "" {
		precision = params.Precision
	}

	index := &Index{
		Name:    name,
		Token:   token,
		URL:     url,
		Version: version,
	}

	index.Checksum = Checksum

	// Set parameters if provided
	if params != nil {
		index.LibToken = params.LibToken
		index.Count = params.TotalElements
		index.SpaceType = params.SpaceType
		index.Dimension = params.Dimension
		index.SparseDim = params.SparseDim
		index.Precision = precision
		index.M = params.M
	}

	return index
}

// buildURL efficiently builds API URLs for index operations
func (idx *Index) buildURL(path string) string {
	var builder strings.Builder
	builder.Grow(len(idx.URL) + len(path) + len(idx.Name) + 10) // Extra space for /index/ and separator
	builder.WriteString(idx.URL)
	if !strings.HasSuffix(idx.URL, "/") {
		builder.WriteString("/")
	}
	if strings.Contains(path, "%s") {
		// Path contains placeholder for index name
		return fmt.Sprintf(builder.String()+path, idx.Name)
	} else {
		// Simple concatenation
		builder.WriteString(path)
		return builder.String()
	}
}

// executeRequest executes HTTP requests with consistent headers and error handling
func (idx *Index) executeRequest(method, path string, body []byte, contentType string) (*http.Response, error) {
	return idx.executeRequestWithContext(context.Background(), method, path, body, contentType)
}

// executeRequestWithContext executes HTTP requests with context support
func (idx *Index) executeRequestWithContext(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, idx.buildURL(path), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req = req.WithContext(ctx)
	req.Header.Set("Authorization", idx.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// normalizeVector normalizes a vector for cosine similarity if needed
func (idx *Index) normalizeVector(vector []float32) ([]float32, float32, error) {
	// Check dimension of the vector
	if len(vector) != idx.Dimension {
		return nil, 0, fmt.Errorf("vector dimension mismatch: expected %d, got %d",
			idx.Dimension, len(vector))
	}

	// Early return for non-cosine spaces
	if idx.SpaceType != "cosine" {
		return vector, 1.0, nil
	}

	var sum float32
	for _, v := range vector {
		sum += v * v
	}
	norm := float32(math.Sqrt(float64(sum)))

	// Handle zero norm case
	if norm == 0 {
		return vector, 1.0, nil
	}

	normalizedVector := make([]float32, len(vector))
	copy(normalizedVector, vector)

	for i := range normalizedVector {
		normalizedVector[i] /= norm
	}

	return normalizedVector, norm, nil
}

// Upsert inserts or updates vectors in the index
func (idx *Index) Upsert(inputArray []VectorItem) error {
	return idx.UpsertWithContext(context.Background(), inputArray)
}

// UpsertWithContext inserts or updates vectors with context support and concurrent processing
func (idx *Index) UpsertWithContext(ctx context.Context, inputArray []VectorItem) error {
	if len(inputArray) > MaxVectorsPerBatch {
		return fmt.Errorf("cannot insert more than %d vectors at a time", MaxVectorsPerBatch)
	}

	if len(inputArray) == 0 {
		return nil
	}

	// Validate each vector item
	for i, item := range inputArray {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("id must not be empty (item index %d)", i)
		}

		// Sparse data validation: both must be present or both nil/empty
		hasIndices := len(item.SparseIndices) > 0
		hasValues := len(item.SparseValues) > 0

		if hasIndices != hasValues {
			return fmt.Errorf("sparse_indices and sparse_values must both be provided together (item id: %s)", item.ID)
		}

		if hasIndices && len(item.SparseIndices) != len(item.SparseValues) {
			return fmt.Errorf("sparse_indices and sparse_values must have the same length (item id: %s)", item.ID)
		}
	}

	// For small batches, use sequential processing
	if len(inputArray) <= 10 {
		return idx.upsertSequential(ctx, inputArray)
	}

	// For larger batches, use concurrent processing
	return idx.upsertConcurrent(ctx, inputArray)
}

// upsertSequential processes vectors sequentially for small batches
func (idx *Index) upsertSequential(ctx context.Context, inputArray []VectorItem) error {
	// Pre-allocate slice with known capacity
	vectorBatch := make([][]interface{}, 0, len(inputArray))

	for _, item := range inputArray {
		// Normalize vector
		normalizedVector, norm, err := idx.normalizeVector(item.Vector)
		if err != nil {
			return err
		}

		// Serialize metadata using JsonZip (zlib compressed)
		metaBytes, err := JsonZip(item.Meta)
		if err != nil {
			return fmt.Errorf("failed to compress metadata: %v", err)
		}

		// Serialize filter
		filterBytes, err := json.Marshal(item.Filter)
		if err != nil {
			return fmt.Errorf("failed to serialize filter: %v", err)
		}

		// Create vector object as array (matching Python structure)
		vectorObj := []interface{}{
			item.ID,             // str(item.get('id', ''))
			metaBytes,           // meta_data (compressed bytes)
			string(filterBytes), // json.dumps(item.get('filter', {}))
			norm,                // float(norms[i])
			normalizedVector,    // normalizedVector[i].tolist()
		}

		// Add sparse vectors if present and index is hybrid-capable (SparseDim > 0)
		// Or just if sparse vectors are present in the item, assume user knows what they are doing
		if len(item.SparseIndices) > 0 && len(item.SparseValues) > 0 {
			vectorObj = append(vectorObj, item.SparseIndices, item.SparseValues)
		}

		vectorBatch = append(vectorBatch, vectorObj)
	}

	// Serialize data using msgpack (matching Python implementation)
	serializedData, err := msgpack.Marshal(vectorBatch)
	if err != nil {
		return fmt.Errorf("failed to serialize vector batch: %w", err)
	}

	// Execute request using helper method with context
	resp, err := idx.executeRequestWithContext(ctx, "POST", "index/%s/vector/insert", serializedData, "application/msgpack")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if err := checkError(resp); err != nil {
		return err
	}

	return nil
}

// upsertConcurrent processes vectors concurrently for large batches
func (idx *Index) upsertConcurrent(ctx context.Context, inputArray []VectorItem) error {
	// Determine optimal batch size and worker count
	numWorkers := runtime.NumCPU()
	if len(inputArray) < numWorkers*2 {
		numWorkers = (len(inputArray) + 1) / 2
	}

	batchSize := (len(inputArray) + numWorkers - 1) / numWorkers
	if batchSize > 100 {
		batchSize = 100 // Limit batch size to avoid memory issues
	}

	// Channel for work distribution
	workChan := make(chan []VectorItem, numWorkers)
	resultChan := make(chan error, numWorkers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range workChan {
				if len(batch) > 0 {
					err := idx.upsertSequential(ctx, batch)
					resultChan <- err
				}
			}
		}()
	}

	// Distribute work
	go func() {
		defer close(workChan)
		for i := 0; i < len(inputArray); i += batchSize {
			end := i + batchSize
			if end > len(inputArray) {
				end = len(inputArray)
			}

			select {
			case workChan <- inputArray[i:end]:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and check for errors
	var errors []error
	for err := range resultChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("upsert failed: %v", errors[0])
	}

	return nil
}

// String implements the fmt.Stringer interface, equivalent to Python's __str__
func (i *Index) String() string {
	return i.Name
}

// GetInfo returns a formatted string with index information for debugging
func (i *Index) GetInfo() string {
	return fmt.Sprintf("Index{Name: %s, Dimension: %d, SparseDim: %d, SpaceType: %s, Count: %d, Precision: %s, M: %d}",
		i.Name, i.Dimension, i.SparseDim, i.SpaceType, i.Count, i.Precision, i.M)
}

func (i *Index) Query(vector []float32, sparseIndices []int, sparseValues []float32, k int, filter map[string]interface{}, ef int, includeVectors bool) ([]QueryResult, error) {
	return i.QueryWithContext(context.Background(), vector, sparseIndices, sparseValues, k, filter, ef, includeVectors)
}

// QueryWithContext performs vector similarity search with context support
func (i *Index) QueryWithContext(ctx context.Context, vector []float32, sparseIndices []int, sparseValues []float32, k int, filter map[string]interface{}, ef int, includeVectors bool) ([]QueryResult, error) {
	// Validate parameters
	if k <= 0 || k > MaxTopKAllowed {
		return nil, fmt.Errorf("top_k must be between 1 and %d", MaxTopKAllowed)
	}
	if ef < 0 || ef > MaxEfSearchAllowed {
		return nil, fmt.Errorf("ef must be between 0 and %d", MaxEfSearchAllowed)
	}

	// Validate that at least one of dense or sparse is provided
	hasDense := len(vector) > 0
	hasSparseIndices := len(sparseIndices) > 0
	hasSparseValues := len(sparseValues) > 0

	if !hasDense && !hasSparseIndices {
		return nil, fmt.Errorf("at least one of vector (dense) or sparse_indices/sparse_values must be provided")
	}

	// Validate sparse data consistency
	if hasSparseIndices != hasSparseValues {
		return nil, fmt.Errorf("sparse_indices and sparse_values must both be provided together")
	}

	if hasSparseIndices && len(sparseIndices) != len(sparseValues) {
		return nil, fmt.Errorf("sparse_indices and sparse_values must have the same length")
	}

	// Normalize query vector
	normalizedVector, norm, err := i.normalizeVector(vector)
	if err != nil {
		return nil, err
	}
	originalVector := normalizedVector

	// Prepare search request
	requestData := QueryRequest{
		Vector:         normalizedVector,
		SparseIndices:  sparseIndices,
		SparseValues:   sparseValues,
		TopK:           k,
		Ef:             ef,
		IncludeVectors: includeVectors,
	}

	// Add filter if provided
	if filter != nil {
		filterBytes, err := json.Marshal([]map[string]interface{}{filter})
		if err != nil {
			return nil, fmt.Errorf("failed to serialize filter: %v", err)
		}
		requestData.Filter = string(filterBytes)
	}

	// Serialize request data
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Execute request using helper method with context
	resp, err := i.executeRequestWithContext(ctx, "POST", "index/%s/search", jsonData, "application/json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status and read body
	if err := checkError(resp); err != nil {
		return nil, err
	}

	// Read response body
	buf := getBuffer()
	defer putBuffer(buf)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse msgpack response
	var results [][]interface{}
	err = msgpack.Unmarshal(buf.Bytes(), &results)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Process results with optimized memory allocation
	// [similarity, id, meta, filter, norm, vector]
	processedResults := make([]QueryResult, 0, len(results))

	// For large result sets, use concurrent processing
	if len(results) > 50 {
		return i.processResultsConcurrent(ctx, results, includeVectors)
	}

	// Sequential processing for smaller result sets
	for _, result := range results {
		if len(result) < 5 {
			continue // Skip malformed results
		}

		similarity := toFloat32(result[0])
		vectorID := safeStringConvert(result[1])
		// metaData might be string or []byte/[]uint8
		var metaDataBytes []byte
		if result[2] != nil {
			switch v := result[2].(type) {
			case string:
				metaDataBytes = []byte(v)
			case []byte:
				metaDataBytes = v
			}
		}
		filterStr := safeStringConvert(result[3])
		normValue := toFloat32(result[4])

		var vectorData []float32
		if len(result) > 5 && result[5] != nil {
			vectorInterface := result[5].([]interface{})
			// Direct allocation instead of pooling
			vectorData = make([]float32, len(vectorInterface))

			// Convert with type safety using helper
			for j, v := range vectorInterface {
				vectorData[j] = toFloat32(v)
			}
		}

		processed := QueryResult{
			ID:         vectorID,
			Similarity: similarity,
			Distance:   1.0 - similarity,
			Norm:       normValue,
		}

		// Parse metadata (placeholder for json_unzip equivalent)
		// Parse metadata (unzip)
		if len(metaDataBytes) > 0 {
			if meta, err := JsonUnzip(metaDataBytes); err == nil {
				processed.Meta = meta
			}
		}

		// Parse filter
		if filterStr != "" {
			var filterMap map[string]interface{}
			if err := json.Unmarshal([]byte(filterStr), &filterMap); err == nil {
				processed.Filter = filterMap
			}
		}

		// Handle vectors
		if includeVectors && len(vectorData) > 0 {
			processed.Vector = vectorData
		}

		processedResults = append(processedResults, processed)
	}

	// Remove vectors if not requested
	if !includeVectors {
		for i := range processedResults {
			processedResults[i].Vector = nil
		}
	}

	// Use variables to avoid unused variable errors
	_ = norm
	_ = originalVector

	return processedResults, nil
}

// processResultsConcurrent processes query results concurrently for large result sets
func (i *Index) processResultsConcurrent(ctx context.Context, results [][]interface{}, includeVectors bool) ([]QueryResult, error) {
	numWorkers := runtime.NumCPU()
	if len(results) < numWorkers*2 {
		numWorkers = (len(results) + 1) / 2
	}

	// Channel for work distribution
	type workItem struct {
		index  int
		result []interface{}
	}

	workChan := make(chan workItem, numWorkers)
	resultChan := make(chan struct {
		index int
		data  QueryResult
		err   error
	}, len(results))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				processed, err := i.processResult(work.result, includeVectors)
				resultChan <- struct {
					index int
					data  QueryResult
					err   error
				}{work.index, processed, err}
			}
		}()
	}

	// Distribute work
	go func() {
		defer close(workChan)
		for i, result := range results {
			select {
			case workChan <- workItem{i, result}:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results in order
	processedResults := make([]QueryResult, len(results))
	for r := range resultChan {
		if r.err != nil {
			return nil, r.err
		}
		processedResults[r.index] = r.data
	}

	return processedResults, nil
}

// processResult processes a single query result
func (i *Index) processResult(result []interface{}, includeVectors bool) (QueryResult, error) {
	if len(result) < 5 {
		return QueryResult{}, fmt.Errorf("invalid result format: expected at least 5 elements, got %d", len(result))
	}

	similarity := toFloat32(result[0])
	vectorID := safeStringConvert(result[1])
	// metaData parsing
	var metaDataBytes []byte
	if result[2] != nil {
		switch v := result[2].(type) {
		case string:
			metaDataBytes = []byte(v)
		case []byte:
			metaDataBytes = v
		}
	}
	filterStr := safeStringConvert(result[3])
	normValue := toFloat32(result[4])

	processed := QueryResult{
		ID:         vectorID,
		Similarity: similarity,
		Distance:   1.0 - similarity,
		Norm:       normValue,
	}

	// Parse metadata (unzip)
	if len(metaDataBytes) > 0 {
		if meta, err := JsonUnzip(metaDataBytes); err == nil {
			processed.Meta = meta
		}
	}

	// Parse filter with pooled map
	if filterStr != "" {
		filter := getMap()
		if err := fastJSONUnmarshal([]byte(filterStr), &filter); err == nil {
			processed.Filter = filter
		} else {
			putMap(filter) // Return map to pool if parsing failed
		}
	}

	// Handle vectors
	if includeVectors && len(result) > 5 && result[5] != nil {
		vectorInterface := result[5].([]interface{})
		// Direct allocation instead of pooling
		vectorData := make([]float32, len(vectorInterface))

		// Convert with type safety
		for j, v := range vectorInterface {
			vectorData[j] = toFloat32(v)
		}

		if includeVectors {
			processed.Vector = vectorData
		}
	}

	return processed, nil
}

// DeleteVectorById deletes a vector by ID from the index
func (i *Index) DeleteVectorById(id string) (string, error) {
	return i.DeleteVectorByIdWithContext(context.Background(), id)
}

// DeleteVectorByIdWithContext deletes a vector by ID with context support
func (i *Index) DeleteVectorByIdWithContext(ctx context.Context, id string) (string, error) {
	// Execute request using helper method with context
	resp, err := i.executeRequestWithContext(ctx, "DELETE", fmt.Sprintf("index/%s/vector/%s/delete", i.Name, id), nil, "")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	buf := getBuffer()
	defer putBuffer(buf)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if err := checkError(resp); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DeleteVectorByFilter deletes vectors matching a specific filter from the index
func (i *Index) DeleteVectorByFilter(filter map[string]interface{}) (string, error) {
	return i.DeleteVectorByFilterWithContext(context.Background(), filter)
}

// DeleteVectorByFilterWithContext deletes vectors matching a filter with context support
func (i *Index) DeleteVectorByFilterWithContext(ctx context.Context, filter map[string]interface{}) (string, error) {
	if filter == nil {
		return "", fmt.Errorf("filter cannot be nil")
	}

	// Prepare request body
	// The API expects the filter as a raw JSON array of objects
	requestData := map[string]interface{}{
		"filter": []map[string]interface{}{filter},
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Execute request using helper method with context
	resp, err := i.executeRequestWithContext(ctx, "DELETE", fmt.Sprintf("index/%s/vectors/delete", i.Name), jsonData, "application/json")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	buf := getBuffer()
	defer putBuffer(buf)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if err := checkError(resp); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DeleteHybridVectorById deletes a hybrid vector by ID from the index
func (i *Index) DeleteHybridVectorById(id string) (string, error) {
	return i.DeleteVectorById(id)
}

// DeleteHybridVectorByIdWithContext deletes a hybrid vector by ID with context support
func (i *Index) DeleteHybridVectorByIdWithContext(ctx context.Context, id string) (string, error) {
	return i.DeleteVectorByIdWithContext(ctx, id)
}

// DeleteHybridVectorByFilter deletes hybrid vectors matching a specific filter from the index
func (i *Index) DeleteHybridVectorByFilter(filter map[string]interface{}) (string, error) {
	return i.DeleteVectorByFilter(filter)
}

// DeleteHybridVectorByFilterWithContext deletes hybrid vectors matching a filter with context support
func (i *Index) DeleteHybridVectorByFilterWithContext(ctx context.Context, filter map[string]interface{}) (string, error) {
	return i.DeleteVectorByFilterWithContext(ctx, filter)
}

func (i *Index) GetVector(id string) (VectorItem, error) {
	return i.GetVectorWithContext(context.Background(), id)
}

// GetVectorWithContext retrieves a vector by ID with context support
func (i *Index) GetVectorWithContext(ctx context.Context, id string) (VectorItem, error) {
	// Prepare request body with the vector ID using fast JSON
	requestData := map[string]string{"id": id}
	jsonData, err := fastJSONMarshal(requestData)
	if err != nil {
		return VectorItem{}, fmt.Errorf("failed to marshal request data: %w", err)
	}

	// Execute request using helper method with context
	resp, err := i.executeRequestWithContext(ctx, "POST", "index/%s/vector/get", jsonData, "application/json")
	if err != nil {
		return VectorItem{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status and read body
	if err := checkError(resp); err != nil {
		return VectorItem{}, err
	}

	// Read response body
	buf := getBuffer()
	defer putBuffer(buf)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return VectorItem{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse msgpack response
	var vectorObj []interface{}
	err = msgpack.Unmarshal(buf.Bytes(), &vectorObj)
	if err != nil {
		return VectorItem{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Ensure we have the expected array structure: [id, meta, filter, norm, vector]
	if len(vectorObj) < 5 {
		return VectorItem{}, fmt.Errorf("invalid response format: expected 5 elements, got %d", len(vectorObj))
	}

	// Extract data from response array
	vectorID := safeStringConvert(vectorObj[0])

	var metaDataBytes []byte
	if vectorObj[1] != nil {
		switch v := vectorObj[1].(type) {
		case string:
			metaDataBytes = []byte(v)
		case []byte:
			metaDataBytes = v
		}
	}

	filterData := safeStringConvert(vectorObj[2])
	normValue := toFloat32(vectorObj[3])
	vectorInterface := vectorObj[4].([]interface{})

	// Handle sparse data if present (elements 5 and 6)
	var sparseIndices []int
	var sparseValues []float32

	if len(vectorObj) >= 7 {
		// Extract sparse indices
		if indicesInterface, ok := vectorObj[5].([]interface{}); ok {
			sparseIndices = make([]int, len(indicesInterface))
			for j, v := range indicesInterface {
				if idx, ok := v.(int64); ok {
					sparseIndices[j] = int(idx)
				} else if idx, ok := v.(uint64); ok {
					sparseIndices[j] = int(idx)
				} // Add other number types if needed
			}
		}

		// Extract sparse values
		if valuesInterface, ok := vectorObj[6].([]interface{}); ok {
			sparseValues = make([]float32, len(valuesInterface))
			for j, v := range valuesInterface {
				sparseValues[j] = toFloat32(v)
			}
		}
	}

	// Convert vector data with type safety
	vector := make([]float32, len(vectorInterface))
	for j, v := range vectorInterface {
		vector[j] = toFloat32(v)
	}

	// Parse metadata using JsonUnzip
	var meta map[string]interface{}
	if len(metaDataBytes) > 0 {
		if m, err := JsonUnzip(metaDataBytes); err == nil {
			meta = m
		} else {
			meta = make(map[string]interface{})
		}
	} else {
		meta = make(map[string]interface{})
	}

	// Parse filter using pooled map and fast JSON
	var filter map[string]interface{}
	if filterData != "" {
		filter = getMap()
		if err := fastJSONUnmarshal([]byte(filterData), &filter); err != nil {
			// If parsing fails, return map to pool and create new empty map
			putMap(filter)
			filter = make(map[string]interface{})
		}
	} else {
		filter = make(map[string]interface{})
	}

	// Use the norm value to avoid unused variable warnings
	_ = normValue

	// Return the VectorItem
	return VectorItem{
		ID:            vectorID,
		Vector:        vector,
		SparseIndices: sparseIndices,
		SparseValues:  sparseValues,
		Meta:          meta,
		Filter:        filter,
	}, nil
}

// safeStringConvert safely converts interface{} to string, handling both string and []uint8 cases
func safeStringConvert(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []uint8:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toFloat32 safely converts an interface{} to float32
func toFloat32(val interface{}) float32 {
	if val == nil {
		return 0.0
	}
	switch v := val.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int:
		return float32(v)
	case int8:
		return float32(v)
	case int16:
		return float32(v)
	case int32:
		return float32(v)
	case int64:
		return float32(v)
	case uint8:
		return float32(v)
	case uint16:
		return float32(v)
	case uint32:
		return float32(v)
	case uint64:
		return float32(v)
	default:
		return 0.0
	}
}
