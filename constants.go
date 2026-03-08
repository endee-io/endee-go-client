package endee

import "time"

// Precision types for vector indices (quantization levels).
const (
	PrecisionBinary  = "binary"  // Binary vectors (1 bit per dimension)
	PrecisionFloat16 = "float16" // 16-bit floating point
	PrecisionFloat32 = "float32" // 32-bit floating point
	PrecisionInt16   = "int16"   // 16-bit integer (default)
	PrecisionInt8    = "int8"    // 8-bit integer (memory efficient)
)

// PrecisionTypesSupported lists all supported precision types.
var PrecisionTypesSupported = []string{
	PrecisionBinary,
	PrecisionFloat16,
	PrecisionFloat32,
	PrecisionInt16,
	PrecisionInt8,
}

// Checksum value used when creating an index.
const Checksum = -1

// HTTP protocol prefixes.
const (
	HTTPSProtocol = "https://"
	HTTPProtocol  = "http://"
)

// HTTPMethodsAllowed defines the allowed HTTP methods for API requests.
var HTTPMethodsAllowed = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// HTTPStatusCodes defines status codes that trigger automatic retries.
var HTTPStatusCodes = []int{429, 500, 502, 503, 504}

// API endpoints.
const (
	LocalBaseURL     = "http://127.0.0.1:8080/api/v1"
	CloudURLTemplate = "https://%s.endee.io/api/v1"
	LocalRegion      = "local"
)

// Vector index limits.
const (
	MaxDimensionAllowed    = 10000 // Maximum vector dimensionality
	MaxVectorsPerBatch     = 1000  // Maximum vectors per batch operation
	MaxTopKAllowed         = 512   // Maximum nearest neighbors (top-k)
	MaxEfSearchAllowed     = 1024  // Maximum ef_search parameter
	MaxIndexNameLenAllowed = 48    // Maximum index name length
)

// Distance metric types.
const (
	Cosine       = "cosine" // Cosine similarity (normalized dot product)
	L2           = "l2"     // Euclidean distance (L2 norm)
	InnerProduct = "ip"     // Inner product (dot product)
)

// SpaceTypesSupported lists all supported distance/space types.
var SpaceTypesSupported = []string{Cosine, L2, InnerProduct}

// API field name constants used in JSON tags.
const (
	AuthorizationHeader = "Authorization"
	NameField           = "name"
	SpaceTypeField      = "space_type"
	DimensionField      = "dimension"
	SparseDimField      = "sparse_dim"
	IsHybridField       = "is_hybrid"
	CountField          = "count"
	PrecisionField      = "precision"
	MaxConnectionsField = "M"
)

// HNSW algorithm defaults.
const (
	DefaultM               = 16               // Bi-directional links per node in HNSW graph
	DefaultEfConstruction  = 128              // Candidate list size during index construction
	DefaultSparseDimension = 0                // 0 = dense-only; > 0 = hybrid index
	DefaultTopK            = 10               // Default nearest neighbors to return
	DefaultEfSearch        = 128              // Candidate list size during search
	DefaultTimeout         = 30 * time.Second // Default HTTP request timeout
)
