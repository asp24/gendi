package parameters

import (
	"math"
	"strings"
	"testing"
	"time"
)

type namedInt int

type castTest struct {
	name    string
	value   any
	want    any
	wantErr string // substring; empty means success expected
}

func runCastTests(t *testing.T, tests []castTest, call func(any) (any, error)) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := call(tt.value)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got value %v", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestNewCastError(t *testing.T) {
	tests := []struct {
		value  any
		target string
		want   string
	}{
		{"abc", "int", `cannot cast string "abc" to int`},
		{nil, "bool", "cannot cast <nil> to bool"},
		{42, "string", "cannot cast int 42 to string"},
		{namedInt(1), "int64", "cannot cast parameters.namedInt 1 to int64"},
	}
	for _, tt := range tests {
		if got := NewCastError(tt.value, tt.target).Error(); got != tt.want {
			t.Fatalf("NewCastError(%v, %s): got %q, want %q", tt.value, tt.target, got, tt.want)
		}
	}
}

func TestStandardCasterToInt64(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", int64(42), int64(42), ""},
		{"int", int(7), int64(7), ""},
		{"int8", int8(-8), int64(-8), ""},
		{"uint32", uint32(9), int64(9), ""},
		{"uint64 in range", uint64(10), int64(10), ""},
		{"uint64 overflow", uint64(math.MaxUint64), int64(0), "overflows"},
		{"string base10", "123", int64(123), ""},
		{"string min", "-9223372036854775808", int64(math.MinInt64), ""},
		{"string invalid", "abc", int64(0), `cannot cast string "abc" to int64`},
		{"string whitespace not trimmed", " 1", int64(0), "cannot cast"},
		{"duration as nanos", 5 * time.Second, int64(5000000000), ""},
		{"float rejected", 1.0, int64(0), "cannot cast float64 1 to int64"},
		{"bool rejected", true, int64(0), "cannot cast bool true to int64"},
		{"nil rejected", nil, int64(0), "cannot cast <nil> to int64"},
		{"named input rejected", namedInt(1), int64(0), "parameters.namedInt"},
		{"time rejected", time.Unix(0, 0).UTC(), int64(0), "time.Time"},
	}, func(v any) (any, error) { return c.ToInt64(v) })
}

func TestStandardCasterToInt8(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"max", int64(127), int8(127), ""},
		{"min", int64(-128), int8(-128), ""},
		{"overflow", int64(128), int8(0), "overflows int8"},
		{"underflow", "-129", int8(0), "overflows int8"},
	}, func(v any) (any, error) { return c.ToInt8(v) })
}

func TestStandardCasterToUint64(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", uint64(42), uint64(42), ""},
		{"int positive", int(7), uint64(7), ""},
		{"int negative", int(-1), uint64(0), "negative"},
		{"string", "18446744073709551615", uint64(math.MaxUint64), ""},
		{"string negative", "-1", uint64(0), "cannot cast"},
		{"duration positive", time.Second, uint64(1000000000), ""},
		{"duration negative", -time.Second, uint64(0), "negative"},
		{"float rejected", 1.5, uint64(0), "cannot cast"},
		{"bool rejected", false, uint64(0), "cannot cast"},
	}, func(v any) (any, error) { return c.ToUint64(v) })
}

func TestStandardCasterToUint8(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"max", uint64(255), uint8(255), ""},
		{"overflow", uint64(256), uint8(0), "overflows uint8"},
	}, func(v any) (any, error) { return c.ToUint8(v) })
}

func TestStandardCasterToFloat64(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", 2.5, 2.5, ""},
		{"float32 widen", float32(1.5), 1.5, ""},
		{"int exact", int64(1 << 53), float64(1 << 53), ""},
		{"int inexact", int64(1<<53 + 1), float64(0), "not exactly representable"},
		{"uint exact", uint64(16), float64(16), ""},
		{"string", "1e3", 1000.0, ""},
		{"string NaN rejected", "NaN", float64(0), "NaN and infinities"},
		{"string Inf rejected", "+Inf", float64(0), "NaN and infinities"},
		{"raw NaN rejected", math.NaN(), float64(0), "NaN and infinities"},
		{"bool rejected", true, float64(0), "cannot cast"},
		{"duration rejected", time.Second, float64(0), "cannot cast"},
	}, func(v any) (any, error) { return c.ToFloat64(v) })
}

func TestStandardCasterToFloat32(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", float32(2.5), float32(2.5), ""},
		{"float64 exact", 3.5, float32(3.5), ""},
		{"float64 inexact", 0.1, float32(0), "not exactly representable"},
		{"int exact", int64(1024), float32(1024), ""},
		{"int inexact", int64(16777217), float32(0), "not exactly representable"},
		{"string parses at float32 precision", "0.1", float32(0.1), ""},
	}, func(v any) (any, error) { return c.ToFloat32(v) })
}

func TestStandardCasterToString(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", "x", "x", ""},
		{"int", int(-42), "-42", ""},
		{"uint64", uint64(7), "7", ""},
		{"float64 canonical", 1.5, "1.5", ""},
		{"float64 roundtrip", 0.1, "0.1", ""},
		{"float32", float32(2.5), "2.5", ""},
		{"bool rejected", true, "", "cannot cast bool true to string"},
		{"duration rejected", time.Second, "", "cannot cast"},
		{"time rejected", time.Unix(0, 0).UTC(), "", "cannot cast"},
	}, func(v any) (any, error) { return c.ToString(v) })
}

func TestStandardCasterToBool(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", true, true, ""},
		{"string true", "true", true, ""},
		{"string 1", "1", true, ""},
		{"string invalid", "yes", false, "cannot cast"},
		{"int rejected", 1, false, "cannot cast int 1 to bool"},
	}, func(v any) (any, error) { return c.ToBool(v) })
}

func TestStandardCasterToDuration(t *testing.T) {
	c := StandardCaster{}
	runCastTests(t, []castTest{
		{"identity", 2 * time.Second, 2 * time.Second, ""},
		{"string", "1h30m", 90 * time.Minute, ""},
		{"string invalid", "5x", time.Duration(0), "cannot cast"},
		{"int as nanos", int64(1500), 1500 * time.Nanosecond, ""},
		{"uint as nanos", uint32(7), 7 * time.Nanosecond, ""},
		{"float rejected", 1.5, time.Duration(0), "cannot cast"},
		{"bool rejected", true, time.Duration(0), "cannot cast"},
		{"time rejected", time.Unix(0, 0).UTC(), time.Duration(0), "cannot cast"},
	}, func(v any) (any, error) { return c.ToDuration(v) })
}

func TestStandardCasterToTime(t *testing.T) {
	c := StandardCaster{}
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	runCastTests(t, []castTest{
		{"identity", when, when, ""},
		{"string RFC3339", "2026-01-02T03:04:05Z", when, ""},
		{"string invalid", "2026-01-02", time.Time{}, "cannot cast"},
		{"int rejected", int64(5), time.Time{}, "cannot cast"},
	}, func(v any) (any, error) { return c.ToTime(v) })
}
