package dygo

import "context"

// SystemRecordWriter mutates system Entities owned by the registering App.
// It retains the caller's transaction and actor attribution, requires a reason,
// and runs framework Activity hooks without re-entering Business App hooks.
// Pass this writer to App services that need to maintain system Records.
type SystemRecordWriter interface {
	Create(ctx context.Context, entity string, input RecordInput) (Record, error)
	Update(ctx context.Context, entity string, id int64, input RecordInput) (Record, error)
	Delete(ctx context.Context, entity string, id int64) error
	// Upsert requires match to identify a metadata-defined unique key.
	Upsert(ctx context.Context, entity string, match RecordInput, input RecordInput) (Record, error)
}
