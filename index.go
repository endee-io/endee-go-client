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

	"github.com/endee-io/endee-go-client/internal/jsonzip"
	"github.com/vmihailenco/msgpack/v5"
)

// Index represents an Endee vector index and its configuration.
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

// IndexParams holds the configuration parameters used when constructing an Index.
type IndexParams struct {
	LibToken      string `json:"lib_token"`
	TotalElements int    `json:"total_elements"`
	SpaceType     string `json:"space_type"`
	Dimension     int    `json:"dimension"`
	SparseDim     int    `json:"sparse_dim"`
	Precision     string `json:"precision"`
	M             int    `json:"M"`
	EfCon         int    `json:"ef_con"`
}

// queryRequest is the request body for a vector similarity search.
type queryRequest struct {
	Vector         []float32     `json:"vector,omitempty"`
	SparseIndices  []int         `json:"sparse_indices,omitempty"`
	SparseValues   []float32     `json:"sparse_values,omitempty"`
	TopK           int           `json:"k"`
	Ef             int           `json:"ef"`
	IncludeVectors bool          `json:"include_vectors"`
	Filter         string        `json:"filter,omitempty"`
	FilterParams   *FilterParams `json:"filter_params,omitempty"`
}

// filterUpdateRequest is the request body for updating vector filter metadata.
type filterUpdateRequest struct {
	Updates []FilterUpdateItem `json:"updates"`
}

// NewIndex creates a new Index instance.
func NewIndex(name string, token string, url string, version int, params *IndexParams) *Index {
	if version == 0 {
		version = 1
	}

	precision := PrecisionInt16
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

// buildURL efficiently builds API URLs for index operations.
func (idx *Index) buildURL(path string) string {
	var builder strings.Builder

	builder.Grow(len(idx.URL) + len(path) + len(idx.Name) + 10)
	builder.WriteString(idx.URL)

	if !strings.HasSuffix(idx.URL, "/") {
		builder.WriteString("/")
	}

	if strings.Contains(path, "%s") {
		return fmt.Sprintf(builder.String()+path, idx.Name)
	}

	builder.WriteString(path)

	return builder.String()
}

// executeRequest executes HTTP requests with consistent headers and error handling.
func (idx *Index) executeRequest(method, path string, body []byte, contentType string) (*http.Response, error) {
	return idx.executeRequestWithContext(context.Background(), method, path, body, contentType)
}

// executeRequestWithContext executes HTTP requests with context support.
func (idx *Index) executeRequestWithContext(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, idx.buildURL(path), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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

// normalizeVector normalizes a vector for cosine similarity searches.
func (idx *Index) normalizeVector(vector []float32) ([]float32, float32, error) {
	if len(vector) != idx.Dimension {
		return nil, 0, fmt.Errorf("vector dimension mismatch: expected %d, got %d",
			idx.Dimension, len(vector))
	}

	if idx.SpaceType != "cosine" {
		return vector, 1.0, nil
	}

	var sum float32

	for _, v := range vector {
		sum += v * v
	}

	norm := float32(math.Sqrt(float64(sum)))

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

// Upsert inserts or updates vectors in the index.
func (idx *Index) Upsert(inputArray []VectorItem) error {
	return idx.UpsertWithContext(context.Background(), inputArray)
}

// UpsertWithContext inserts or updates vectors with context support and concurrent processing.
func (idx *Index) UpsertWithContext(ctx context.Context, inputArray []VectorItem) error {
	if len(inputArray) > MaxVectorsPerBatch {
		return fmt.Errorf("cannot insert more than %d vectors at a time", MaxVectorsPerBatch)
	}

	if len(inputArray) == 0 {
		return nil
	}

	for i, item := range inputArray {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("id must not be empty (item index %d)", i)
		}

		hasIndices := len(item.SparseIndices) > 0
		hasValues := len(item.SparseValues) > 0

		if hasIndices != hasValues {
			return fmt.Errorf("sparse_indices and sparse_values must both be provided together (item id: %s)", item.ID)
		}

		if hasIndices && len(item.SparseIndices) != len(item.SparseValues) {
			return fmt.Errorf("sparse_indices and sparse_values must have the same length (item id: %s)", item.ID)
		}
	}

	if len(inputArray) <= 10 {
		return idx.upsertSequential(ctx, inputArray)
	}

	return idx.upsertConcurrent(ctx, inputArray)
}

// upsertSequential processes vectors sequentially for small batches.
func (idx *Index) upsertSequential(ctx context.Context, inputArray []VectorItem) error {
	vectorBatch := make([][]interface{}, 0, len(inputArray))

	for _, item := range inputArray {
		normalizedVector, norm, err := idx.normalizeVector(item.Vector)
		if err != nil {
			return err
		}

		metaBytes, err := jsonzip.Zip(item.Meta)
		if err != nil {
			return fmt.Errorf("failed to compress metadata: %w", err)
		}

		filterBytes, err := json.Marshal(item.Filter)
		if err != nil {
			return fmt.Errorf("failed to serialize filter: %w", err)
		}

		vectorObj := []interface{}{
			item.ID,
			metaBytes,
			string(filterBytes),
			norm,
			normalizedVector,
		}

		if len(item.SparseIndices) > 0 && len(item.SparseValues) > 0 {
			vectorObj = append(vectorObj, item.SparseIndices, item.SparseValues)
		}

		vectorBatch = append(vectorBatch, vectorObj)
	}

	serializedData, err := msgpack.Marshal(vectorBatch)
	if err != nil {
		return fmt.Errorf("failed to serialize vector batch: %w", err)
	}

	resp, err := idx.executeRequestWithContext(ctx, "POST", "index/%s/vector/insert", serializedData, "application/msgpack")
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	return checkError(resp)
}

// upsertConcurrent processes vectors concurrently for large batches.
func (idx *Index) upsertConcurrent(ctx context.Context, inputArray []VectorItem) error {
	numWorkers := runtime.NumCPU()
	if len(inputArray) < numWorkers*2 {
		numWorkers = (len(inputArray) + 1) / 2
	}

	batchSize := (len(inputArray) + numWorkers - 1) / numWorkers
	if batchSize > 100 {
		batchSize = 100
	}

	workChan := make(chan []VectorItem, numWorkers)
	resultChan := make(chan error, numWorkers)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for batch := range workChan {
				if len(batch) > 0 {
					resultChan <- idx.upsertSequential(ctx, batch)
				}
			}
		}()
	}

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

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var errs []error

	for err := range resultChan {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("upsert failed: %w", errs[0])
	}

	return nil
}

