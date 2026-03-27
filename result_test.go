package axquery

import (
	"testing"
)

// ---------------------------------------------------------------------------
// ResultType
// ---------------------------------------------------------------------------

func TestResult_Nil(t *testing.T) {
	r := NewResult(nil)
	if r.Type() != ResultNil {
		t.Errorf("Type() = %v, want ResultNil", r.Type())
	}
	if !r.IsNil() {
		t.Error("IsNil() = false, want true")
	}
	if r.Raw() != nil {
		t.Errorf("Raw() = %v, want nil", r.Raw())
	}
}

func TestResult_String(t *testing.T) {
	r := NewResult("hello")
	if r.Type() != ResultString {
		t.Errorf("Type() = %v, want ResultString", r.Type())
	}
	if r.String() != "hello" {
		t.Errorf("String() = %q, want %q", r.String(), "hello")
	}
	if r.IsNil() {
		t.Error("IsNil() = true, want false")
	}
}

func TestResult_Int(t *testing.T) {
	r := NewResult(int64(42))
	if r.Type() != ResultInt {
		t.Errorf("Type() = %v, want ResultInt", r.Type())
	}
	if r.Int() != 42 {
		t.Errorf("Int() = %d, want 42", r.Int())
	}
}

func TestResult_IntFromRegularInt(t *testing.T) {
	// goja sometimes exports int instead of int64
	r := NewResult(int(7))
	if r.Type() != ResultInt {
		t.Errorf("Type() = %v, want ResultInt", r.Type())
	}
	if r.Int() != 7 {
		t.Errorf("Int() = %d, want 7", r.Int())
	}
}

func TestResult_Float(t *testing.T) {
	r := NewResult(float64(3.14))
	if r.Type() != ResultFloat {
		t.Errorf("Type() = %v, want ResultFloat", r.Type())
	}
	if r.Float() != 3.14 {
		t.Errorf("Float() = %f, want 3.14", r.Float())
	}
}

func TestResult_FloatFromInt(t *testing.T) {
	// Int should be convertible to Float
	r := NewResult(int64(5))
	if r.Float() != 5.0 {
		t.Errorf("Float() from int = %f, want 5.0", r.Float())
	}
}

func TestResult_IntFromFloat(t *testing.T) {
	// Float should be truncated to Int
	r := NewResult(float64(7.9))
	if r.Int() != 7 {
		t.Errorf("Int() from float = %d, want 7", r.Int())
	}
}

func TestResult_Bool(t *testing.T) {
	r := NewResult(true)
	if r.Type() != ResultBool {
		t.Errorf("Type() = %v, want ResultBool", r.Type())
	}
	if r.Bool() != true {
		t.Error("Bool() = false, want true")
	}

	r2 := NewResult(false)
	if r2.Bool() != false {
		t.Error("Bool() = true, want false")
	}
}

func TestResult_Slice(t *testing.T) {
	input := []interface{}{"a", "b", "c"}
	r := NewResult(input)
	if r.Type() != ResultSlice {
		t.Errorf("Type() = %v, want ResultSlice", r.Type())
	}
	s := r.Slice()
	if len(s) != 3 {
		t.Fatalf("Slice() len = %d, want 3", len(s))
	}
	if s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("Slice() = %v, want [a b c]", s)
	}
}

func TestResult_StringSlice(t *testing.T) {
	input := []interface{}{"hello", "world"}
	r := NewResult(input)
	ss := r.StringSlice()
	if len(ss) != 2 {
		t.Fatalf("StringSlice() len = %d, want 2", len(ss))
	}
	if ss[0] != "hello" || ss[1] != "world" {
		t.Errorf("StringSlice() = %v, want [hello world]", ss)
	}
}

func TestResult_StringSlice_MixedTypes(t *testing.T) {
	// Non-string elements should be converted via fmt.Sprint
	input := []interface{}{"text", int64(42), true}
	r := NewResult(input)
	ss := r.StringSlice()
	if len(ss) != 3 {
		t.Fatalf("StringSlice() len = %d, want 3", len(ss))
	}
	if ss[0] != "text" {
		t.Errorf("ss[0] = %q, want %q", ss[0], "text")
	}
	if ss[1] != "42" {
		t.Errorf("ss[1] = %q, want %q", ss[1], "42")
	}
	if ss[2] != "true" {
		t.Errorf("ss[2] = %q, want %q", ss[2], "true")
	}
}

