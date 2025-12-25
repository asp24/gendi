package parameters

import (
	"fmt"
	"strconv"
	"time"
)

// convertToString converts a value to string, with type coercion for numeric types.
func convertToString(val interface{}) (string, error) {
	switch cVal := val.(type) {
	case int8:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int16:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int32:
		return strconv.FormatInt(int64(cVal), 10), nil
	case int64:
		return strconv.FormatInt(cVal, 10), nil
	case int:
		return strconv.FormatInt(int64(cVal), 10), nil
	case uint:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(cVal), 10), nil
	case uint64:
		return strconv.FormatUint(cVal, 10), nil
	case float32:
		return strconv.FormatFloat(float64(cVal), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(cVal, 'f', -1, 64), nil
	case string:
		return cVal, nil
	default:
		return "", fmt.Errorf("expected string, got %T", val)
	}
}

// convertToInt converts a value to int, with type coercion for various int types.
func convertToInt(val interface{}) (int, error) {
	switch cVal := val.(type) {
	case int8:
		return int(cVal), nil
	case int16:
		return int(cVal), nil
	case int32:
		return int(cVal), nil
	case int64:
		return int(cVal), nil
	case int:
		return cVal, nil
	default:
		return 0, fmt.Errorf("expected int, got %T", val)
	}
}

// convertToBool converts a value to bool with strict type checking.
func convertToBool(val interface{}) (bool, error) {
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("expected bool, got %T", val)
	}
	return b, nil
}

// convertToFloat converts a value to float64 with strict type checking.
func convertToFloat(val interface{}) (float64, error) {
	f, ok := val.(float64)
	if !ok {
		return 0, fmt.Errorf("expected float64, got %T", val)
	}
	return f, nil
}

// convertToDuration converts a value to time.Duration, supporting multiple input types.
func convertToDuration(val interface{}) (time.Duration, error) {
	switch cVal := val.(type) {
	case time.Duration:
		return cVal, nil
	case string:
		parsed, err := time.ParseDuration(cVal)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %w", err)
		}
		return parsed, nil
	case int:
		return time.Duration(cVal), nil
	case int8:
		return time.Duration(cVal), nil
	case int16:
		return time.Duration(cVal), nil
	case int32:
		return time.Duration(cVal), nil
	case int64:
		return time.Duration(cVal), nil
	case uint:
		return time.Duration(cVal), nil
	case uint8:
		return time.Duration(cVal), nil
	case uint16:
		return time.Duration(cVal), nil
	case uint32:
		return time.Duration(cVal), nil
	case uint64:
		if cVal > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("duration overflows int64")
		}
		return time.Duration(cVal), nil
	default:
		return 0, fmt.Errorf("expected duration, got %T", val)
	}
}
