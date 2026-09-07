// Package dygodata adapts internal runtime services to public dygo interfaces.
package dygodata

import (
	"context"
	"fmt"
	"strings"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/internal/recordsecret"
	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
)

// RecordData exposes metadata-backed Record access through the public SDK.
type RecordData struct {
	queryer       db.RecordQueryer
	hooks         *db.RecordHookRegistry
	mutationHooks db.RecordMutationHookPolicy
	actor         *dygo.Actor
	activityActor *dygo.Actor
	systemReason  string
	lockAction    permissions.Action
	systemMode    bool
}

var _ dygo.RecordData = RecordData{}

// NewRecordData returns dygo RecordData that uses the supplied Record hooks.
func NewRecordData(queryer db.RecordQueryer, hooks *db.RecordHookRegistry) RecordData {
	return RecordData{queryer: queryer, hooks: hooks, mutationHooks: db.RecordMutationHooksFrameworkOnly}
}

// NewRecordDataWithHookPolicy returns dygo RecordData with an explicit mutation hook policy.
func NewRecordDataWithHookPolicy(queryer db.RecordQueryer, policy db.RecordMutationHookPolicy) RecordData {
	return RecordData{queryer: queryer, mutationHooks: policy}
}

func (d RecordData) store() db.RecordStore {
	if d.hooks != nil {
		return db.NewRecordStoreWithHooks(d.queryer, d.hooks)
	}
	return db.NewRecordStoreWithHookPolicy(d.queryer, d.mutationHooks)
}

func (d RecordData) scopedStore(ctx context.Context, appName string, entity string, action permissions.Action) (db.RecordStore, error) {
	if d.systemMode && d.systemReason == "" {
		return db.RecordStore{}, fmt.Errorf("system Record access reason is required")
	}
	store := d.store()
	if d.actor == nil || d.actor.Administrator {
		return store, nil
	}
	scope, err := permissions.NewChecker(d.queryer).RecordScope(ctx, permissions.Request{Actor: *d.actor, Resource: dygo.Resource{Kind: dygo.ResourceEntity, App: appName, Name: entity}, Action: action})
	if err != nil {
		return db.RecordStore{}, err
	}
	return store.WithScope(db.RecordScope{Where: scope.Where, Args: scope.Args, FieldRead: scope.FieldRead, FieldWrite: scope.FieldWrite}), nil
}

// WithLockAction returns an internal Record view whose Lock uses a custom Entity action scope.
func (d RecordData) WithLockAction(actor dygo.Actor, action permissions.Action) RecordData {
	d.actor = &actor
	d.lockAction = action
	return d
}

// WithActionActor attributes trusted action mutations without adding a second permission check.
func (d RecordData) WithActionActor(actor dygo.Actor) RecordData {
	d.activityActor = &actor
	return d
}

func (d RecordData) context(ctx context.Context) context.Context {
	if d.activityActor != nil {
		ctx = db.WithActivityActor(ctx, d.activityActor.UserID, d.activityActor.Email, d.activityActor.Administrator)
	} else if d.actor != nil {
		ctx = db.WithActivityActor(ctx, d.actor.UserID, d.actor.Email, d.actor.Administrator)
	}
	ctx = db.WithActivitySystemReason(ctx, d.systemReason)
	return recordsecret.WithOperation(ctx)
}

// List returns a page of Records by app/entity identity.
func (d RecordData) List(ctx context.Context, appName string, entity string, params dygo.RecordListParams) (dygo.RecordListResult, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return dygo.RecordListResult{}, err
	}
	result, err := store.ListRecordsByIdentity(ctx, appName, entity, dbRecordListParams(params))
	if err != nil {
		return dygo.RecordListResult{}, err
	}
	return dygo.RecordListResult{
		Records: dygoRecords(result.Records),
		Limit:   result.Limit,
		Offset:  result.Offset,
		Count:   result.Count,
	}, nil
}

// Count returns the number of matching Records, ignoring pagination.
func (d RecordData) Count(ctx context.Context, appName string, entity string, params dygo.RecordListParams) (int64, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return 0, err
	}
	return store.CountRecordsByIdentity(ctx, appName, entity, dbRecordListParams(params))
}

// Exists reports whether at least one Record matches, ignoring pagination.
func (d RecordData) Exists(ctx context.Context, appName string, entity string, params dygo.RecordListParams) (bool, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return false, err
	}
	return store.ExistsRecordsByIdentity(ctx, appName, entity, dbRecordListParams(params))
}

// Aggregate evaluates supported aggregate expressions over matching Records.
func (d RecordData) Aggregate(ctx context.Context, appName string, entity string, params dygo.AggregateParams) ([]dygo.AggregateResult, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return nil, err
	}
	return store.AggregateRecordsByIdentity(ctx, appName, entity, params)
}

// GroupBy evaluates grouped aggregate expressions over matching Records.
func (d RecordData) GroupBy(ctx context.Context, appName string, entity string, params dygo.GroupByParams) ([]dygo.GroupByResult, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return nil, err
	}
	return store.GroupRecordsByIdentity(ctx, appName, entity, params)
}

