package dygodata

import (
	"context"
	"errors"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
)

// DecryptSecret requires explicit system access; ordinary trusted SDK reads cannot reveal secrets.
func (d RecordData) DecryptSecret(ctx context.Context, appName, entity string, id int64, field string) (string, error) {
	ctx = d.context(ctx)
	var value string
	var err error
	if !d.systemMode || d.systemReason == "" {
		err = db.RecordError{Code: db.RecordErrorPermissionDenied, Message: "secret decryption requires explicit system access"}
	} else {
		store := d.store()
		if d.privateMode {
			store, err = d.scopedStore(ctx, appName, entity, permissions.ActionRead)
		}
		if err == nil {
			value, err = store.DecryptSecret(ctx, appName, entity, id, field)
		}
	}
	outcome := "success"
	if err != nil {
		outcome = "denied-or-failed"
	}
	actor := ""
	if a, ok := db.ActivityActorFromContext(ctx); ok {
		actor = a.Email
	}
	// Keep this sink framework-owned. Public LogWriter context values are app
	// observability hooks and must not be able to suppress the security audit.
	logErr := NewLogData(d.queryer).WriteLog(ctx, dygo.LogEntry{Type: dygo.TypeInfo, Source: dygo.SourceSDK, Title: "Record secret decryption", App: appName, Actor: actor, ReferenceEntity: appName + "." + entity, ReferenceRecordID: id, Metadata: map[string]any{"field": field, "reason": d.systemReason, "outcome": outcome}})
	if logErr != nil {
		if err != nil {
			return "", err
		}
		return "", errors.New("record secret access audit failed")
	}
	return value, err
}
func (d RecordData) SecretStatus(ctx context.Context, appName, entity string, id int64) (dygo.SecretStatus, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return dygo.SecretStatus{}, err
	}
	return store.SecretStatusByIdentity(ctx, appName, entity, id)
}
