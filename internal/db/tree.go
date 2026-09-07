package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
)

// TreeRecords shares one read scope and transaction across the page, paths, and
// child indicators. Recursive work is batched, never one query per node.
func (s RecordStore) TreeRecords(ctx context.Context, app, entity, operation string, anchor int64, params RecordListParams, exclude int64) (dygo.TreeResult, error) {
	if _, transactional := s.queryer.(pgx.Tx); !transactional {
		var result dygo.TreeResult
		_, err := s.withRecordMutation(ctx, func(txStore RecordStore) (Record, error) {
			if _, ok := txStore.queryer.(pgx.Tx); !ok {
				return nil, recordError(RecordErrorInternal, "tree access requires a transaction", nil, nil)
			}
			var err error
			result, err = txStore.TreeRecords(ctx, app, entity, operation, anchor, params, exclude)
			return nil, err
		})
		return result, err
	}
	layout, err := s.recordLayoutByIdentity(ctx, app, entity)
	if err != nil {
		return dygo.TreeResult{}, err
	}
	if layout.Tree == nil {
		return dygo.TreeResult{}, recordError(RecordErrorInvalidRequest, "Entity does not support trees", nil, nil)
	}
	// Keep parent values stable until all permission-safe path data is assembled.
	if _, err := s.queryer.Exec(ctx, `SELECT pg_advisory_xact_lock_shared(hashtextextended($1, 0))`, "dygo.tree:"+layout.Table); err != nil {
		return dygo.TreeResult{}, err
	}
	params, err = normalizeRecordListParams(params)
	if err != nil {
		return dygo.TreeResult{}, err
	}
	if len(params.Sort) == 0 {
		params.Sort = []RecordSort{{Field: "name"}}
	}
	hasName := false
	for _, term := range params.Sort {
		if term.Field == "name" {
			hasName = true
		}
	}
	if !hasName {
		params.Sort = append(append([]RecordSort{}, params.Sort...), RecordSort{Field: "name"})
	}
	result := dygo.TreeResult{Nodes: []dygo.TreeNode{}, Context: []dygo.Record{}, Limit: params.Limit, Offset: params.Offset}
	if operation != "roots" && operation != "search" {
		if anchor <= 0 {
			return result, recordError(RecordErrorInvalidRequest, "tree anchor is required", nil, nil)
		}
		if _, err := s.getRecordWithLayout(ctx, layout, anchor); err != nil {
			return result, err
		}
	}
	if operation == "ancestors" || operation == "path" {
		paths, records, err := s.treePaths(ctx, layout, []int64{anchor}, nil)
		if err != nil {
			return result, err
		}
		path, ok := readableTreePath(paths[anchor], treeRecordMap(records), layout.Tree.ParentField)
		if !ok {
			return result, treeAccessError()
		}
		if operation == "ancestors" {
			path = path[:len(path)-1]
		}
		children, err := s.treeChildIndicators(ctx, layout, treeRecordIDs(path))
		if err != nil {
			return result, err
		}
		for _, record := range path {
			result.Nodes = append(result.Nodes, treeNode(layout, record, children, false))
		}
		result.Count = len(result.Nodes)
		result.Limit = result.Count
		result.Offset = 0
		return result, nil
	}
	queryStore := s
	column := recordSourceColumn(layout.FieldByName[layout.Tree.ParentField].Column)
	switch operation {
	case "roots":
		queryStore = s.withTreePredicate(column+" IS NULL", nil)
	case "children":
		queryStore = s.withTreePredicate(column+" = $1", []any{anchor})
	case "descendants":
		queryStore = s.withTreePredicate(recordSourceColumn("id")+" IN ("+treeSubtreeSQL(layout)+") AND "+recordSourceColumn("id")+" <> $1", []any{anchor})
	case "search":
	default:
		return result, recordError(RecordErrorInvalidRequest, "unknown tree operation", nil, nil)
	}
	if exclude != 0 {
		if operation != "search" {
			return result, recordError(RecordErrorInvalidRequest, "subtree exclusion requires tree search", nil, nil)
		}
		if _, err := s.getRecordWithLayout(ctx, layout, exclude); err != nil {
			return result, err
		}
		queryStore = queryStore.withTreePredicate(recordSourceColumn("id")+" NOT IN ("+treeSubtreeSQL(layout)+")", []any{exclude})
	}
	if operation != "search" && queryStore.scope != nil {
		if predicate := queryStore.scope.FieldRead[layout.Tree.ParentField]; predicate != "" {
			scope := *queryStore.scope
			scope.Where += " AND (" + predicate + ")"
			queryStore = queryStore.WithScope(scope)
		}
	}
	page, err := queryStore.listRecords(ctx, layout, entity, params)
	if err != nil {
		return result, err
	}
	ids := treeRecordIDs(page.Records)
	children, err := s.treeChildIndicators(ctx, layout, ids)
	if err != nil {
		return result, err
	}
	paths := map[int64][]treePathStep{}
	ordered := page.Records
	switch operation {
	case "roots":
		for _, id := range ids {
			paths[id] = []treePathStep{{id: id}}
		}
	case "children":
		if len(ids) > 0 {
			var anchors map[int64][]treePathStep
			anchors, ordered, err = s.treePaths(ctx, layout, []int64{anchor}, nil)
			ordered = append(ordered, page.Records...)
			for _, id := range ids {
				paths[id] = append(append([]treePathStep{}, anchors[anchor]...), treePathStep{id: id, parent: &anchor})
			}
		}
	default:
		paths, ordered, err = s.treePaths(ctx, layout, ids, params.Sort)
	}
	if err != nil {
		return result, err
	}
	byID := treeRecordMap(ordered)
	displayIDs := map[int64]bool{}
	for _, record := range page.Records {
		id, _ := activityRecordID(record)
		path, readable := readableTreePath(paths[id], byID, layout.Tree.ParentField)
		result.Nodes = append(result.Nodes, treeNode(layout, record, children, !readable))
		if operation == "search" && readable {
			for _, ancestor := range path {
				ancestorID, _ := activityRecordID(ancestor)
				displayIDs[ancestorID] = true
			}
		}
	}
	if operation == "search" {
		// Include matches so an insertion-ordered frontend map retains sibling sort.
		for _, record := range ordered {
			id, _ := activityRecordID(record)
			if displayIDs[id] {
				result.Context = append(result.Context, dygo.Record(record))
			}
		}
	}
	result.Count = len(result.Nodes)
	return result, nil
}

