package access

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/internal/shape"
)

// Explanation reports one user's compiled access result without exposing SQL.
type Explanation struct {
	User         string
	Target       shape.AppRef
	Action       permissions.Action
	RecordID     int64
	Roles        []string
	MatchedRoles []string
	RowAllowed   bool
	DeniedRead   []string
	DeniedWrite  []string
}

// Explain compiles and evaluates the same scope used by Record operations.
func Explain(ctx context.Context, databaseURL string, target shape.AppRef, email string, action permissions.Action, recordID int64) (Explanation, error) {
	pool, err := db.OpenRuntimePool(ctx, databaseURL)
	if err != nil {
		return Explanation{}, err
	}
	defer pool.Close()
	var actor permissions.Actor
	if err := pool.QueryRow(ctx, `SELECT id, COALESCE(administrator, false) FROM "user" WHERE email = $1 AND COALESCE(enabled, false) = true`, strings.TrimSpace(email)).Scan(&actor.UserID, &actor.Administrator); err != nil {
		return Explanation{}, fmt.Errorf("load user %q: %w", email, err)
	}
	actor.Email = strings.TrimSpace(email)
	scope, err := permissions.NewChecker(pool).RecordScope(ctx, permissions.Request{Actor: actor, Resource: permissions.Resource{Kind: permissions.ResourceEntity, App: target.App, Name: target.Name}, Action: action})
	if err != nil {
		return Explanation{}, err
	}
	result := Explanation{User: actor.Email, Target: target, Action: action, RecordID: recordID, Roles: scope.Roles}
	if actor.Administrator {
		result.RowAllowed = true
		return result, nil
	}
	if recordID > 0 {
		result.RowAllowed, err = explainPredicate(ctx, pool, target, recordID, scope.Where, scope.Args)
		if err != nil {
			return Explanation{}, err
		}
		for _, role := range scope.Roles {
			matched, err := explainPredicate(ctx, pool, target, recordID, scope.RoleWhere[role], scope.Args)
			if err != nil {
				return Explanation{}, err
			}
			if matched {
				result.MatchedRoles = append(result.MatchedRoles, role)
			}
		}
		result.DeniedRead, err = explainDeniedFields(ctx, pool, target, recordID, scope.FieldRead, scope.Args)
		if err != nil {
			return Explanation{}, err
		}
		result.DeniedWrite, err = explainDeniedFields(ctx, pool, target, recordID, scope.FieldWrite, scope.Args)
		if err != nil {
			return Explanation{}, err
		}
	} else {
		result.RowAllowed = scope.Where == "TRUE"
		result.DeniedRead = sortedKeys(scope.FieldRead)
		result.DeniedWrite = sortedKeys(scope.FieldWrite)
	}
	return result, nil
}

func explainPredicate(ctx context.Context, queryer db.RecordQueryer, target shape.AppRef, recordID int64, predicate string, args []any) (bool, error) {
	args = append(append([]any{}, args...), recordID)
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s AS %s WHERE (%s) AND %s.id = $%d)", quoteExplain(db.EntityStorageTableName(target.App, target.Name)), quoteExplain("_dygo_record"), predicate, quoteExplain("_dygo_record"), len(args))
	var allowed bool
	if err := queryer.QueryRow(ctx, query, args...).Scan(&allowed); err != nil {
		return false, err
	}
	return allowed, nil
}

func explainDeniedFields(ctx context.Context, queryer db.RecordQueryer, target shape.AppRef, recordID int64, fields map[string]string, args []any) ([]string, error) {
	denied := []string{}
	for _, field := range sortedKeys(fields) {
		allowed, err := explainPredicate(ctx, queryer, target, recordID, fields[field], args)
		if err != nil {
			return nil, err
		}
		if !allowed {
			denied = append(denied, field)
		}
	}
	return denied, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func quoteExplain(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
