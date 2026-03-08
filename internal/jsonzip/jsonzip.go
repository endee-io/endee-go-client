// Package jsonzip provides compression utilities for JSON metadata.
package jsonzip

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
)

// Zip compresses a map into zlib-compressed JSON bytes.
func Zip(data map[string]interface{}) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var b bytes.Buffer

	w := zlib.NewWriter(&b)

	if _, err := w.Write(jsonData); err != nil {
		_ = w.Close()

		return nil, fmt.Errorf("failed to compress metadata: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compression: %w", err)
	}

	return b.Bytes(), nil
}

// Unzip decompresses zlib-compressed JSON bytes into a map.
func Unzip(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}

	defer func() { _ = r.Close() }()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress metadata: %w", err)
	}

	var result map[string]interface{}

	if err := json.Unmarshal(decompressed, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return result, nil
}