func treeSubtreeSQL(layout recordLayout) string {
	return fmt.Sprintf(`WITH RECURSIVE subtree AS (
 SELECT id FROM %[1]s WHERE id = $1
 UNION SELECT n.id FROM %[1]s n JOIN subtree p ON n.%[2]s = p.id
) SELECT id FROM subtree`, quoteIdent(layout.Table), quoteIdent(layout.FieldByName[layout.Tree.ParentField].Column))
}

func (s RecordStore) withTreePredicate(predicate string, args []any) RecordStore {
	scope := RecordScope{Where: "TRUE"}
	if s.scope != nil {
		scope = *s.scope
	}
	if scope.Where == "" {
		scope.Where = "TRUE"
	}
	scope.Where = "(" + scope.Where + ") AND (" + shiftPlaceholders(predicate, len(scope.Args)) + ")"
	scope.Args = append(append([]any{}, scope.Args...), args...)
	return s.WithScope(scope)
}

func treeRecordIDs(records []Record) []int64 {
	ids := make([]int64, len(records))
	for i, record := range records {
		ids[i], _ = activityRecordID(record)
	}
	return ids
}

func treeRecordMap(records []Record) map[int64]Record {
	result := make(map[int64]Record, len(records))
	for _, record := range records {
		id, _ := activityRecordID(record)
		result[id] = record
	}
	return result
}

func treeNode(layout recordLayout, record Record, children map[int64]bool, unavailable bool) dygo.TreeNode {
	id, _ := activityRecordID(record)
	node := dygo.TreeNode{Record: dygo.Record(record), Matched: true, HasChildren: children[id], PathUnavailable: unavailable}
	if unavailable {
		delete(node.Record, layout.Tree.ParentField)
	} else {
		node.Parent, _ = record[layout.Tree.ParentField].(string)
	}
	return node
}

