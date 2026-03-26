package axquery

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// === WaitUntil tests ===

func TestWaitUntil_ImmediateSuccess(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	// Inject fake time/sleep
	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {} // no-op
	now := time.Now()
	nowFn = func() time.Time { return now }

	result := sel.WaitUntil(func(s *Selection) bool {
		return true // immediately satisfied
	}, time.Second)

	if result != sel {
		t.Fatal("WaitUntil should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitUntil_EventualSuccess(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {} // no-op

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		// Advance time slowly (50ms per call) so we don't time out
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 50 * time.Millisecond)
	}

	var conditionCalls int
	result := sel.WaitUntil(func(s *Selection) bool {
		conditionCalls++
		return conditionCalls >= 3 // succeed on 3rd poll
	}, time.Second)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if conditionCalls < 3 {
		t.Fatalf("expected at least 3 condition calls, got %d", conditionCalls)
	}
}

func TestWaitUntil_Timeout(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {} // no-op

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		// Jump 500ms each call to quickly exceed timeout
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	result := sel.WaitUntil(func(s *Selection) bool {
		return false // never satisfied
	}, time.Second)

	if result.Err() == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

func TestWaitUntil_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")

	result := sel.WaitUntil(func(s *Selection) bool {
		return true
	}, time.Second)

	if result != sel {
		t.Fatal("WaitUntil on error selection should return same selection")
	}
	if !errors.Is(result.Err(), errTest) {
		t.Fatalf("expected errTest, got %v", result.Err())
	}
}

func TestWaitUntil_SleepCalledWithInterval(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()

	var sleepDurations []time.Duration
	sleepFn = func(d time.Duration) {
		sleepDurations = append(sleepDurations, d)
	}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 50 * time.Millisecond)
	}

	condCalls := 0
	sel.WaitUntil(func(s *Selection) bool {
		condCalls++
		return condCalls >= 2
	}, time.Second)

	// Should have slept at least once with the default poll interval
	if len(sleepDurations) == 0 {
		t.Fatal("expected at least one sleep call")
	}
	for _, d := range sleepDurations {
		if d != DefaultPollInterval {
			t.Fatalf("expected sleep of %v, got %v", DefaultPollInterval, d)
		}
	}
}

// === WaitVisible tests ===

func TestWaitVisible_AlreadyVisible(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}
	now := time.Now()
	nowFn = func() time.Time { return now }

	result := sel.WaitVisible(time.Second)

	if result != sel {
		t.Fatal("WaitVisible should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitVisible_BecomesVisible(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: false, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()

	var pollCount int
	sleepFn = func(d time.Duration) {
		pollCount++
		if pollCount >= 2 {
			btn.visible = true // element becomes visible after 2 polls
		}
	}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 50 * time.Millisecond)
	}

	result := sel.WaitVisible(time.Second)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitVisible_Timeout(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: false, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	result := sel.WaitVisible(time.Second)

	if result.Err() == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

func TestWaitVisible_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.WaitVisible(time.Second)

	if result != sel {
		t.Fatal("WaitVisible on error selection should return same selection")
	}
}

func TestWaitVisible_EmptySelection(t *testing.T) {
	sel := newSelectionFromNodes(nil, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	result := sel.WaitVisible(time.Second)

	if result.Err() == nil {
		t.Fatal("expected timeout error for empty selection")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

// === WaitEnabled tests ===

func TestWaitEnabled_AlreadyEnabled(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}
	now := time.Now()
	nowFn = func() time.Time { return now }

	result := sel.WaitEnabled(time.Second)

	if result != sel {
		t.Fatal("WaitEnabled should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitEnabled_BecomesEnabled(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: false}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()

	var pollCount int
	sleepFn = func(d time.Duration) {
		pollCount++
		if pollCount >= 2 {
			btn.enabled = true // element becomes enabled after 2 polls
		}
	}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 50 * time.Millisecond)
	}

	result := sel.WaitEnabled(time.Second)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitEnabled_Timeout(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: false}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	result := sel.WaitEnabled(time.Second)

	if result.Err() == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

func TestWaitEnabled_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.WaitEnabled(time.Second)

	if result != sel {
		t.Fatal("WaitEnabled on error selection should return same selection")
	}
}

// === WaitGone tests ===

func TestWaitGone_AlreadyGone(t *testing.T) {
	// Empty selection = already gone
	sel := newSelectionFromNodes(nil, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}
	now := time.Now()
	nowFn = func() time.Time { return now }

	result := sel.WaitGone(time.Second)

	if result != sel {
		t.Fatal("WaitGone should return same Selection for chaining")
	}
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitGone_ElementDisappears(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()

	var pollCount int
	sleepFn = func(d time.Duration) {
		pollCount++
		if pollCount >= 2 {
			btn.role = "" // element "dies" — role becomes empty
		}
	}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 50 * time.Millisecond)
	}

	result := sel.WaitGone(time.Second)

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
}

func TestWaitGone_Timeout(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	result := sel.WaitGone(time.Second)

	if result.Err() == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

func TestWaitGone_ErrorSelection(t *testing.T) {
	sel := newSelectionError(errTest, "AXButton")
	result := sel.WaitGone(time.Second)

	// Error selection → already "gone" semantically; should return immediately
	if result != sel {
		t.Fatal("WaitGone on error selection should return same selection")
	}
}

// === Chaining tests ===

func TestWait_Chaining(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: true, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}
	now := time.Now()
	nowFn = func() time.Time { return now }

	// Chain: WaitVisible -> WaitEnabled
	result := sel.WaitVisible(time.Second).WaitEnabled(time.Second)

	if result.Err() != nil {
		t.Fatalf("unexpected error in chain: %v", result.Err())
	}
}

func TestWait_ChainStopsOnError(t *testing.T) {
	btn := &mockTraversalNode{role: "AXButton", visible: false, enabled: true}
	sel := newSelectionFromNodes([]queryNode{btn}, "AXButton")

	origSleep := sleepFn
	origNow := nowFn
	defer func() { sleepFn = origSleep; nowFn = origNow }()
	sleepFn = func(d time.Duration) {}

	var callCount int32
	now := time.Now()
	nowFn = func() time.Time {
		c := atomic.AddInt32(&callCount, 1)
		return now.Add(time.Duration(c) * 500 * time.Millisecond)
	}

	// WaitVisible will timeout, WaitEnabled should not execute
	result := sel.WaitVisible(time.Second).WaitEnabled(time.Second)

	if result.Err() == nil {
		t.Fatal("expected error from WaitVisible timeout")
	}
	if !errors.Is(result.Err(), ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", result.Err())
	}
}

// === DefaultPollInterval tests ===

func TestDefaultPollInterval_Value(t *testing.T) {
	if DefaultPollInterval != 200*time.Millisecond {
		t.Fatalf("expected DefaultPollInterval to be 200ms, got %v", DefaultPollInterval)
	}
}
