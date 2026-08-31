package dygo

import "context"

// AggregateFunction identifies a supported Record aggregate.
type AggregateFunction string

const (
	AggregateCount AggregateFunction = "count"
	AggregateSum   AggregateFunction = "sum"
	AggregateMin   AggregateFunction = "min"
	AggregateMax   AggregateFunction = "max"
)

// AggregateSpec describes one aggregate expression.
//
// Count may omit Field to count all matching Records. Sum, min, and max
// require a metadata-backed scalar Field.
type AggregateSpec struct {
	Function AggregateFunction
	Field    string
	Alias    string
}

// AggregateResult is one value returned by Aggregate.
type AggregateResult struct {
	Function AggregateFunction
	Field    string
	Alias    string
	Value    any
}

// AggregateParams controls aggregate filters and expressions.
type AggregateParams struct {
	Filters    []RecordFilter
	Aggregates []AggregateSpec
}

// GroupByParams controls grouped aggregates.
type GroupByParams struct {
	Filters    []RecordFilter
	GroupBy    []string
	Aggregates []AggregateSpec
	Limit      int
	Offset     int
}

// GroupByResult is one grouped aggregate row.
type GroupByResult struct {
	Group      map[string]any
	Aggregates map[string]any
}

// RecordTransactionFunc runs app code inside one Record transaction.
type RecordTransactionFunc func(context.Context, RecordData) error
