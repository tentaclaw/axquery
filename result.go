package axquery

import "fmt"

// ---------------------------------------------------------------------------
// ResultType — enum for JS output types
// ---------------------------------------------------------------------------

// ResultType represents the type of a script execution result.
type ResultType int

const (
	// ResultNil indicates a nil/null/undefined result.
	ResultNil ResultType = iota
	// ResultString indicates a string result.
	ResultString
	// ResultInt indicates an integer result.
	ResultInt
	// ResultFloat indicates a floating-point result.
	ResultFloat
	// ResultBool indicates a boolean result.
	ResultBool
	// ResultSlice indicates an array/slice result.
	ResultSlice
	// ResultMap indicates an object/map result.
	ResultMap
)

// String returns the human-readable name of the result type.
func (t ResultType) String() string {
	switch t {
	case ResultNil:
		return "nil"
	case ResultString:
		return "string"
	case ResultInt:
		return "int"
	case ResultFloat:
		return "float"
	case ResultBool:
		return "bool"
	case ResultSlice:
		return "slice"
	case ResultMap:
		return "map"
	default:
		return "unknown"
	}
}

// ---------------------------------------------------------------------------
// Result — typed wrapper for script output
// ---------------------------------------------------------------------------

// Result wraps a script execution output value and provides type-safe
// accessor methods. If the underlying value does not match the requested
// type, the accessor returns the zero value for that type.
type Result struct {
	value any
}

// NewResult creates a Result wrapping the given value.
func NewResult(v any) *Result {
	return &Result{value: v}
}

// Type returns the ResultType of the underlying value.
func (r *Result) Type() ResultType {
	if r.value == nil {
		return ResultNil
	}
	switch r.value.(type) {
	case string:
		return ResultString
	case int64, int:
		return ResultInt
	case float64:
		return ResultFloat
	case bool:
		return ResultBool
	case []interface{}:
		return ResultSlice
	case map[string]interface{}:
		return ResultMap
	default:
		return ResultNil
	}
}

// IsNil returns true if the underlying value is nil.
func (r *Result) IsNil() bool {
	return r.value == nil
}

// Raw returns the underlying value without conversion.
func (r *Result) Raw() any {
	return r.value
}

// String returns the string value, or "" if the value is not a string.
func (r *Result) String() string {
	if s, ok := r.value.(string); ok {
		return s
	}
	return ""
}

// Int returns the int64 value. If the value is a float64, it is truncated.
// Returns 0 if the value is not numeric.
func (r *Result) Int() int64 {
	switch v := r.value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// Float returns the float64 value. If the value is an int64, it is converted.
// Returns 0 if the value is not numeric.
func (r *Result) Float() float64 {
	switch v := r.value.(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

// Bool returns the bool value, or false if the value is not a bool.
func (r *Result) Bool() bool {
	if b, ok := r.value.(bool); ok {
		return b
	}
	return false
}

// Slice returns the []any value, or nil if the value is not a slice.
func (r *Result) Slice() []any {
	if s, ok := r.value.([]interface{}); ok {
		return s
	}
	return nil
}

// StringSlice converts the slice value to []string. Each element is converted
// to string via fmt.Sprint. Returns nil if the value is not a slice.
func (r *Result) StringSlice() []string {
	s, ok := r.value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, len(s))
	for i, v := range s {
		if str, ok := v.(string); ok {
			result[i] = str
		} else {
			result[i] = fmt.Sprint(v)
		}
	}
	return result
}

// Map returns the map[string]any value, or nil if the value is not a map.
func (r *Result) Map() map[string]any {
	if m, ok := r.value.(map[string]interface{}); ok {
		return m
	}
	return nil
}