// String implements the fmt.Stringer interface.
func (idx *Index) String() string {
	return idx.Name
}

// GetInfo returns a formatted string with index information for debugging.
func (idx *Index) GetInfo() string {
	return fmt.Sprintf("Index{Name: %s, Dimension: %d, SparseDim: %d, SpaceType: %s, Count: %d, Precision: %s, M: %d}",
		idx.Name, idx.Dimension, idx.SparseDim, idx.SpaceType, idx.Count, idx.Precision, idx.M)
}

// Query performs a vector similarity search.
func (idx *Index) Query(vector []float32, sparseIndices []int, sparseValues []float32, k int, filter map[string]interface{}, ef int, includeVectors bool, filterParams *FilterParams) ([]QueryResult, error) {
	return idx.QueryWithContext(context.Background(), vector, sparseIndices, sparseValues, k, filter, ef, includeVectors, filterParams)
}

// QueryWithContext performs vector similarity search with context support.
func (idx *Index) QueryWithContext(ctx context.Context, vector []float32, sparseIndices []int, sparseValues []float32, k int, filter map[string]interface{}, ef int, includeVectors bool, filterParams *FilterParams) ([]QueryResult, error) {
	if k <= 0 || k > MaxTopKAllowed {
		return nil, fmt.Errorf("top_k must be between 1 and %d", MaxTopKAllowed)
	}

	if ef < 0 || ef > MaxEfSearchAllowed {
		return nil, fmt.Errorf("ef must be between 0 and %d", MaxEfSearchAllowed)
	}

	hasDense := len(vector) > 0
	hasSparseIndices := len(sparseIndices) > 0
	hasSparseValues := len(sparseValues) > 0

	if !hasDense && !hasSparseIndices {
		return nil, fmt.Errorf("at least one of vector (dense) or sparse_indices/sparse_values must be provided")
	}

	if hasSparseIndices != hasSparseValues {
		return nil, fmt.Errorf("sparse_indices and sparse_values must both be provided together")
	}

	if hasSparseIndices && len(sparseIndices) != len(sparseValues) {
		return nil, fmt.Errorf("sparse_indices and sparse_values must have the same length")
	}

	if filterParams != nil {
		if filterParams.BoostPercentage < 0 || filterParams.BoostPercentage > 100 {
			return nil, fmt.Errorf("filter_boost_percentage must be between 0 and 100")
		}

		if filterParams.PrefilterThreshold != 0 &&
			(filterParams.PrefilterThreshold < 1000 || filterParams.PrefilterThreshold > 1000000) {
			return nil, fmt.Errorf("prefilter_cardinality_threshold must be between 1,000 and 1,000,000")
		}
	}

	normalizedVector, norm, err := idx.normalizeVector(vector)
	if err != nil {
		return nil, err
	}

	originalVector := normalizedVector

	requestData := queryRequest{
		Vector:         normalizedVector,
		SparseIndices:  sparseIndices,
		SparseValues:   sparseValues,
		TopK:           k,
		Ef:             ef,
		IncludeVectors: includeVectors,
		FilterParams:   filterParams,
	}

	if filter != nil {
		filterBytes, err := json.Marshal([]map[string]interface{}{filter})
		if err != nil {
			return nil, fmt.Errorf("failed to serialize filter: %w", err)
		}

		requestData.Filter = string(filterBytes)
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	resp, err := idx.executeRequestWithContext(ctx, "POST", "index/%s/search", jsonData, "application/json")
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

	var results [][]interface{}

	if err := msgpack.Unmarshal(buf.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(results) > 50 {
		return idx.processResultsConcurrent(ctx, results, includeVectors)
	}

	processedResults := make([]QueryResult, 0, len(results))

	for _, result := range results {
		if len(result) < 5 {
			continue
		}

		similarity := toFloat32(result[0])
		vectorID := safeStringConvert(result[1])

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
			vectorData = make([]float32, len(vectorInterface))

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

		if len(metaDataBytes) > 0 {
			if meta, err := jsonzip.Unzip(metaDataBytes); err == nil {
				processed.Meta = meta
			}
		}

		if filterStr != "" {
			var filterMap map[string]interface{}

			if err := json.Unmarshal([]byte(filterStr), &filterMap); err == nil {
				processed.Filter = filterMap
			}
		}

		if includeVectors && len(vectorData) > 0 {
			processed.Vector = vectorData
		}

		processedResults = append(processedResults, processed)
	}

	if !includeVectors {
		for i := range processedResults {
			processedResults[i].Vector = nil
		}
	}

	_ = norm
	_ = originalVector

	return processedResults, nil
}

// processResultsConcurrent processes query results concurrently for large result sets.
func (idx *Index) processResultsConcurrent(ctx context.Context, results [][]interface{}, includeVectors bool) ([]QueryResult, error) {
	numWorkers := runtime.NumCPU()
	if len(results) < numWorkers*2 {
		numWorkers = (len(results) + 1) / 2
	}

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

	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for work := range workChan {
				processed, err := idx.processResult(work.result, includeVectors)
				resultChan <- struct {
					index int
					data  QueryResult
					err   error
				}{work.index, processed, err}
			}
		}()
	}

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

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	processedResults := make([]QueryResult, len(results))

	for r := range resultChan {
		if r.err != nil {
			return nil, r.err
		}

		processedResults[r.index] = r.data
	}

	return processedResults, nil
}

