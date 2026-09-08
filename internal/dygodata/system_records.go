package dygodata

import (
	"context"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/pkg/dygo"
)

// System returns a trusted writer scoped to the App that registered the caller.
func (d RecordData) System(reason string) dygo.SystemRecordWriter {
	return systemRecordWriter{data: d, reason: reason}
}

type systemRecordWriter struct {
	data   RecordData
	reason string
}

func (w systemRecordWriter) Create(ctx context.Context, entity string, input dygo.RecordInput) (dygo.Record, error) {
	record, err := db.NewSystemRecordWriter(w.data.queryer).Scoped(w.data.appScope, w.reason).Create(w.data.context(ctx), entity, db.RecordInput(input))
	return dygo.Record(record), err
}

func (w systemRecordWriter) Update(ctx context.Context, entity string, id int64, input dygo.RecordInput) (dygo.Record, error) {
	record, err := db.NewSystemRecordWriter(w.data.queryer).Scoped(w.data.appScope, w.reason).Update(w.data.context(ctx), entity, id, db.RecordInput(input))
	return dygo.Record(record), err
}

func (w systemRecordWriter) Delete(ctx context.Context, entity string, id int64) error {
	return db.NewSystemRecordWriter(w.data.queryer).Scoped(w.data.appScope, w.reason).Delete(w.data.context(ctx), entity, id)
}

func (w systemRecordWriter) Upsert(ctx context.Context, entity string, match dygo.RecordInput, input dygo.RecordInput) (dygo.Record, error) {
	record, err := db.NewSystemRecordWriter(w.data.queryer).Scoped(w.data.appScope, w.reason).Upsert(w.data.context(ctx), entity, db.RecordInput(match), db.RecordInput(input))
	return dygo.Record(record), err
}