func (s RecordStore) treeChildIndicators(ctx context.Context, layout recordLayout, ids []int64) (map[int64]bool, error) {
	result := map[int64]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	column := recordSourceColumn(layout.FieldByName[layout.Tree.ParentField].Column)
	where, args := s.scopedReadWhere(column+" = ANY($1::bigint[])", []any{ids}, []string{layout.Tree.ParentField})
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s AS %s WHERE %s", column, quoteIdent(layout.Table), quoteIdent(recordSelectSourceAlias), where)
	rows, err := s.queryer.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result[id] = true
	}
	return result, rows.Err()
}

type treePathStep struct {
	id     int64
	parent *int64
	depth  int
}

// Return complete paths or an explicit size/cycle error. The batch and depth
// bounds protect pathological metadata; no path is silently truncated.
func (s RecordStore) treePaths(ctx context.Context, layout recordLayout, ids []int64, order []RecordSort) (map[int64][]treePathStep, []Record, error) {
	paths := map[int64][]treePathStep{}
	if len(ids) == 0 {
		return paths, []Record{}, nil
	}
	query := fmt.Sprintf(`WITH RECURSIVE path AS (
 SELECT id AS origin, id, %[2]s AS parent_id, ARRAY[id] AS trail, false AS cycle FROM %[1]s WHERE id = ANY($1::bigint[])
 UNION ALL SELECT p.origin, n.id, n.%[2]s, p.trail || n.id, n.id = ANY(p.trail)
 FROM %[1]s n JOIN path p ON n.id = p.parent_id WHERE NOT p.cycle AND cardinality(p.trail) <= 2500
) SELECT origin, id, parent_id, cycle, cardinality(trail) FROM path LIMIT 10001`, quoteIdent(layout.Table), quoteIdent(layout.FieldByName[layout.Tree.ParentField].Column))
	rows, err := s.queryer.Query(ctx, query, ids)
	if err != nil {
		return nil, nil, err
	}
	allIDs := map[int64]bool{}
	count := 0
	for rows.Next() {
		var origin int64
		var step treePathStep
		var cycle bool
		if err := rows.Scan(&origin, &step.id, &step.parent, &cycle, &step.depth); err != nil {
			rows.Close()
			return nil, nil, err
		}
		count++
		if count > 10000 || step.depth > 2500 {
			rows.Close()
			return nil, nil, treeSizeError()
		}
		if cycle {
			rows.Close()
			return nil, nil, recordError(RecordErrorConstraintViolation, "tree contains a cycle", nil, nil)
		}
		paths[origin] = append(paths[origin], step)
		allIDs[step.id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	for _, path := range paths {
		sort.Slice(path, func(i, j int) bool { return path[i].depth > path[j].depth })
	}
	union := make([]int64, 0, len(allIDs))
	for id := range allIDs {
		union = append(union, id)
	}
	if len(union) == 0 {
		return paths, []Record{}, nil
	}
	if len(order) == 0 {
		order = []RecordSort{{Field: "name"}}
	}
	page, err := s.withTreePredicate(recordSourceColumn("id")+" = ANY($1::bigint[])", []any{union}).listRecords(ctx, layout, layout.Entity, RecordListParams{Limit: len(union), Sort: order})
	return paths, page.Records, err
}

func readableTreePath(steps []treePathStep, records map[int64]Record, parentField string) ([]Record, bool) {
	if len(steps) == 0 || steps[0].parent != nil {
		return nil, false
	}
	path := make([]Record, 0, len(steps))
	for _, step := range steps {
		record, ok := records[step.id]
		if !ok {
			return nil, false
		}
		if _, ok := record[parentField]; !ok {
			return nil, false
		}
		path = append(path, record)
	}
	return path, true
}

func treeAccessError() error {
	return recordError(RecordErrorPermissionDenied, "tree path or destination is unavailable", nil, nil)
}
func isTreeAccessError(err error) bool {
	var e RecordError
	return errors.As(err, &e) && (e.Code == RecordErrorPermissionDenied || e.Code == RecordErrorNotFound)
}
func treeSizeError() error {
	return recordError(RecordErrorInvalidRequest, "tree path or context is too large; narrow the search", nil, nil)
}

// ResolveTreeName resolves an HTTP anchor using the same read scope as traversal.
func (s RecordStore) ResolveTreeName(ctx context.Context, app, entity, name string) (int64, error) {
	record, err := s.FindRecordByIdentity(ctx, app, entity, RecordInput{"name": []byte(strconv.Quote(name))})
	if err != nil {
		return 0, err
	}
	return activityRecordID(record)
}
