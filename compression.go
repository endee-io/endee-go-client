package endee

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
)

// JSONZip compresses a map into zlib-compressed JSON bytes.
func JSONZip(data map[string]interface{}) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(jsonData); err != nil {
		_ = w.Close()

		return nil, fmt.Errorf("failed to write compressed data: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zlib writer: %w", err)
	}

	return b.Bytes(), nil
}

// JSONUnzip decompresses zlib-compressed JSON bytes into a map.
func JSONUnzip(data []byte) (map[string]interface{}, error) {
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
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(decompressed, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}