func TestResult_Map(t *testing.T) {
	input := map[string]interface{}{
		"count":  int64(5),
		"name":   "test",
		"active": true,
	}
	r := NewResult(input)
	if r.Type() != ResultMap {
		t.Errorf("Type() = %v, want ResultMap", r.Type())
	}
	m := r.Map()
	if m["count"] != int64(5) {
		t.Errorf("Map()[count] = %v, want 5", m["count"])
	}
	if m["name"] != "test" {
		t.Errorf("Map()[name] = %v, want test", m["name"])
	}
}

// ---------------------------------------------------------------------------
// Zero values on type mismatch
// ---------------------------------------------------------------------------

func TestResult_StringOnNonString(t *testing.T) {
	r := NewResult(int64(42))
	if r.String() != "" {
		t.Errorf("String() on int = %q, want empty", r.String())
	}
}

func TestResult_IntOnNonInt(t *testing.T) {
	r := NewResult("hello")
	if r.Int() != 0 {
		t.Errorf("Int() on string = %d, want 0", r.Int())
	}
}

func TestResult_FloatOnNonNumeric(t *testing.T) {
	r := NewResult("hello")
	if r.Float() != 0 {
		t.Errorf("Float() on string = %f, want 0", r.Float())
	}
}

func TestResult_BoolOnNonBool(t *testing.T) {
	r := NewResult("hello")
	if r.Bool() != false {
		t.Error("Bool() on string = true, want false")
	}
}

func TestResult_SliceOnNonSlice(t *testing.T) {
	r := NewResult("hello")
	if r.Slice() != nil {
		t.Errorf("Slice() on string = %v, want nil", r.Slice())
	}
}

func TestResult_StringSliceOnNonSlice(t *testing.T) {
	r := NewResult(int64(42))
	if r.StringSlice() != nil {
		t.Errorf("StringSlice() on int = %v, want nil", r.StringSlice())
	}
}

func TestResult_MapOnNonMap(t *testing.T) {
	r := NewResult("hello")
	if r.Map() != nil {
		t.Errorf("Map() on string = %v, want nil", r.Map())
	}
}

// ---------------------------------------------------------------------------
// Nil Result
// ---------------------------------------------------------------------------

func TestResult_NilResult_AllHelpersReturnZero(t *testing.T) {
	r := NewResult(nil)
	if r.String() != "" {
		t.Errorf("String() = %q", r.String())
	}
	if r.Int() != 0 {
		t.Errorf("Int() = %d", r.Int())
	}
	if r.Float() != 0 {
		t.Errorf("Float() = %f", r.Float())
	}
	if r.Bool() != false {
		t.Error("Bool() = true")
	}
	if r.Slice() != nil {
		t.Errorf("Slice() = %v", r.Slice())
	}
	if r.StringSlice() != nil {
		t.Errorf("StringSlice() = %v", r.StringSlice())
	}
	if r.Map() != nil {
		t.Errorf("Map() = %v", r.Map())
	}
}

// ---------------------------------------------------------------------------
// Raw() escape hatch
// ---------------------------------------------------------------------------

func TestResult_Raw(t *testing.T) {
	r := NewResult("hello")
	if r.Raw() != "hello" {
		t.Errorf("Raw() = %v, want hello", r.Raw())
	}
}

// ---------------------------------------------------------------------------
// ResultType String representation
// ---------------------------------------------------------------------------

func TestResultType_String(t *testing.T) {
	tests := []struct {
		rt   ResultType
		want string
	}{
		{ResultNil, "nil"},
		{ResultString, "string"},
		{ResultInt, "int"},
		{ResultFloat, "float"},
		{ResultBool, "bool"},
		{ResultSlice, "slice"},
		{ResultMap, "map"},
		{ResultType(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.rt.String(); got != tt.want {
			t.Errorf("ResultType(%d).String() = %q, want %q", tt.rt, got, tt.want)
		}
	}
}
