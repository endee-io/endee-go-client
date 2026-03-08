package endee

// IndexInfo holds metadata about a vector index as returned by the list endpoint.
type IndexInfo struct {
	Name          string `json:"name"`
	Dimension     int    `json:"dimension"`
	SpaceType     string `json:"space_type"`
	TotalElements int    `json:"total_elements"`
	CreatedAt     int64  `json:"created_at"`
	Precision     string `json:"precision,omitempty"`
	M             int    `json:"M,omitempty"`
	EfCon         int    `json:"ef_con,omitempty"`
	SparseDim     int    `json:"sparse_dim,omitempty"`
}

// VectorItem represents a vector with metadata for upserting into an index.
type VectorItem struct {
	ID            string                 `json:"id"`
	Vector        []float32              `json:"vector"`
	SparseIndices []int                  `json:"sparse_indices,omitempty"`
	SparseValues  []float32              `json:"sparse_values,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	Filter        map[string]interface{} `json:"filter,omitempty"`
}

// QueryResult represents a single vector search result.
type QueryResult struct {
	ID         string                 `json:"id"`
	Similarity float32                `json:"similarity"`
	Distance   float32                `json:"distance"`
	Meta       map[string]interface{} `json:"meta"`
	Filter     map[string]interface{} `json:"filter,omitempty"`
	Norm       float32                `json:"norm"`
	Vector     []float32              `json:"vector,omitempty"`
}

// FilterParams configures advanced filtering behavior for HNSW search.
type FilterParams struct {
	BoostPercentage    int `json:"boost_percentage,omitempty"`
	PrefilterThreshold int `json:"prefilter_threshold,omitempty"`
}

// FilterUpdateItem represents a single filter metadata update for a vector.
type FilterUpdateItem struct {
	ID     string                 `json:"id"`
	Filter map[string]interface{} `json:"filter"`
}
