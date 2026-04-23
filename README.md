# Endee - High-Performance Vector Database (Go Client)

Endee is a high-performance vector database designed for speed and efficiency. The Go client enables rapid Approximate Nearest Neighbor (ANN) searches for applications requiring robust vector search capabilities with advanced filtering, metadata support, and hybrid search combining dense and sparse vectors.

## Key Features

- **Fast ANN Searches**: Efficient similarity searches on vector data using HNSW algorithm
- **Hybrid Search**: Combine dense and sparse vectors for powerful semantic + keyword search using Reciprocal Rank Fusion (RRF)
- **Multiple Distance Metrics**: Support for cosine, L2, and inner product distance metrics
- **Metadata Support**: Attach and search with metadata and filters
- **Advanced Filtering**: Powerful query filtering with operators like `$eq`, `$in`, and `$range`
- **High Performance**: Optimized for speed and efficiency with connection pooling and concurrent processing
- **Scalable**: Handle millions of vectors with ease
- **Configurable Precision**: Multiple precision levels for memory/accuracy tradeoffs
- **Context Support**: Full context.Context support for cancellation and timeouts

## Installation

```bash
go get github.com/endee-io/endee-go-client
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/endee-io/endee-go-client"
)

func main() {
    // Initialize client with your API token
    client := endee.EndeeClient("your-token-here")
    // For no auth development use:
    // client := endee.EndeeClient("")
    
    // List existing indexes
    indexes, err := client.ListIndexes()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Found %d indexes\n", len(indexes))
    
    // Create a new dense index
    err = client.CreateIndex(
        "my_vectors",            // name
        768,                     // dimension
        "cosine",                // space_type (cosine, l2, ip)
        16,                      // M - HNSW connectivity parameter
        128,                     // ef_con - construction parameter
        endee.PrecisionFloat32,  // precision (float32, float16, int16, int8, binary)
        nil,                     // version (optional)
        "",                      // sparseModel ("" for dense-only, "default" or "endee_bm25" for hybrid)
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Get index reference
    index, err := client.GetIndex("my_vectors")
    if err != nil {
        log.Fatal(err)
    }
    
    // Insert vectors
    vectors := []endee.VectorItem{
        {
            ID:     "doc1",
            Vector: []float32{0.1, 0.2, 0.3, /* ... 768 dimensions */},
            Meta: map[string]interface{}{
                "text":     "Example document",
                "category": "reference",
            },
            Filter: map[string]interface{}{
                "category": "reference",
                "tags":     "important",
            },
        },
    }
    
    err = index.Upsert(vectors)
    if err != nil {
        log.Fatal(err)
    }
    
    // Query similar vectors with filtering
    queryVector := []float32{0.2, 0.3, 0.4, /* ... 768 dimensions */}
    filter := map[string]interface{}{
        "category": map[string]interface{}{
            "$eq": "reference",
        },
    }
    
    results, err := index.Query(queryVector, nil, nil, 10, filter, 128, false, nil, 0.5, 60)
    if err != nil {
        log.Fatal(err)
    }
    
    // Process results
    for _, item := range results {
        fmt.Printf("ID: %s, Similarity: %.3f\n", item.ID, item.Similarity)
        fmt.Printf("Metadata: %+v\n", item.Meta)
    }
}
```

## Basic Usage

To interact with the Endee platform, you'll need to authenticate using an API token. This token is used to securely identify your workspace and authorize all actions — including index creation, vector upserts, and queries.

Not using a token at any development stage will result in open APIs and vectors.

### Generate Your API Token

- Each token is tied to your workspace and should be kept private
- Once you have your token, you're ready to initialize the client and begin using the SDK

### Initializing the Client

The Endee client acts as the main interface for all vector operations — such as creating indexes, upserting vectors, and running similarity queries. You can initialize the client in just a few lines:

```go
import "github.com/endee-io/endee-go-client"

// Initialize with your API token
client := endee.EndeeClient("your-token-here")

// For local development without authentication
client := endee.EndeeClient("")
```

### Setting Up Your Domain

The Endee client allows for setting custom domain URL and port changes (default port 8080):

