package parameters

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Caster converts raw parameter values returned by Provider.Lookup to the
// scalar type required by a constructor argument. The generated container
// selects the method from each injection site's target type, so one runtime
// value can be requested through different conversions.
type Caster interface {
	ToString(value any) (string, error)
	ToBool(value any) (bool, error)

	ToInt(value any) (int, error)
	ToInt8(value any) (int8, error)
	ToInt16(value any) (int16, error)
	ToInt32(value any) (int32, error)
	ToInt64(value any) (int64, error)

	ToUint(value any) (uint, error)
	ToUint8(value any) (uint8, error)
	ToUint16(value any) (uint16, error)
	ToUint32(value any) (uint32, error)
	ToUint64(value any) (uint64, error)

	ToFloat32(value any) (float32, error)
	ToFloat64(value any) (float64, error)

	ToDuration(value any) (time.Duration, error)
	ToTime(value any) (time.Time, error)
}

// StandardCaster is the default conversion policy. It accepts raw values of
// base scalar kinds plus exact time.Time and time.Duration, and rejects any
// conversion that could lose information: float to integer, bool to anything,
// values that overflow the target, inexact integer/float conversions, NaN and
// infinities, and named input types. Custom casters can embed it and override
// only the conversions whose policy differs.
type StandardCaster struct{}

var _ Caster = StandardCaster{}

func (c StandardCaster) ToString(value any) (string, error) {
	switch cv := value.(type) {
	case string:
		return cv, nil
	case int:
		return strconv.FormatInt(int64(cv), 10), nil
	case int8:
		return strconv.FormatInt(int64(cv), 10), nil
	case int16:
		return strconv.FormatInt(int64(cv), 10), nil
	case int32:
		return strconv.FormatInt(int64(cv), 10), nil
	case int64:
		return strconv.FormatInt(cv, 10), nil
	case uint:
		return strconv.FormatUint(uint64(cv), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(cv), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(cv), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(cv), 10), nil
	case uint64:
		return strconv.FormatUint(cv, 10), nil
	case float32:
		if err := c.rejectNonFinite(float64(cv), value, "string"); err != nil {
			return "", err
		}
		return strconv.FormatFloat(float64(cv), 'g', -1, 32), nil
	case float64:
		if err := c.rejectNonFinite(cv, value, "string"); err != nil {
			return "", err
		}
		return strconv.FormatFloat(cv, 'g', -1, 64), nil
	default:
		return "", NewCastError(value, "string")
	}
}

func (c StandardCaster) ToBool(value any) (bool, error) {
	switch cv := value.(type) {
	case bool:
		return cv, nil
	case string:
		b, err := strconv.ParseBool(cv)
		if err != nil {
			return false, fmt.Errorf("%s: %w", NewCastError(value, "bool"), err)
		}
		return b, nil
	default:
		return false, NewCastError(value, "bool")
	}
}

func (c StandardCaster) ToInt(value any) (int, error) {
	v, err := c.toSigned(value, "int", math.MinInt, math.MaxInt)
	return int(v), err
}

func (c StandardCaster) ToInt8(value any) (int8, error) {
	v, err := c.toSigned(value, "int8", math.MinInt8, math.MaxInt8)
	return int8(v), err
}

func (c StandardCaster) ToInt16(value any) (int16, error) {
	v, err := c.toSigned(value, "int16", math.MinInt16, math.MaxInt16)
	return int16(v), err
}

func (c StandardCaster) ToInt32(value any) (int32, error) {
	v, err := c.toSigned(value, "int32", math.MinInt32, math.MaxInt32)
	return int32(v), err
}

func (c StandardCaster) ToInt64(value any) (int64, error) {
	return c.toSigned(value, "int64", math.MinInt64, math.MaxInt64)
}

func (c StandardCaster) ToUint(value any) (uint, error) {
	v, err := c.toUnsigned(value, "uint", math.MaxUint)
	return uint(v), err
}

func (c StandardCaster) ToUint8(value any) (uint8, error) {
	v, err := c.toUnsigned(value, "uint8", math.MaxUint8)
	return uint8(v), err
}

func (c StandardCaster) ToUint16(value any) (uint16, error) {
	v, err := c.toUnsigned(value, "uint16", math.MaxUint16)
	return uint16(v), err
}

func (c StandardCaster) ToUint32(value any) (uint32, error) {
	v, err := c.toUnsigned(value, "uint32", math.MaxUint32)
	return uint32(v), err
}

func (c StandardCaster) ToUint64(value any) (uint64, error) {
	return c.toUnsigned(value, "uint64", math.MaxUint64)
}

func (c StandardCaster) ToFloat32(value any) (float32, error) {
	f, err := c.toFloat(value, "float32", 32)
	return float32(f), err
}

func (c StandardCaster) ToFloat64(value any) (float64, error) {
	return c.toFloat(value, "float64", 64)
}

func (c StandardCaster) ToDuration(value any) (time.Duration, error) {
	switch cv := value.(type) {
	case time.Duration:
		return cv, nil
	case string:
		d, err := time.ParseDuration(cv)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", NewCastError(value, "time.Duration"), err)
		}
		return d, nil
	case float32, float64, bool, time.Time:
		return 0, NewCastError(value, "time.Duration")
	default:
		v, err := c.toSigned(value, "time.Duration", math.MinInt64, math.MaxInt64)
		if err != nil {
			return 0, err
		}
		return time.Duration(v), nil
	}
}

func (c StandardCaster) ToTime(value any) (time.Time, error) {
	switch cv := value.(type) {
	case time.Time:
		return cv, nil
	case string:
		ts, err := time.Parse(time.RFC3339, cv)
		if err != nil {
			return time.Time{}, fmt.Errorf("%s: %w", NewCastError(value, "time.Time"), err)
		}
		return ts, nil
	default:
		return time.Time{}, NewCastError(value, "time.Time")
	}
}

