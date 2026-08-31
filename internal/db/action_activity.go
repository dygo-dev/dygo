package db

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/hapyco/dygo/internal/corevalues"
)

// WriteActionActivity appends one successful Entity action timeline entry.
func WriteActionActivity(ctx context.Context, queryer RecordQueryer, meta MetadataEntityMeta, recordID int64, action string, label string) error {
	details, err := json.Marshal(map[string]any{"action": action})
	if err != nil {
		return err
	}
	input := RecordInput{
		"kind":      json.RawMessage(strconv.Quote(corevalues.ActivityKindAction)),
		"operation": json.RawMessage(strconv.Quote(corevalues.ActivityOperationAction)),
		"status":    json.RawMessage(strconv.Quote("success")),
		"entity":    json.RawMessage(strconv.Quote(meta.Name)),
		"record-id": json.RawMessage(strconv.FormatInt(recordID, 10)),
		"title":     json.RawMessage(strconv.Quote(label)),
		"details":   details,
	}
	if actor, ok := ActivityActorNameFromContext(ctx); ok {
		input["actor"] = json.RawMessage(strconv.Quote(actor))
	}
	return NewSystemRecordWriter(queryer).InsertByIdentity(ctx, "core", "activity", input, SystemMutationSilent)
}
