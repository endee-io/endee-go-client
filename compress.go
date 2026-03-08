package endee

import "github.com/endee-io/endee-go-client/internal/jsonzip"

// JsonZip compresses a map into zlib-compressed JSON bytes.
//
// Deprecated: This function is an internal implementation detail and will be
// removed in a future version. It should not be called directly.
//
//nolint:revive,wrapcheck // deprecated wrapper; original name preserved for backward compatibility
func JsonZip(data map[string]interface{}) ([]byte, error) {
	return jsonzip.Zip(data)
}

// JsonUnzip decompresses zlib-compressed JSON bytes into a map.
//
// Deprecated: This function is an internal implementation detail and will be
// removed in a future version. It should not be called directly.
//
//nolint:revive,wrapcheck // deprecated wrapper; original name preserved for backward compatibility
func JsonUnzip(data []byte) (map[string]interface{}, error) {
	return jsonzip.Unzip(data)
}
