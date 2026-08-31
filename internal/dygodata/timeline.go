package dygodata

import (
	"context"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/pkg/dygo"
)

// TimelineData persists public SDK timeline entries through Core Activity.
type TimelineData struct {
	reader db.ActivityReader
	actor  *dygo.Actor
}

var _ dygo.TimelineData = TimelineData{}

// NewTimelineData returns TimelineData backed by metadata-driven Activity storage.
func NewTimelineData(queryer db.ActivityQueryer) TimelineData {
	return TimelineData{reader: db.NewActivityReader(queryer)}
}

// NewTimelineDataAsActor returns TimelineData that attributes entries to actor.
func NewTimelineDataAsActor(queryer db.ActivityQueryer, actor dygo.Actor) TimelineData {
	return TimelineData{reader: db.NewActivityReader(queryer), actor: &actor}
}

func (d TimelineData) context(ctx context.Context) context.Context {
	if d.actor == nil {
		return ctx
	}
	return db.WithActivityActor(ctx, d.actor.UserID, d.actor.Email, d.actor.Administrator)
}

// AddComment appends one comment to a Record timeline.
func (d TimelineData) AddComment(ctx context.Context, appName string, entity string, recordID int64, message string) error {
	return d.reader.AddCommentByIdentity(d.context(ctx), appName, entity, recordID, message)
}

// AddEvent appends one event to a Record timeline.
func (d TimelineData) AddEvent(ctx context.Context, appName string, entity string, recordID int64, event dygo.TimelineEvent) error {
	return d.reader.AddEventByIdentity(d.context(ctx), appName, entity, recordID, db.TimelineEvent{
		Kind: event.Kind, Operation: event.Operation, Status: event.Status, Title: event.Title,
		Message: event.Message, Changes: event.Changes, Snapshot: event.Snapshot, Details: event.Details,
	})
}
