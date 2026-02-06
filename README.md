# Endee - High-Performance Vector Database (Go Client)

Endee is a high-performance vector database designed for speed and efficiency. The Go client enables rapid Approximate Nearest Neighbor (ANN) searches for applications requiring robust vector search capabilities with advanced filtering, metadata support, and hybrid search combining dense and sparse vectors.

## Key Features

- **Fast ANN Searches**: Efficient similarity searches on vector data using HNSW algorithm
- **Hybrid Search**: Combine dense and sparse vectors for powerful semantic + keyword search
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
    
    // Create a new index
    err = client.CreateIndex(
        "my_vectors",     // name
        768,             // dimension
        "cosine",         // space_type (cosine, l2, ip)
        16,               // M - HNSW connectivity parameter
        128,              // ef_con - construction parameter
        endee.PrecisionFloat32, // precision (float32, float16, int16d, int8d, binary)
        nil,              // version (optional)
        0,                // sparse_dim (optional, 0 for dense-only)
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
    
    results, err := index.Query(queryVector, nil, nil, 10, filter, 128, false)
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

### 🔐 Generate Your API Token

- Each token is tied to your workspace and should be kept private
- Once you have your token, you're ready to initialize the client and begin using the SDK

### Initializing the Client

The Endee client acts as the main interface for all vector operations — such as creating indexes, upserting vectors, and running similarity queries. You can initialize the client in just a few lines:

```go
import "github.com/EndeeLabs/endee-go-client"

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
client.BaseUrl = "http://0.0.0.0:8081/api/v1"
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

The `client.CreateIndex()` method initializes a new vector index with customizable parameters such as dimensionality, distance metric, graph construction settings, and precision level. These configurations determine how the index stores and retrieves high-dimensional vector data.

```go
client := endee.EndeeClient("your-token-here")

// Create an index with custom parameters
err := client.CreateIndex(
    "my_custom_index",  // name
    768,                // dimension
    "cosine",           // space_type
    16,                 // M (graph connectivity, default = 16)
    128,                // ef_con (construction parameter, default = 128)
    endee.PrecisionFloat32, // precision
    nil,                // version (optional)
    0,                  // sparse_dim (optional, 0 for dense-only)
)
if err != nil {
    log.Fatal(err)
}
```

**Parameters:**

- `name`: Unique name for your index (alphanumeric + underscores, max 48 chars)
- `dimension`: Vector dimensionality (must match your embedding model's output, max 10000)
- `spaceType`: Distance metric - `"cosine"`, `"l2"`, or `"ip"` (inner product)
- `M`: HNSW graph connectivity parameter - higher values increase recall but use more memory (default: 16)
- `efCon`: HNSW construction parameter - higher values improve index quality but slow down indexing (default: 128)
- `precision`: Support for multiple precision levels - `PrecisionFloat32`, `PrecisionFloat16`, `PrecisionInt16D`, `PrecisionInt8D`, `PrecisionBinary`
- `version`: Optional version parameter for index versioning
- `sparseDim`: Dimension for sparse vectors (0 for dense-only)

**Precision Levels:**

The Go client supports various precision levels for memory/accuracy tradeoffs:

| Precision | Constant | Data Type | Memory Usage | Use Case |
|-----------|----------|-----------|--------------|----------|
| FP32 (default) | `PrecisionFloat32` | 32-bit float | Highest | Maximum accuracy |
| FP16 | `PrecisionFloat16` | 16-bit float | ~50% less | Good accuracy, lower memory |
| INT16 | `PrecisionInt16D` | 16-bit int | Optimized | Quantized accuracy |
| INT8 | `PrecisionInt8D` | 8-bit int | ~75% less | Maximum memory savings |
| Binary | `PrecisionBinary` | 1-bit | Minimum | Fast, low-memory keyword-like search |

**Example with different precision levels:**

```go
client := endee.EndeeClient("your-token-here")

// High accuracy index (FP32)
err := client.CreateIndex("high_accuracy_index", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, 0)

// Memory-optimized index (INT8)
err = client.CreateIndex("low_memory_index", 768, "cosine", 16, 128, endee.PrecisionInt8D, nil, 0)
```

### Get an Index

The `client.GetIndex()` method retrieves a reference to an existing index. This is required before performing vector operations like upsert, query, or delete.

```go
client := endee.EndeeClient("your-token-here")

// Get reference to an existing index
index, err := client.GetIndex("my_custom_index")
if err != nil {
    log.Fatal(err)
}

// Now you can perform operations on the index
fmt.Println(index.GetInfo())
```

**Parameters:**

- `name`: Name of the index to retrieve

**Returns:** An `*Index` instance configured with server parameters

### Ingestion of Data

The `index.Upsert()` method is used to add or update vectors (embeddings) in an existing index. Each vector is represented as a `VectorItem` containing a unique identifier, the vector data itself, optional metadata, and optional filter fields for future querying.

```go
client := endee.EndeeClient("your-token-here")

// Accessing the index
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

// Insert multiple vectors in a batch
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

- `ID`: Unique identifier for the vector (required)
- `Vector`: Slice of float32 representing the embedding (required)
- `Meta`: Map for storing additional information (optional)
- `Filter`: Map with key-value pairs for structured filtering during queries (optional)

> **Note:** Maximum batch size is 1000 vectors per upsert call.

### Querying the Index

The `index.Query()` method performs a similarity search in the index using a given query vector. It returns the closest vectors (based on the index's distance metric) along with optional metadata and vector data.

```go
client := endee.EndeeClient("your-token-here")

// Accessing the index
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

// Query with custom parameters
queryVector := []float32{/* your query vector */}
results, err := index.Query(
    queryVector,  // query vector
    5,            // top_k - number of results (max 512)
    nil,          // filter (optional)
    128,          // ef - runtime parameter (max 1024)
    true,         // include_vectors
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
- `k`: Number of nearest neighbors to return (max 512, default: 10)
- `filter`: Optional filter criteria (map[string]interface{})
- `ef`: Runtime search parameter - higher values improve recall but increase latency (max 1024, default: 128)
- `includeVectors`: Whether to return the actual vector data in results (default: false)

**Result Fields:**

- `ID`: Vector identifier
- `Similarity`: Similarity score
- `Distance`: Distance score (1.0 - similarity)
- `Meta`: Metadata map
- `Norm`: Vector norm
- `Filter`: Filter map (if filter was included during upsert)
- `Vector`: Vector data (if `includeVectors=true`)

## Filtered Querying

The `index.Query()` method supports structured filtering using the `filter` parameter. This allows you to restrict search results based on metadata conditions, in addition to vector similarity.

To apply multiple filter conditions, pass a map with filter objects, where each key defines a field and value defines the condition. **All filters are combined with logical AND** — meaning a vector must match all specified conditions to be included in the results.

```go
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

// Query with multiple filter conditions (AND logic)
filter := map[string]interface{}{
    "tags": map[string]interface{}{
        "$eq": "important",
    },
    "visibility": map[string]interface{}{
        "$eq": "public",
    },
}

results, err := index.Query(queryVector, 5, filter, 128, true)
if err != nil {
    log.Fatal(err)
}
```

### Filtering Operators

The `filter` parameter in `index.Query()` supports a range of comparison operators to build structured queries. These operators allow you to include or exclude vectors based on metadata or custom fields.

| Operator | Description | Supported Type | Example Usage |
|----------|-------------|----------------|---------------|
| `$eq` | Matches values that are equal | String, Number | `{"status": {"$eq": "published"}}` |
| `$in` | Matches any value in the provided list | String | `{"tags": {"$in": []string{"ai", "ml"}}}` |
| `$range` | Matches values between start and end, inclusive | Number | `{"score": {"$range": []int{70, 95}}}` |

**Important Notes:**

- Operators are **case-sensitive** and must be prefixed with a `$`
- Filters operate on fields provided under the `Filter` key during vector upsert
- The `$range` operator supports values only within the range **[0 – 999]**. If your data exceeds this range (e.g., timestamps, large scores), you should normalize or scale your values to fit within [0, 999] prior to upserting or querying

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

The system supports two types of deletion operations — **vector deletion** and **index deletion**. These allow you to remove specific vectors or entire indexes from your workspace, giving you full control over lifecycle and storage.

### Vector Deletion

Vector deletion is used to remove specific vectors from an index using their unique `ID`. This is useful when:

- A document is outdated or revoked
- You want to update a vector by first deleting its old version
- You're cleaning up test data or low-quality entries

```go
client := endee.EndeeClient("your-token-here")
index, err := client.GetIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}

// Delete a single vector by ID
result, err := index.DeleteVector("vec1")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
```

### Index Deletion

Index deletion permanently removes the entire index and all vectors associated with it. This should be used when:

- The index is no longer needed
- You want to re-create the index with a different configuration
- You're managing index rotation in batch pipelines

```go
client := endee.EndeeClient("your-token-here")

// Delete an entire index
err := client.DeleteIndex("your-index-name")
if err != nil {
    log.Fatal(err)
}
```

> ⚠️ **Caution:** Deletion operations are **irreversible**. Ensure you have the correct `ID` or index name before performing deletion, especially at the index level.

## Additional Operations

### Get Vector by ID

The `index.GetVector()` method retrieves a specific vector from the index by its unique identifier.

```go
// Retrieve a specific vector by its ID
vector, err := index.GetVector("vec1")
if err != nil {
    log.Fatal(err)
}

// The returned VectorItem contains:
// - ID: Vector identifier
// - Meta: Metadata map
// - Filter: Filter fields map
// - Vector: Vector data array
fmt.Printf("Vector: %+v\n", vector)
```

### Get Index Info

```go
// Get index statistics and configuration info
info := index.GetInfo()
fmt.Println(info)
```

## Context Support

All operations support Go's `context.Context` for cancellation and timeouts:

```go
import (
    "context"
    "time"
)

// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use context with operations
err := client.CreateIndexWithContext(ctx, "my_index", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, 0)

indexes, err := client.ListIndexesWithContext(ctx)

index, err := client.GetIndexWithContext(ctx, "my_index")

err = index.UpsertWithContext(ctx, vectors)

results, err := index.QueryWithContext(ctx, queryVector, 10, nil, 128, false)

err = client.DeleteIndexWithContext(ctx, "my_index")
```

---

## API Reference

### Endee Client

| Method | Description |
|--------|-------------|
| `EndeeClient(token string) *Endee` | Initialize client with optional API token |
| `CreateIndex(name, dimension, spaceType, M, efCon, precision, version, sparseDim) error` | Create a new vector index |
| `CreateIndexWithContext(ctx, name, dimension, spaceType, M, efCon, precision, version, sparseDim) error` | Create index with context support |
| `ListIndexes() ([]interface{}, error)` | List all indexes in workspace |
| `ListIndexesWithContext(ctx) ([]interface{}, error)` | List indexes with context support |
| `DeleteIndex(name string) error` | Delete a vector index |
| `DeleteIndexWithContext(ctx, name) error` | Delete index with context support |
| `GetIndex(name string) (*Index, error)` | Get reference to a vector index |
| `GetIndexWithContext(ctx, name) (*Index, error)` | Get index with context support |

### Index Operations

| Method | Description |
|--------|-------------|
| `Upsert(vectors []VectorItem) error` | Insert or update vectors (max 1000 per batch) |
| `UpsertWithContext(ctx, vectors) error` | Upsert with context support |
| `Query(vector, sparseIndices, sparseValues, k, filter, ef, includeVectors) ([]QueryResult, error)` | Search for similar vectors |
| `QueryWithContext(ctx, vector, sparseIndices, sparseValues, k, filter, ef, includeVectors) ([]QueryResult, error)` | Query with context support |
| `DeleteVectorById(id string) (string, error)` | Delete a vector by ID |
| `DeleteVectorByIdWithContext(ctx, id) (string, error)` | Delete vector with context support |
| `DeleteVectorByFilter(filter map[string]interface{}) (string, error)` | Delete vectors matching a specific filter |
| `DeleteVectorByFilterWithContext(ctx, filter) (string, error)` | Delete vectors matching a specific filter with context support |
| `DeleteHybridVectorById(id string) (string, error)` | Delete a hybrid vector by ID |
| `DeleteHybridVectorByIdWithContext(ctx, id) (string, error)` | Delete hybrid vector with context support |
| `DeleteHybridVectorByFilter(filter map[string]interface{}) (string, error)` | Delete hybrid vectors matching a specific filter |
| `DeleteHybridVectorByFilterWithContext(ctx, filter) (string, error)` | Delete hybrid vectors matching a specific filter with context support |
| `GetVector(id string) (VectorItem, error)` | Get a specific vector by ID |
| `GetVectorWithContext(ctx, id) (VectorItem, error)` | Get vector with context support |
| `GetInfo() string` | Get index statistics and configuration |
| `String() string` | Get string representation of index |

### Data Types

```go
// VectorItem represents a vector with metadata
type VectorItem struct {
    ID     string                 `json:"id"`
    Vector []float32              `json:"vector"`
    Meta   map[string]interface{} `json:"meta,omitempty"`
    Filter map[string]interface{} `json:"filter,omitempty"`
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
```

## Constants

The package provides useful constants for configuration:

```go
// Precision types
const (
    PrecisionBinary  = "binary"   // 1-bit binary quantization
    PrecisionFloat16 = "float16"  // 16-bit floating point
    PrecisionFloat32 = "float32"  // 32-bit floating point
    PrecisionInt16D  = "int16d"   // 16-bit integer quantization
    PrecisionInt8D   = "int8d"    // 8-bit integer quantization
)

// Distance metrics
const (
    Cosine       = "cosine"  // Cosine similarity
    L2           = "l2"      // Euclidean distance
    InnerProduct = "ip"      // Inner product
)

// Limits
const (
    MaxDimensionAllowed    = 10000  // Maximum vector dimensionality
    MaxVectorsPerBatch     = 1000   // Maximum vectors per upsert
    MaxTopKAllowed         = 512    // Maximum top-k results
    MaxEfSearchAllowed     = 1024   // Maximum ef parameter
    MaxIndexNameLenAllowed = 48     // Maximum index name length
)

// Defaults
const (
    DefaultM              = 16   // Default HNSW M parameter
    DefaultEfConstruction = 128  // Default ef_construction
    DefaultTopK           = 10   // Default top-k
    DefaultEfSearch       = 128  // Default ef_search
)
```

## Performance Features

The Go client includes several performance optimizations:

- **Connection Pooling**: Advanced HTTP connection pooling scaled to CPU cores
- **Concurrent Processing**: Automatic concurrent processing for large batches
- **Memory Pooling**: Reusable buffer pools to reduce GC pressure
- **Streaming JSON**: Fast JSON encoding/decoding with streaming
- **MessagePack**: Efficient binary serialization for vector data
- **Context Support**: Full cancellation and timeout support

## Error Handling

```go
// Handle common errors
err := client.CreateIndex("test", 768, "cosine", 16, 128, endee.PrecisionFloat32, nil, 0)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "already exists"):
        fmt.Println("Index already exists, continuing...")
    case strings.Contains(err.Error(), "invalid index name"):
        fmt.Println("Invalid index name format")
    case strings.Contains(err.Error(), "dimension cannot be greater"):
        fmt.Println("Dimension too large")
    default:
        log.Fatal("Unexpected error:", err)
    }
}
```

## Requirements

- Go 1.24.5 or later

## Dependencies

- `github.com/vmihailenco/msgpack/v5` - Efficient binary serialization

## License

MIT License
