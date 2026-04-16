package endee

import (
	"regexp"
	"time"
)

// Precision types for vector indices (quantization levels).
// Defines the data types available for storing vector embeddings.
const (
	PrecisionBinary  = "binary"  // Binary vectors (1 bit per dimension)
	PrecisionFloat16 = "float16" // 16-bit floating point
	PrecisionFloat32 = "float32" // 32-bit floating point
	PrecisionInt16   = "int16"   // 16-bit integer
	PrecisionInt8    = "int8"    // 8-bit integer
)

// PrecisionTypesSupported lists all supported precision types.
var PrecisionTypesSupported = []string{
	PrecisionBinary,
	PrecisionFloat16,
	PrecisionFloat32,
	PrecisionInt16,
	PrecisionInt8,
}

// Checksum value while creating an index.
const Checksum = -1

// HTTP Configuration.
const (
	HTTPSProtocol = "https://"
	HTTPProtocol  = "http://"
)

// HTTPMethodsAllowed defines allowed HTTP methods for API requests.
var HTTPMethodsAllowed = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

// HTTPStatusCodes defines status codes that trigger automatic retries.
var HTTPStatusCodes = []int{429, 500, 502, 503, 504}

// API Endpoints.
const (
	LocalBaseURL     = "http://127.0.0.1:8080/api/v1"
	CloudURLTemplate = "https://%s.endee.io/api/v1"
	LocalRegion      = "local"
)

// Vector Index Limits.
const (
	MaxDimensionAllowed    = 8000  // Maximum vector dimensionality allowed
	MaxVectorsPerBatch     = 1000  // Maximum number of vectors in a single batch operation
	MaxTopKAllowed         = 4096  // Maximum number of nearest neighbors (top-k) that can be retrieved
	MaxEfSearchAllowed     = 1024  // Maximum ef_search parameter for HNSW query accuracy
	MaxIndexNameLenAllowed = 48    // Maximum length for index names (alphanumeric + underscores)

	// Filter key/value size limits.
	MaxKeyBytes   = 128  // Maximum bytes for a filter key name
	MaxValueBytes = 1024 // Maximum bytes for a string filter value
)

// Distance metric types.
const (
	Cosine       = "cosine" // Cosine similarity (normalized dot product)
	L2           = "l2"     // Euclidean distance (L2 norm)
	InnerProduct = "ip"     // Inner product (dot product)
)

// SpaceTypesSupported lists all supported distance/space types.
var SpaceTypesSupported = []string{Cosine, L2, InnerProduct}

// Sparse model types.
const (
	SparseModelDefault   = "default"    // Sparse search without server-side IDF
	SparseModelEndEeBM25 = "endee_bm25" // BM25 scoring with server-side IDF
)

// API Field Names.
// Common field names used in API requests/responses.
const (
	AuthorizationHeader = "Authorization"
	NameField           = "name"
	SpaceTypeField      = "space_type"
	DimensionField      = "dimension"
	SparseModelField    = "sparse_model"
	IsHybridField       = "is_hybrid"
	CountField          = "count"
	PrecisionField      = "precision"
	MaxConnectionsField = "M"
)

// Session Configuration (for HTTP clients).
const (
	// Requests Library Session Configuration.
	SessionPoolConnections = 1  // Number of connection pools to cache (one per unique host)
	SessionPoolMaxSize     = 10 // Maximum number of connections to save in each pool for reuse
	SessionMaxRetries      = 3  // Maximum number of retry attempts for failed requests

	// HTTPX Library Client Configuration.
	HTTPXMaxConnections          = 1    // Maximum total connections across all hosts
	HTTPXMaxKeepaliveConnections = 10   // Maximum idle connections to keep alive for reuse
	HTTPXMaxRetries              = 3    // Maximum number of retry attempts for failed requests
	HTTPXTimeoutSec              = 30.0 // Request timeout in seconds (prevents hanging requests)
)

// HNSW Algorithm Defaults.
const (
	DefaultM                              = 16  // Default M parameter: number of bi-directional links per node in HNSW graph
	DefaultEfConstruction                 = 128 // Default ef_construction: size of dynamic candidate list during index construction
	DefaultTopK                           = 10  // Default number of nearest neighbors to return in search queries
	DefaultEfSearch                       = 128 // Default ef_search: size of dynamic candidate list during search
	DefaultTimeout                        = 30 * time.Second
	DefaultPrefilterCardinalityThreshold  = 10000 // Switches from HNSW to brute-force for filtered queries
	DefaultFilterBoostPercentage          = 0     // Expands HNSW candidate pool by this percentage
	DefaultDenseRRFWeight         float64 = 0.5   // RRF weight for the dense component in hybrid search
	DefaultRRFRankConstant                = 60    // RRF rank constant in hybrid search
)

// NameRegex is a pre-compiled regex for validating index names.
var NameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
