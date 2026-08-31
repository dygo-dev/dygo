package dygo

import "context"

// TimelineEvent is a human-readable event attached to a Record timeline.
// Kind and Operation use the values defined by Core Activity metadata.
type TimelineEvent struct {
	Kind      string
	Operation string
	Status    string
	Title     string
	Message   string
	Changes   any
	Snapshot  any
	Details   any
}

// TimelineData appends human-facing Record history through Core Activity.
type TimelineData interface {
	AddComment(context.Context, string, string, int64, string) error
	AddEvent(context.Context, string, string, int64, TimelineEvent) error
}
