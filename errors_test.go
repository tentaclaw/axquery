package axquery

import (
	"errors"
	"testing"
)

// --- NotFoundError ---

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{Selector: "AXButton"}
	want := `axquery: no elements matching "AXButton"`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestNotFoundError_Is(t *testing.T) {
	err := &NotFoundError{Selector: "AXButton"}
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("NotFoundError should match ErrNotFound sentinel")
	}
	if errors.Is(err, ErrTimeout) {
		t.Fatal("NotFoundError should NOT match ErrTimeout")
	}
}

func TestNotFoundError_Unwrap(t *testing.T) {
	err := &NotFoundError{Selector: "AXButton"}
	if err.Unwrap() != ErrNotFound {
		t.Fatal("Unwrap should return ErrNotFound")
	}
}

// --- TimeoutError ---

func TestTimeoutError_Error(t *testing.T) {
	err := &TimeoutError{Selector: "AXButton", Duration: "5s"}
	want := `axquery: timeout after 5s waiting for "AXButton"`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestTimeoutError_Is(t *testing.T) {
	err := &TimeoutError{Selector: "AXButton", Duration: "5s"}
	if !errors.Is(err, ErrTimeout) {
		t.Fatal("TimeoutError should match ErrTimeout sentinel")
	}
}

func TestTimeoutError_Unwrap(t *testing.T) {
	err := &TimeoutError{Selector: "AXButton", Duration: "5s"}
	if err.Unwrap() != ErrTimeout {
		t.Fatal("Unwrap should return ErrTimeout")
	}
}

// --- AmbiguousError ---

func TestAmbiguousError_Error(t *testing.T) {
	err := &AmbiguousError{Selector: "AXButton", Count: 5}
	want := `axquery: selector "AXButton" matched 5 elements, expected 1`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestAmbiguousError_Is(t *testing.T) {
	err := &AmbiguousError{Selector: "AXButton", Count: 5}
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatal("AmbiguousError should match ErrAmbiguous sentinel")
	}
}

func TestAmbiguousError_Unwrap(t *testing.T) {
	err := &AmbiguousError{Selector: "AXButton", Count: 5}
	if err.Unwrap() != ErrAmbiguous {
		t.Fatal("Unwrap should return ErrAmbiguous")
	}
}

// --- InvalidSelectorError ---

func TestInvalidSelectorError_Error(t *testing.T) {
	err := &InvalidSelectorError{Selector: "[broken", Reason: "unclosed bracket"}
	want := `axquery: invalid selector "[broken": unclosed bracket`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestInvalidSelectorError_Is(t *testing.T) {
	err := &InvalidSelectorError{Selector: "[broken", Reason: "unclosed bracket"}
	if !errors.Is(err, ErrInvalidSelector) {
		t.Fatal("InvalidSelectorError should match ErrInvalidSelector sentinel")
	}
}

func TestInvalidSelectorError_Unwrap(t *testing.T) {
	err := &InvalidSelectorError{Selector: "[broken", Reason: "unclosed bracket"}
	if err.Unwrap() != ErrInvalidSelector {
		t.Fatal("Unwrap should return ErrInvalidSelector")
	}
}

// --- NotActionableError ---

func TestNotActionableError_Error(t *testing.T) {
	err := &NotActionableError{Action: "click", Reason: "element not enabled"}
	want := `axquery: cannot perform "click": element not enabled`
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestNotActionableError_Is(t *testing.T) {
	err := &NotActionableError{Action: "click", Reason: "element not enabled"}
	if !errors.Is(err, ErrNotActionable) {
		t.Fatal("NotActionableError should match ErrNotActionable sentinel")
	}
}

func TestNotActionableError_Unwrap(t *testing.T) {
	err := &NotActionableError{Action: "click", Reason: "element not enabled"}
	if err.Unwrap() != ErrNotActionable {
		t.Fatal("Unwrap should return ErrNotActionable")
	}
}

// --- Sentinel errors are distinct ---

func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{ErrNotFound, ErrTimeout, ErrAmbiguous, ErrInvalidSelector, ErrNotActionable}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %d and %d should be distinct", i, j)
			}
		}
	}
}

func TestSentinelErrors_Messages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrNotFound, "not found"},
		{ErrTimeout, "timeout"},
		{ErrAmbiguous, "ambiguous"},
		{ErrInvalidSelector, "invalid selector"},
		{ErrNotActionable, "not actionable"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("sentinel %v: got %q, want %q", c.err, c.err.Error(), c.want)
		}
	}
}
