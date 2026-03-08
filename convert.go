package endee

import "fmt"

// safeStringConvert safely converts an interface{} value to string.
func safeStringConvert(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []uint8:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// toFloat32 safely converts an interface{} value to float32.
func toFloat32(val interface{}) float32 {
	if val == nil {
		return 0.0
	}
	switch v := val.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int:
		return float32(v)
	case int8:
		return float32(v)
	case int16:
		return float32(v)
	case int32:
		return float32(v)
	case int64:
		return float32(v)
	case uint8:
		return float32(v)
	case uint16:
		return float32(v)
	case uint32:
		return float32(v)
	case uint64:
		return float32(v)
	default:
		return 0.0
	}
}
