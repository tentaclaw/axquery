package axquery

import (
	"testing"
	"time"
)

func TestDefaultQueryOptions(t *testing.T) {
	opts := defaultQueryOptions()
	if opts.Timeout != 0 {
		t.Fatalf("default timeout should be 0 (no timeout), got %v", opts.Timeout)
	}
	if opts.MaxDepth != 0 {
		t.Fatalf("default max depth should be 0 (unlimited), got %d", opts.MaxDepth)
	}
	if opts.MaxResults != 0 {
		t.Fatalf("default max results should be 0 (unlimited), got %d", opts.MaxResults)
	}
	if opts.Strategy != StrategyBFS {
		t.Fatalf("default strategy should be BFS, got %d", opts.Strategy)
	}
}

func TestWithTimeout(t *testing.T) {
	opts := defaultQueryOptions()
	WithTimeout(5 * time.Second)(&opts)
	if opts.Timeout != 5*time.Second {
		t.Fatalf("expected 5s timeout, got %v", opts.Timeout)
	}
}

func TestWithMaxDepth(t *testing.T) {
	opts := defaultQueryOptions()
	WithMaxDepth(10)(&opts)
	if opts.MaxDepth != 10 {
		t.Fatalf("expected max depth 10, got %d", opts.MaxDepth)
	}
}

func TestWithMaxResults(t *testing.T) {
	opts := defaultQueryOptions()
	WithMaxResults(5)(&opts)
	if opts.MaxResults != 5 {
		t.Fatalf("expected max results 5, got %d", opts.MaxResults)
	}
}

func TestWithStrategy(t *testing.T) {
	opts := defaultQueryOptions()
	WithStrategy(StrategyDFS)(&opts)
	if opts.Strategy != StrategyDFS {
		t.Fatalf("expected DFS strategy, got %d", opts.Strategy)
	}
}

func TestApplyOptions(t *testing.T) {
	opts := applyOptions(
		WithTimeout(3*time.Second),
		WithMaxDepth(5),
		WithMaxResults(10),
		WithStrategy(StrategyDFS),
	)

	if opts.Timeout != 3*time.Second {
		t.Fatalf("expected 3s timeout, got %v", opts.Timeout)
	}
	if opts.MaxDepth != 5 {
		t.Fatalf("expected max depth 5, got %d", opts.MaxDepth)
	}
	if opts.MaxResults != 10 {
		t.Fatalf("expected max results 10, got %d", opts.MaxResults)
	}
	if opts.Strategy != StrategyDFS {
		t.Fatalf("expected DFS strategy, got %d", opts.Strategy)
	}
}

func TestApplyOptions_Empty(t *testing.T) {
	opts := applyOptions()
	def := defaultQueryOptions()
	if opts != def {
		t.Fatal("applyOptions with no options should return defaults")
	}
}

func TestSearchStrategy_Constants(t *testing.T) {
	if StrategyBFS != 0 {
		t.Fatal("StrategyBFS should be 0")
	}
	if StrategyDFS != 1 {
		t.Fatal("StrategyDFS should be 1")
	}
}
