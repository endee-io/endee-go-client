package endee

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
)

var (
	// bufferPool reuses byte buffers to reduce GC pressure.
	bufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 0, 1024)

			return bytes.NewBuffer(buf)
		},
	}

	// mapPool reuses maps for metadata and filter parsing.
	mapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 10)
		},
	}

	// jsonEncoderPool reuses JSON encoders for streaming serialization.
	jsonEncoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewEncoder(&bytes.Buffer{})
		},
	}

	// jsonDecoderPool reuses JSON decoders for streaming deserialization.
	jsonDecoderPool = sync.Pool{
		New: func() interface{} {
			return json.NewDecoder(strings.NewReader(""))
		},
	}
)

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

func getMap() map[string]interface{} {
	m := mapPool.Get().(map[string]interface{})

	for k := range m {
		delete(m, k)
	}

	return m
}

func putMap(m map[string]interface{}) {
	if m != nil && len(m) < 100 {
		mapPool.Put(m)
	}
}

func getJSONEncoder(w *bytes.Buffer) *json.Encoder {
	_ = jsonEncoderPool.Get().(*json.Encoder)
	enc := json.NewEncoder(w)

	return enc
}

func putJSONEncoder(enc *json.Encoder) {
	jsonEncoderPool.Put(enc)
}

func getJSONDecoder(r *bytes.Reader) *json.Decoder {
	_ = jsonDecoderPool.Get().(*json.Decoder)
	dec := json.NewDecoder(r)

	return dec
}

func putJSONDecoder(dec *json.Decoder) {
	jsonDecoderPool.Put(dec)
}
