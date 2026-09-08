package dygo

import "context"

// TreeSearchParams filters a page of matches, optionally excluding a subtree.
type TreeSearchParams struct {
	RecordListParams
	ExcludeSubtree int64
}

// TreeNode contains a readable Record and permission-safe tree presentation state.
type TreeNode struct {
	Record          Record `json:"record"`
	HasChildren     bool   `json:"hasChildren"`
	Matched         bool   `json:"matched"`
	PathUnavailable bool   `json:"pathUnavailable"`
	Parent          string `json:"parent,omitempty"`
}

// TreeResult separates paged matches from their complete readable ancestor context.
type TreeResult struct {
	Nodes   []TreeNode
	Context []Record
	Count   int
	Limit   int
	Offset  int
}

// TreeData traverses one Entity using its RecordData actor and transaction.
// Anchors are internal Record IDs. Move accepts a parent Record name, or nil for a root.
type TreeData interface {
	Roots(context.Context, RecordListParams) (TreeResult, error)
	Children(context.Context, int64, RecordListParams) (TreeResult, error)
	Descendants(context.Context, int64, RecordListParams) (TreeResult, error)
	Ancestors(context.Context, int64) (TreeResult, error)
	Path(context.Context, int64) (TreeResult, error)
	Move(context.Context, int64, *string) (Record, error)
	Search(context.Context, TreeSearchParams) (TreeResult, error)
}
