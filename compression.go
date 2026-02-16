package endee

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"io"
)

// JsonZip compresses a map into zlib-compressed JSON bytes
func JsonZip(data map[string]interface{}) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write(jsonData); err != nil {
		_ = w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// JsonUnzip decompresses zlib-compressed JSON bytes into a map
func JsonUnzip(data []byte) (map[string]interface{}, error) {
	if len(data) == 0 {
		return make(map[string]interface{}), nil
	}

	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(decompressed, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