// toSigned converts to a signed integer family value with range checking.
// Durations convert as nanoseconds.
func (c StandardCaster) toSigned(value any, target string, min, max int64) (int64, error) {
	var v int64
	switch cv := value.(type) {
	case int:
		v = int64(cv)
	case int8:
		v = int64(cv)
	case int16:
		v = int64(cv)
	case int32:
		v = int64(cv)
	case int64:
		v = cv
	case time.Duration:
		v = int64(cv)
	case uint:
		if uint64(cv) > math.MaxInt64 {
			return 0, c.overflowError(value, target)
		}
		v = int64(cv)
	case uint8:
		v = int64(cv)
	case uint16:
		v = int64(cv)
	case uint32:
		v = int64(cv)
	case uint64:
		if cv > math.MaxInt64 {
			return 0, c.overflowError(value, target)
		}
		v = int64(cv)
	case string:
		parsed, err := strconv.ParseInt(cv, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", NewCastError(value, target), err)
		}
		v = parsed
	default:
		return 0, NewCastError(value, target)
	}
	if v < min || v > max {
		return 0, c.overflowError(value, target)
	}
	return v, nil
}

// toUnsigned converts to an unsigned integer family value with sign and
// range checking. Durations convert as nanoseconds.
func (c StandardCaster) toUnsigned(value any, target string, max uint64) (uint64, error) {
	var v uint64
	switch cv := value.(type) {
	case uint:
		v = uint64(cv)
	case uint8:
		v = uint64(cv)
	case uint16:
		v = uint64(cv)
	case uint32:
		v = uint64(cv)
	case uint64:
		v = cv
	case int:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case int8:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case int16:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case int32:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case int64:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case time.Duration:
		if cv < 0 {
			return 0, c.negativeError(value, target)
		}
		v = uint64(cv)
	case string:
		parsed, err := strconv.ParseUint(cv, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", NewCastError(value, target), err)
		}
		v = parsed
	default:
		return 0, NewCastError(value, target)
	}
	if v > max {
		return 0, c.overflowError(value, target)
	}
	return v, nil
}

// toFloat converts to a float value. Integer inputs must be exactly
// representable; bitSize 32 additionally requires the value to survive
// narrowing to float32 (strings are parsed directly at float32 precision).
func (c StandardCaster) toFloat(value any, target string, bitSize int) (float64, error) {
	var f float64
	switch cv := value.(type) {
	case float64:
		f = cv
	case float32:
		f = float64(cv)
	case int:
		var err error
		if f, err = c.exactSignedFloat(int64(cv), value, target); err != nil {
			return 0, err
		}
	case int8:
		f = float64(cv)
	case int16:
		f = float64(cv)
	case int32:
		f = float64(cv)
	case int64:
		var err error
		if f, err = c.exactSignedFloat(cv, value, target); err != nil {
			return 0, err
		}
	case uint:
		var err error
		if f, err = c.exactUnsignedFloat(uint64(cv), value, target); err != nil {
			return 0, err
		}
	case uint8:
		f = float64(cv)
	case uint16:
		f = float64(cv)
	case uint32:
		f = float64(cv)
	case uint64:
		var err error
		if f, err = c.exactUnsignedFloat(cv, value, target); err != nil {
			return 0, err
		}
	case string:
		parsed, err := strconv.ParseFloat(cv, bitSize)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", NewCastError(value, target), err)
		}
		f = parsed
	default:
		return 0, NewCastError(value, target)
	}
	if err := c.rejectNonFinite(f, value, target); err != nil {
		return 0, err
	}
	if bitSize == 32 {
		if _, isString := value.(string); !isString {
			if float64(float32(f)) != f {
				return 0, c.inexactError(value, target)
			}
		}
	}
	return f, nil
}

func (c StandardCaster) exactSignedFloat(v int64, raw any, target string) (float64, error) {
	f := float64(v)
	if f >= -9223372036854775808.0 && f < 9223372036854775808.0 && int64(f) == v {
		return f, nil
	}
	return 0, c.inexactError(raw, target)
}

func (c StandardCaster) exactUnsignedFloat(v uint64, raw any, target string) (float64, error) {
	f := float64(v)
	if f < 18446744073709551616.0 && uint64(f) == v {
		return f, nil
	}
	return 0, c.inexactError(raw, target)
}

func (c StandardCaster) rejectNonFinite(f float64, raw any, target string) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("%s: NaN and infinities are rejected", NewCastError(raw, target))
	}
	return nil
}

func (c StandardCaster) overflowError(value any, target string) error {
	return fmt.Errorf("%s: value overflows %s", NewCastError(value, target), target)
}

func (c StandardCaster) negativeError(value any, target string) error {
	return fmt.Errorf("%s: negative value", NewCastError(value, target))
}

func (c StandardCaster) inexactError(value any, target string) error {
	return fmt.Errorf("%s: value is not exactly representable", NewCastError(value, target))
}

// NewCastError reports that a raw parameter value cannot be converted to the
// target type, naming both the value's dynamic type and the target. Custom
// Caster implementations can use it to keep their errors consistent with
// StandardCaster.
func NewCastError(value any, target string) error {
	var desc string
	switch v := value.(type) {
	case nil:
		desc = "<nil>"
	case string:
		desc = fmt.Sprintf("string %q", v)
	default:
		desc = fmt.Sprintf("%T %v", value, v)
	}
	return fmt.Errorf("cannot cast %s to %s", desc, target)
}