```go
client := endee.EndeeClient("your-token-here")

// Manually set base URL if needed
client.BaseURL = "http://0.0.0.0:8081/api/v1"
```

### Listing All Indexes

The `client.ListIndexes()` method returns a list of all the indexes currently available in your environment or workspace. This is useful for managing, debugging, or programmatically selecting indexes for vector operations like upsert or search.

```go
client := endee.EndeeClient("your-token-here")

// List all indexes in your workspace
indexes, err := client.ListIndexes()
if err != nil {
    log.Fatal(err)
}

for i, idx := range indexes {
    fmt.Printf("%d. %+v\n", i+1, idx)
}
```

### Create an Index

The `client.CreateIndex()` method initializes a new vector index with customizable parameters such as dimensionality, distance metric, graph construction settings, and precision level.

```go
client := endee.EndeeClient("your-token-here")

// Create a dense index
err := client.CreateIndex(
    "my_custom_index",       // name
    768,                     // dimension
    "cosine",                // space_type
    16,                      // M (graph connectivity, default = 16)
    128,                     // ef_con (construction parameter, default = 128)
    endee.PrecisionFloat32,  // precision
    nil,                     // version (optional)
    "",                      // sparseModel ("" for dense-only)
)
if err != nil {
    log.Fatal(err)
}

// Create a hybrid (dense + sparse) index
err = client.CreateIndex(
    "my_hybrid_index",
    768,
    "cosine",
    16,
    128,
    endee.PrecisionFloat32,
    nil,
    endee.SparseModelDefault, // or endee.SparseModelEndEeBM25
)
```

**Parameters:**

