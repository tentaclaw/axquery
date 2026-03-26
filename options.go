package axquery

import "time"

// SearchStrategy determines how the AX tree is traversed.
type SearchStrategy int

const (
	// StrategyBFS uses breadth-first search (default).
	StrategyBFS SearchStrategy = iota
	// StrategyDFS uses depth-first search.
	StrategyDFS
)

// QueryOptions holds configuration for a query operation.
type QueryOptions struct {
	Timeout    time.Duration  // 0 means no timeout
	MaxDepth   int            // 0 means unlimited
	MaxResults int            // 0 means unlimited
	Strategy   SearchStrategy // default BFS
}

// QueryOption is a functional option for configuring queries.
type QueryOption func(*QueryOptions)

// WithTimeout sets the maximum duration for a query operation.
func WithTimeout(d time.Duration) QueryOption {
	return func(o *QueryOptions) {
		o.Timeout = d
	}
}

// WithMaxDepth sets the maximum tree depth to search.
func WithMaxDepth(n int) QueryOption {
	return func(o *QueryOptions) {
		o.MaxDepth = n
	}
}

// WithMaxResults sets the maximum number of matching elements to return.
func WithMaxResults(n int) QueryOption {
	return func(o *QueryOptions) {
		o.MaxResults = n
	}
}

// WithStrategy sets the tree traversal strategy.
func WithStrategy(s SearchStrategy) QueryOption {
	return func(o *QueryOptions) {
		o.Strategy = s
	}
}

// defaultQueryOptions returns the default query configuration.
func defaultQueryOptions() QueryOptions {
	return QueryOptions{
		Strategy: StrategyBFS,
	}
}

// applyOptions creates a QueryOptions by applying functional options to defaults.
func applyOptions(opts ...QueryOption) QueryOptions {
	o := defaultQueryOptions()
	for _, fn := range opts {
		fn(&o)
	}
	return o
}
