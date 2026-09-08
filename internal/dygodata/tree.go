package dygodata

import (
	"context"
	"encoding/json"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
)

type treeData struct {
	records     RecordData
	app, entity string
}

func (d RecordData) Tree(app, entity string) dygo.TreeData { return treeData{d, app, entity} }

func (t treeData) read(ctx context.Context, operation string, anchor int64, params dygo.RecordListParams, exclude int64) (dygo.TreeResult, error) {
	ctx = t.records.context(ctx)
	store, err := t.records.scopedStore(ctx, t.app, t.entity, permissions.ActionRead)
	if err != nil {
		return dygo.TreeResult{}, err
	}
	return store.TreeRecords(ctx, t.app, t.entity, operation, anchor, dbRecordListParams(params), exclude)
}
func (t treeData) Roots(ctx context.Context, p dygo.RecordListParams) (dygo.TreeResult, error) {
	return t.read(ctx, "roots", 0, p, 0)
}
func (t treeData) Children(ctx context.Context, id int64, p dygo.RecordListParams) (dygo.TreeResult, error) {
	return t.read(ctx, "children", id, p, 0)
}
func (t treeData) Descendants(ctx context.Context, id int64, p dygo.RecordListParams) (dygo.TreeResult, error) {
	return t.read(ctx, "descendants", id, p, 0)
}
func (t treeData) Ancestors(ctx context.Context, id int64) (dygo.TreeResult, error) {
	return t.read(ctx, "ancestors", id, dygo.RecordListParams{}, 0)
}
func (t treeData) Path(ctx context.Context, id int64) (dygo.TreeResult, error) {
	return t.read(ctx, "path", id, dygo.RecordListParams{}, 0)
}
func (t treeData) Search(ctx context.Context, p dygo.TreeSearchParams) (dygo.TreeResult, error) {
	return t.read(ctx, "search", 0, p.RecordListParams, p.ExcludeSubtree)
}
func (t treeData) Move(ctx context.Context, id int64, parent *string) (dygo.Record, error) {
	meta, err := db.NewMetadataReader(t.records.queryer).GetEntityMetaByIdentity(ctx, t.app, t.entity)
	if err != nil {
		return nil, err
	}
	if meta.Tree == nil {
		return nil, db.RecordError{Code: db.RecordErrorInvalidRequest, Message: "Entity does not support trees"}
	}
	raw, err := json.Marshal(parent)
	if err != nil {
		return nil, err
	}
	return t.records.Update(ctx, t.app, t.entity, id, dygo.RecordInput{meta.Tree.ParentField: raw})
}