- `name`: Unique name for your index (alphanumeric + underscores, max 48 chars)
- `dimension`: Vector dimensionality (must match your embedding model's output, max 8000)
- `spaceType`: Distance metric - `"cosine"`, `"l2"`, or `"ip"` (inner product)
- `M`: HNSW graph connectivity parameter - higher values increase recall but use more memory (default: 16)
- `efCon`: HNSW construction parameter - higher values improve index quality but slow down indexing (default: 128)
- `precision`: Support for multiple precision levels - `PrecisionFloat32`, `PrecisionFloat16`, `PrecisionInt16`, `PrecisionInt8`, `PrecisionBinary`
- `version`: Optional version parameter for index versioning
- `sparseModel`: Sparse model for hybrid search (`""` for dense-only, `"default"` or `"endee_bm25"` for hybrid)

**Precision Levels:**

The Go client supports various precision levels for memory/accuracy tradeoffs:

| Precision | Constant | Data Type | Memory Usage | Use Case |
|-----------|----------|-----------|--------------|----------|
| FP32 | `PrecisionFloat32` | 32-bit float | Highest | Maximum accuracy |
| FP16 | `PrecisionFloat16` | 16-bit float | ~50% less | Good accuracy, lower memory |
| INT16 (default) | `PrecisionInt16` | 16-bit int | Optimized | Quantized accuracy |
| INT8 | `PrecisionInt8` | 8-bit int | ~75% less | Maximum memory savings |
| Binary | `PrecisionBinary` | 1-bit | Minimum | Fast, low-memory keyword-like search |

```go
// High accuracy index (FP32)
err := client.CreateIndex("high_accuracy_index", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, "")

// Memory-optimized index (INT8)
err = client.CreateIndex("low_memory_index", 768, "cosine", 16, 128, endee.PrecisionInt8, nil, "")
```

### Get an Index

The `client.GetIndex()` method retrieves a reference to an existing index. This is required before performing vector operations like upsert, query, or delete.

```go
client := endee.EndeeClient("your-token-here")

index, err := client.GetIndex("my_custom_index")
if err != nil {
    log.Fatal(err)
}

fmt.Println(index.GetInfo())
```

### Ingestion of Data

The `index.Upsert()` method adds or updates vectors in an existing index. Each `VectorItem` contains a unique identifier, vector data, optional metadata, and optional filter fields.

```go
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

vectors := []endee.VectorItem{
    {
        ID:     "vec1",
        Vector: []float32{/* your vector */},
        Meta: map[string]interface{}{
            "title": "First document",
        },
        Filter: map[string]interface{}{
            "tags": "important",
        },
    },
    {
        ID:     "vec2",
        Vector: []float32{/* another vector */},
        Meta: map[string]interface{}{
            "title": "Second document",
        },
        Filter: map[string]interface{}{
            "visibility": "public",
            "tags":       "important",
        },
    },
}

err = index.Upsert(vectors)
if err != nil {
    log.Fatal(err)
}
```

**VectorItem Fields:**

- `ID`: Unique identifier for the vector (required, must be non-empty)
- `Vector`: Slice of float32 representing the embedding (required for dense/hybrid indexes; no NaN or Inf values)
- `SparseIndices`: Sparse vector indices (required for hybrid indexes, must pair with `SparseValues`)
- `SparseValues`: Sparse vector values (required for hybrid indexes, must pair with `SparseIndices`)
- `Meta`: Map for storing additional information (optional)
- `Filter`: Map with key-value pairs for structured filtering during queries (optional)

> **Note:** Maximum batch size is 1000 vectors per upsert call. Duplicate IDs within a single batch are rejected. For hybrid indexes, **all** items in the batch must include sparse data; for dense-only indexes, sparse data is not allowed.

### Hybrid Search

Hybrid indexes combine dense and sparse vectors using Reciprocal Rank Fusion (RRF) to blend semantic similarity with keyword-level precision.

#### Creating a Hybrid Index

```go
err := client.CreateIndex(
    "my_hybrid_index",
    768,
    "cosine",
    16,
    128,
    endee.PrecisionFloat32,
    nil,
    endee.SparseModelDefault, // enable hybrid mode
)
```

#### Upserting Hybrid Vectors

Every item in a hybrid upsert must provide both dense and sparse components:

```go
vectors := []endee.VectorItem{
    {
        ID:            "doc1",
        Vector:        []float32{/* dense embedding */},
        SparseIndices: []int{5, 42, 100},
        SparseValues:  []float32{0.8, 0.3, 0.6},
        Meta:          map[string]interface{}{"title": "Example"},
        Filter:        map[string]interface{}{"category": "news"},
    },
}

err = index.Upsert(vectors)
```

#### Querying a Hybrid Index

Pass sparse data alongside the dense vector. Use `denseRRFWeight` and `rrfRankConstant` to tune RRF blending:

```go
results, err := index.Query(
    denseVector,   // dense query vector
    []int{5, 42},  // sparseIndices
    []float32{0.8, 0.3}, // sparseValues
    10,            // top_k
    nil,           // filter
    128,           // ef
    false,         // includeVectors
    nil,           // filterParams
    0.5,           // denseRRFWeight (0.0–1.0; 0.5 = equal weight)
    60,            // rrfRankConstant (≥1; default: 60)
)
```

### Querying the Index

The `index.Query()` method performs a similarity search using a query vector.

```go
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

queryVector := []float32{/* your query vector */}
results, err := index.Query(
    queryVector,  // query vector
    nil,          // sparseIndices (for hybrid search)
    nil,          // sparseValues (for hybrid search)
    5,            // top_k - number of results (max 4096)
    nil,          // filter (optional)
    128,          // ef - runtime parameter (max 1024)
    true,         // include_vectors
    nil,          // filterParams (optional)
    0.5,          // denseRRFWeight (0.0–1.0, used for hybrid)
    60,           // rrfRankConstant (≥1, used for hybrid)
)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("ID: %s, Similarity: %.3f\n", result.ID, result.Similarity)
    fmt.Printf("Metadata: %+v\n", result.Meta)
}
```

**Query Parameters:**

- `vector`: Query vector (must match index dimension)
- `sparseIndices`: Sparse vector indices (for hybrid search; must pair with `sparseValues`)
- `sparseValues`: Sparse vector values (for hybrid search; must pair with `sparseIndices`)
- `k`: Number of nearest neighbors to return (1–4096, default: 10)
- `filter`: Optional filter criteria (`map[string]interface{}`)
- `ef`: Runtime search parameter — higher values improve recall but increase latency (0–1024, default: 128)
- `includeVectors`: Whether to return the actual vector data in results (default: false)
- `filterParams`: Advanced filter parameters (optional, `*FilterParams`):
  - `BoostPercentage`: Expand candidate pool by X% during filtered search (0–400, default: 0)
  - `PrefilterThreshold`: Switch to brute-force when matches < threshold (0 disables; 1000–1000000, default: 10000)
- `denseRRFWeight`: RRF weight for the dense component (0.0–1.0; default: 0.5; ignored for dense-only indexes)
- `rrfRankConstant`: RRF rank constant (≥1; default: 60; ignored for dense-only indexes)

**Result Fields:**

- `ID`: Vector identifier
- `Similarity`: Similarity score
- `Distance`: Distance score (1.0 - similarity)
- `Meta`: Metadata map
- `Norm`: Vector norm
- `Filter`: Filter map (if filter was included during upsert)
- `Vector`: Vector data (if `includeVectors=true`)

## Filtered Querying

The `index.Query()` method supports structured filtering using the `filter` parameter. All filters are combined with **logical AND** — a vector must match every condition to be returned.

```go
filter := map[string]interface{}{
    "tags": map[string]interface{}{
        "$eq": "important",
    },
    "visibility": map[string]interface{}{
        "$eq": "public",
    },
}

results, err := index.Query(queryVector, nil, nil, 5, filter, 128, true, nil, 0.5, 60)
```

### Filtering Operators

| Operator | Description | Supported Type | Example Usage |
|----------|-------------|----------------|---------------|
| `$eq` | Matches values that are equal | String, Number | `{"status": {"$eq": "published"}}` |
| `$in` | Matches any value in the provided list | String | `{"tags": {"$in": []string{"ai", "ml"}}}` |
| `$range` | Matches values between start and end, inclusive | Number | `{"score": {"$range": []int{70, 95}}}` |

**Important Notes:**

- Operators are **case-sensitive** and must be prefixed with `$`
- Filters operate on fields set under `Filter` during vector upsert
- The `$range` operator supports values only within **[0 – 999]**. Normalize or scale values to fit this range prior to upserting

### Filter Examples

```go
// Equal operator - exact match
filter := map[string]interface{}{
    "status": map[string]interface{}{
        "$eq": "published",
    },
}

// In operator - match any value in list
filter = map[string]interface{}{
    "tags": map[string]interface{}{
        "$in": []string{"ai", "ml", "data-science"},
    },
}

// Range operator - numeric range (inclusive)
filter = map[string]interface{}{
    "score": map[string]interface{}{
        "$range": []int{70, 95},
    },
}

// Combined filters (AND logic)
filter = map[string]interface{}{
    "status": map[string]interface{}{
        "$eq": "published",
    },
    "tags": map[string]interface{}{
        "$in": []string{"ai", "ml"},
    },
    "score": map[string]interface{}{
        "$range": []int{80, 100},
    },
}
```

## Deletion Methods

### Vector Deletion

```go
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

// Delete by ID
result, err := index.DeleteVectorByID("vec1")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)

// Delete by filter
result, err = index.DeleteVectorByFilter(map[string]interface{}{
    "category": map[string]interface{}{"$eq": "old"},
})
```

### Index Deletion

```go
err := client.DeleteIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}
```

> **Caution:** Deletion operations are **irreversible**. Verify the correct ID or index name before proceeding.

## Additional Operations

### Get Vector by ID

```go
vector, err := index.GetVector("vec1")
if err != nil {
    log.Fatal(err)
}

// VectorItem contains: ID, Meta, Filter, Vector, SparseIndices, SparseValues
fmt.Printf("Vector: %+v\n", vector)
```

### Update Filters

The `index.UpdateFilters()` method updates filter metadata for multiple vectors without modifying vector data or other metadata. Useful when filter criteria need to change after ingestion.

```go
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

updates := []endee.FilterUpdateItem{
    {
        ID: "vec1",
        Filter: map[string]interface{}{
            "category": "B",
        },
    },
    {
        ID: "vec2",
        Filter: map[string]interface{}{
            "category": "C",
            "priority": 1,
        },
    },
}

result, err := index.UpdateFilters(updates)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result) // "2 filters updated"
```

**Parameters:**

- `updates`: Slice of `FilterUpdateItem`, each containing:
  - `ID`: Vector identifier (required, must be non-empty)
  - `Filter`: New filter metadata (replaces existing filter fields entirely)

**Notes:**
- Only filter metadata is replaced; vector data and `Meta` remain unchanged
- If a vector ID doesn't exist, the operation will fail for that update

### Describe Index (Local)

Returns a map of the index's configuration from local cache — no HTTP call required:

```go
info := index.Describe()
// keys: name, space_type, dimension, sparse_model, is_hybrid, count, precision, M, ef_con
fmt.Printf("%+v\n", info)
```

### Refresh Metadata

Re-fetches index metadata from the server and updates all local fields:

```go
meta, err := index.RefreshMetadata()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Updated count: %v\n", meta["count"])
```

### Rebuild Index

Triggers an HNSW index rebuild with optional new `M` and `efCon` parameters. The index must be non-empty.

```go
newM := 32
newEfCon := 256

result, err := index.Rebuild(&newM, &newEfCon)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", result)

// Check rebuild status
status, err := index.RebuildStatus()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%+v\n", status)
```

Pass `nil` for either parameter to keep the current value.

### Get Index Info

```go
fmt.Println(index.GetInfo())
```

## Context Support

All operations support `context.Context` for cancellation and timeouts:

```go
import (
    "context"
    "time"
)

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := client.CreateIndexWithContext(ctx, "my_index", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, "")

indexes, err := client.ListIndexesWithContext(ctx)

index, err := client.GetIndexWithContext(ctx, "my_index")

err = index.UpsertWithContext(ctx, vectors)

results, err := index.QueryWithContext(ctx, queryVector, nil, nil, 10, nil, 128, false, nil, 0.5, 60)

result, err := index.UpdateFiltersWithContext(ctx, updates)

meta, err := index.RefreshMetadataWithContext(ctx)

result, err := index.RebuildWithContext(ctx, nil, nil)

err = client.DeleteIndexWithContext(ctx, "my_index")
```

---

## API Reference

### Endee Client

| Method | Description |
|--------|-------------|
| `EndeeClient(token string) *Endee` | Initialize client with optional API token |
| `CreateIndex(name, dimension, spaceType, M, efCon, precision, version, sparseModel) error` | Create a new vector index |
| `ListIndexes() ([]IndexInfo, error)` | List all indexes in workspace |
| `DeleteIndex(name string) error` | Delete a vector index |
| `GetIndex(name string) (*Index, error)` | Get reference to a vector index |

### Index Operations

| Method | Description |
|--------|-------------|
| `Upsert(vectors []VectorItem) error` | Insert or update vectors (max 1000 per batch) |
| `Query(vector, sparseIndices, sparseValues, k, filter, ef, includeVectors, filterParams, denseRRFWeight, rrfRankConstant) ([]QueryResult, error)` | Search for similar vectors |
| `DeleteVectorByID(id string) (string, error)` | Delete a vector by ID |
| `DeleteVectorByFilter(filter map[string]interface{}) (string, error)` | Delete vectors matching a filter |
| `GetVector(id string) (VectorItem, error)` | Get a specific vector by ID |
| `UpdateFilters(updates []FilterUpdateItem) (string, error)` | Update filter metadata for multiple vectors |
| `Describe() map[string]interface{}` | Return index configuration from local cache (no HTTP) |
| `RefreshMetadata() (map[string]interface{}, error)` | Re-fetch index metadata from server |
| `Rebuild(m, efCon *int) (map[string]interface{}, error)` | Rebuild HNSW index with optional new parameters |
| `RebuildStatus() (map[string]interface{}, error)` | Get current rebuild operation status |
| `GetInfo() string` | Get index statistics and configuration |
| `String() string` | Get string representation of index |

### Data Types

```go
// VectorItem represents a vector with metadata
type VectorItem struct {
    ID            string                 `json:"id"`
    Vector        []float32              `json:"vector"`
    SparseIndices []int                  `json:"sparse_indices,omitempty"`
    SparseValues  []float32              `json:"sparse_values,omitempty"`
    Meta          map[string]interface{} `json:"meta,omitempty"`
    Filter        map[string]interface{} `json:"filter,omitempty"`
}

// QueryResult represents a search result
type QueryResult struct {
    ID         string                 `json:"id"`
    Similarity float32                `json:"similarity"`
    Distance   float32                `json:"distance"`
    Meta       map[string]interface{} `json:"meta"`
    Filter     map[string]interface{} `json:"filter,omitempty"`
    Norm       float32                `json:"norm"`
    Vector     []float32              `json:"vector,omitempty"`
}

// FilterUpdateItem represents a filter update for a single vector
type FilterUpdateItem struct {
    ID     string                 `json:"id"`
    Filter map[string]interface{} `json:"filter"`
}

// FilterParams controls advanced filtering behavior
type FilterParams struct {
    BoostPercentage    int // 0–400: expand candidate pool during filtered search
    PrefilterThreshold int // 0 disables; 1000–1000000: switch to brute-force below this
}
```

## Constants

```go
// Precision types
const (
    PrecisionBinary  = "binary"   // 1-bit binary quantization
    PrecisionFloat16 = "float16"  // 16-bit floating point
    PrecisionFloat32 = "float32"  // 32-bit floating point
    PrecisionInt16   = "int16"    // 16-bit integer quantization (default)
    PrecisionInt8    = "int8"     // 8-bit integer quantization
)

// Distance metrics
const (
    Cosine       = "cosine"  // Cosine similarity
    L2           = "l2"      // Euclidean distance
    InnerProduct = "ip"      // Inner product
)

// Sparse models for hybrid indexes
const (
    SparseModelDefault    = "default"     // Default sparse model
    SparseModelEndEeBM25  = "endee_bm25"  // BM25-based sparse model
)

// Limits
const (
    MaxDimensionAllowed    = 8000   // Maximum vector dimensionality
    MaxVectorsPerBatch     = 1000   // Maximum vectors per upsert
    MaxTopKAllowed         = 4096   // Maximum top-k results
    MaxEfSearchAllowed     = 1024   // Maximum ef parameter
    MaxIndexNameLenAllowed = 48     // Maximum index name length
)

// Defaults
const (
    DefaultM              = 16    // Default HNSW M parameter
    DefaultEfConstruction = 128   // Default ef_construction
    DefaultEfSearch       = 128   // Default ef_search
    DefaultDenseRRFWeight = 0.5   // Default RRF weight for dense component
    DefaultRRFRankConstant = 60   // Default RRF rank constant
)
```

## Error Handling

The client returns typed errors that can be inspected for specific HTTP failure conditions:

```go
import "errors"

err := client.CreateIndex("test", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, "")
if err != nil {
    var notFound *endee.NotFoundError
    var conflict *endee.ConflictError
    var authErr *endee.AuthenticationError

    switch {
    case errors.As(err, &conflict):
        fmt.Println("Index already exists")
    case errors.As(err, &notFound):
        fmt.Println("Resource not found")
    case errors.As(err, &authErr):
        fmt.Println("Invalid or missing API token")
    default:
        log.Fatal("Unexpected error:", err)
    }
}
```

**Error Types:**

| Type | HTTP Status | Description |
|------|-------------|-------------|
| `APIError` | 400 | Bad request / general API error |
| `AuthenticationError` | 401 | Invalid or missing token |
| `SubscriptionError` | 402 | Subscription limit reached |
| `ForbiddenError` | 403 | Insufficient permissions |
| `NotFoundError` | 404 | Index or vector not found |
| `ConflictError` | 409 | Resource already exists |
| `ServerError` | 5xx | Server-side error |

## Performance Features

The Go client includes several performance optimizations:

- **Connection Pooling**: Advanced HTTP connection pooling scaled to CPU cores
- **Concurrent Processing**: Automatic concurrent processing for large batches (>10 vectors)
- **Memory Pooling**: Reusable buffer pools to reduce GC pressure
- **Streaming JSON**: Fast JSON encoding/decoding with streaming
- **MessagePack**: Efficient binary serialization for vector data
- **Context Support**: Full cancellation and timeout support

## Requirements

- Go 1.24.5 or later

## Dependencies

- `github.com/vmihailenco/msgpack/v5` - Efficient binary serialization

## License

MIT License