// processResult processes a single query result entry.
func (idx *Index) processResult(result []interface{}, includeVectors bool) (QueryResult, error) {
	if len(result) < 5 {
		return QueryResult{}, fmt.Errorf("invalid result format: expected at least 5 elements, got %d", len(result))
	}

	similarity := toFloat32(result[0])
	vectorID := safeStringConvert(result[1])

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

	if len(metaDataBytes) > 0 {
		if meta, err := jsonzip.Unzip(metaDataBytes); err == nil {
			processed.Meta = meta
		}
	}

	if filterStr != "" {
		filter := getMap()

		if err := fastJSONUnmarshal([]byte(filterStr), &filter); err == nil {
			processed.Filter = filter
		} else {
			putMap(filter)
		}
	}

	if includeVectors && len(result) > 5 && result[5] != nil {
		vectorInterface := result[5].([]interface{})
		vectorData := make([]float32, len(vectorInterface))

		for j, v := range vectorInterface {
			vectorData[j] = toFloat32(v)
		}

		processed.Vector = vectorData
	}

	return processed, nil
}

// DeleteVectorByID deletes a vector by ID from the index.
func (idx *Index) DeleteVectorByID(id string) (string, error) {
	return idx.DeleteVectorByIDWithContext(context.Background(), id)
}