// Lock returns matching Records in deterministic order and locks them with SELECT FOR UPDATE.
// The RecordData should be backed by a transaction and remain in that transaction while the rows are used.
func (d RecordData) Lock(ctx context.Context, appName string, entity string, params dygo.RecordListParams) (dygo.RecordListResult, error) {
	ctx = d.context(ctx)
	action := permissions.ActionRead
	if d.lockAction != "" {
		action = d.lockAction
	}
	store, err := d.scopedStore(ctx, appName, entity, action)
	if err != nil {
		return dygo.RecordListResult{}, err
	}
	result, err := store.LockRecordsByIdentity(ctx, appName, entity, dbRecordListParams(params))
	if err != nil {
		return dygo.RecordListResult{}, err
	}
	return dygo.RecordListResult{Records: dygoRecords(result.Records), Limit: result.Limit, Offset: result.Offset, Count: result.Count}, nil
}

func dbRecordListParams(params dygo.RecordListParams) db.RecordListParams {
	converted := db.RecordListParams{
		Limit:  params.Limit,
		Offset: params.Offset,
	}
	if len(params.Filters) > 0 {
		converted.Filters = make([]db.RecordFilter, len(params.Filters))
		for i, filter := range params.Filters {
			converted.Filters[i] = db.RecordFilter{Field: filter.Field, Operator: filter.Operator, Value: filter.Value}
		}
	}
	if len(params.Sort) > 0 {
		converted.Sort = make([]db.RecordSort, len(params.Sort))
		for i, sortTerm := range params.Sort {
			converted.Sort[i] = db.RecordSort{Field: sortTerm.Field, Desc: sortTerm.Desc}
		}
	}
	return converted
}

// Get returns one Record by app/entity identity and row ID.
func (d RecordData) Get(ctx context.Context, appName string, entity string, id int64) (dygo.Record, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return nil, err
	}
	record, err := store.GetRecordByIdentity(ctx, appName, entity, id)
	if err != nil {
		return nil, err
	}
	return dygo.Record(record), nil
}

// Find returns one Record matching metadata-backed fields.
func (d RecordData) Find(ctx context.Context, appName string, entity string, match dygo.RecordInput) (dygo.Record, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionRead)
	if err != nil {
		return nil, err
	}
	record, err := store.FindRecordByIdentity(ctx, appName, entity, db.RecordInput(match))
	if err != nil {
		return nil, err
	}
	return dygo.Record(record), nil
}

// Create creates one Record by app/entity identity.
func (d RecordData) Create(ctx context.Context, appName string, entity string, input dygo.RecordInput) (dygo.Record, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionCreate)
	if err != nil {
		return nil, err
	}
	record, err := store.CreateRecordByIdentity(ctx, appName, entity, db.RecordInput(input))
	if err != nil {
		return nil, err
	}
	return dygo.Record(record), nil
}

// Update updates one Record by app/entity identity and row ID.
func (d RecordData) Update(ctx context.Context, appName string, entity string, id int64, input dygo.RecordInput) (dygo.Record, error) {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionUpdate)
	if err != nil {
		return nil, err
	}
	record, err := store.UpdateRecordByIdentity(ctx, appName, entity, id, db.RecordInput(input))
	if err != nil {
		return nil, err
	}
	return dygo.Record(record), nil
}

// Delete deletes one Record by app/entity identity and row ID.
func (d RecordData) Delete(ctx context.Context, appName string, entity string, id int64) error {
	ctx = d.context(ctx)
	store, err := d.scopedStore(ctx, appName, entity, permissions.ActionDelete)
	if err != nil {
		return err
	}
	return store.DeleteRecordByIdentity(ctx, appName, entity, id)
}

// Transaction runs fn with a RecordData view bound to one PostgreSQL transaction.
func (d RecordData) Transaction(ctx context.Context, fn dygo.RecordTransactionFunc) error {
	if fn == nil {
		return fmt.Errorf("record transaction function is required")
	}
	if d.systemMode && d.systemReason == "" {
		return fmt.Errorf("system Record access reason is required")
	}
	beginner, ok := d.queryer.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("record transaction requires a database transaction beginner")
	}
	tx, err := beginner.Begin(d.context(ctx))
	if err != nil {
		return fmt.Errorf("begin record transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	transactional := RecordData{queryer: tx, hooks: d.hooks, mutationHooks: d.mutationHooks, actor: d.actor, activityActor: d.activityActor, systemReason: d.systemReason, lockAction: d.lockAction, systemMode: d.systemMode}
	if err := fn(d.context(ctx), transactional); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit record transaction: %w", err)
	}
	committed = true
	return nil
}

// AsActor returns a RecordData view that attributes mutations to actor.
func (d RecordData) AsActor(actor dygo.Actor) dygo.RecordData {
	d.actor = &actor
	d.systemReason = ""
	d.systemMode = false
	return d
}

// AsSystem returns a RecordData view that attributes mutations to the system reason.
func (d RecordData) AsSystem(reason string) dygo.RecordData {
	d.actor = nil
	d.systemReason = strings.TrimSpace(reason)
	d.systemMode = true
	return d
}

func dygoRecords(records []db.Record) []dygo.Record {
	converted := make([]dygo.Record, len(records))
	for i, record := range records {
		converted[i] = dygo.Record(record)
	}
	return converted
}