// DeleteVectorById deletes a vector by ID from the index.
//
// Deprecated: Use DeleteVectorByID instead.
func (idx *Index) DeleteVectorById(id string) (string, error) { //nolint:revive
	return idx.DeleteVectorByIDWithContext(context.Background(), id)
}

// DeleteVectorByIDWithContext deletes a vector by ID with context support.
func (idx *Index) DeleteVectorByIDWithContext(ctx context.Context, id string) (string, error) {
	resp, err := idx.executeRequestWithContext(ctx, "DELETE", fmt.Sprintf("index/%s/vector/%s/delete", idx.Name, id), nil, "")
	if err != nil {
		return "", err
	}

	defer func() { _ = resp.Body.Close() }()

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if err := checkError(resp); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DeleteVectorByFilter deletes vectors matching a filter from the index.
func (idx *Index) DeleteVectorByFilter(filter map[string]interface{}) (string, error) {
	return idx.DeleteVectorByFilterWithContext(context.Background(), filter)
}

// DeleteVectorByFilterWithContext deletes vectors matching a filter with context support.
func (idx *Index) DeleteVectorByFilterWithContext(ctx context.Context, filter map[string]interface{}) (string, error) {
	if filter == nil {
		return "", fmt.Errorf("filter cannot be nil")
	}

	requestData := map[string]interface{}{
		"filter": []map[string]interface{}{filter},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}

	resp, err := idx.executeRequestWithContext(ctx, "DELETE", fmt.Sprintf("index/%s/vectors/delete", idx.Name), jsonData, "application/json")
	if err != nil {
		return "", err
	}

	defer func() { _ = resp.Body.Close() }()

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if err := checkError(resp); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DeleteHybridVectorByID deletes a hybrid vector by ID from the index.
func (idx *Index) DeleteHybridVectorByID(id string) (string, error) {
	return idx.DeleteVectorByID(id)
}

// DeleteHybridVectorByIDWithContext deletes a hybrid vector by ID with context support.
func (idx *Index) DeleteHybridVectorByIDWithContext(ctx context.Context, id string) (string, error) {
	return idx.DeleteVectorByIDWithContext(ctx, id)
}

// DeleteHybridVectorById deletes a hybrid vector by ID from the index.
//
// Deprecated: Use DeleteHybridVectorByID instead.
func (idx *Index) DeleteHybridVectorById(id string) (string, error) { //nolint:revive
	return idx.DeleteVectorByID(id)
}

// DeleteHybridVectorByIdWithContext deletes a hybrid vector by ID with context support.
//
// Deprecated: Use DeleteHybridVectorByIDWithContext instead.
func (idx *Index) DeleteHybridVectorByIdWithContext(ctx context.Context, id string) (string, error) { //nolint:revive
	return idx.DeleteVectorByIDWithContext(ctx, id)
}

// DeleteHybridVectorByFilter deletes hybrid vectors matching a filter from the index.
func (idx *Index) DeleteHybridVectorByFilter(filter map[string]interface{}) (string, error) {
	return idx.DeleteVectorByFilter(filter)
}

// DeleteHybridVectorByFilterWithContext deletes hybrid vectors matching a filter with context support.
func (idx *Index) DeleteHybridVectorByFilterWithContext(ctx context.Context, filter map[string]interface{}) (string, error) {
	return idx.DeleteVectorByFilterWithContext(ctx, filter)
}

// GetVector retrieves a vector by ID.
func (idx *Index) GetVector(id string) (VectorItem, error) {
	return idx.GetVectorWithContext(context.Background(), id)
}

// GetVectorWithContext retrieves a vector by ID with context support.
func (idx *Index) GetVectorWithContext(ctx context.Context, id string) (VectorItem, error) {
	requestData := map[string]string{"id": id}

	jsonData, err := fastJSONMarshal(requestData)
	if err != nil {
		return VectorItem{}, fmt.Errorf("failed to marshal request data: %w", err)
	}

	resp, err := idx.executeRequestWithContext(ctx, "POST", "index/%s/vector/get", jsonData, "application/json")
	if err != nil {
		return VectorItem{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkError(resp); err != nil {
		return VectorItem{}, err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return VectorItem{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var vectorObj []interface{}

	if err := msgpack.Unmarshal(buf.Bytes(), &vectorObj); err != nil {
		return VectorItem{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(vectorObj) < 5 {
		return VectorItem{}, fmt.Errorf("invalid response format: expected 5 elements, got %d", len(vectorObj))
	}

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

	var sparseIndices []int

	var sparseValues []float32

	if len(vectorObj) >= 7 {
		if indicesInterface, ok := vectorObj[5].([]interface{}); ok {
			sparseIndices = make([]int, len(indicesInterface))

			for j, v := range indicesInterface {
				if val, ok := v.(int64); ok {
					sparseIndices[j] = int(val)
				} else if val, ok := v.(uint64); ok {
					sparseIndices[j] = int(val)
				}
			}
		}

		if valuesInterface, ok := vectorObj[6].([]interface{}); ok {
			sparseValues = make([]float32, len(valuesInterface))

			for j, v := range valuesInterface {
				sparseValues[j] = toFloat32(v)
			}
		}
	}

	vector := make([]float32, len(vectorInterface))

	for j, v := range vectorInterface {
		vector[j] = toFloat32(v)
	}

	var meta map[string]interface{}

	if len(metaDataBytes) > 0 {
		if m, err := jsonzip.Unzip(metaDataBytes); err == nil {
			meta = m
		} else {
			meta = make(map[string]interface{})
		}
	} else {
		meta = make(map[string]interface{})
	}

	var filter map[string]interface{}

	if filterData != "" {
		filter = getMap()

		if err := fastJSONUnmarshal([]byte(filterData), &filter); err != nil {
			putMap(filter)
			filter = make(map[string]interface{})
		}
	} else {
		filter = make(map[string]interface{})
	}

	_ = normValue

	return VectorItem{
		ID:            vectorID,
		Vector:        vector,
		SparseIndices: sparseIndices,
		SparseValues:  sparseValues,
		Meta:          meta,
		Filter:        filter,
	}, nil
}

// UpdateFilters updates filter metadata for multiple vectors by ID.
func (idx *Index) UpdateFilters(updates []FilterUpdateItem) (string, error) {
	return idx.UpdateFiltersWithContext(context.Background(), updates)
}

// UpdateFiltersWithContext updates filter metadata for multiple vectors with context support.
func (idx *Index) UpdateFiltersWithContext(ctx context.Context, updates []FilterUpdateItem) (string, error) {
	if len(updates) == 0 {
		return "", fmt.Errorf("updates cannot be empty")
	}

	for i, update := range updates {
		if strings.TrimSpace(update.ID) == "" {
			return "", fmt.Errorf("id must not be empty (update index %d)", i)
		}

		if update.Filter == nil {
			return "", fmt.Errorf("filter cannot be nil (update index %d, id: %s)", i, update.ID)
		}
	}

	requestData := filterUpdateRequest{Updates: updates}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}

	resp, err := idx.executeRequestWithContext(ctx, "POST", fmt.Sprintf("index/%s/filters/update", idx.Name), jsonData, "application/json")
	if err != nil {
		return "", err
	}

	defer func() { _ = resp.Body.Close() }()

	if err := checkError(resp); err != nil {
		return "", err
	}

	buf := getBuffer()
	defer putBuffer(buf)

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return buf.String(), nil
}
