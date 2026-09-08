package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/auth"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/dygodata"
	"github.com/hapyco/dygo/internal/hooks"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
)

func TestTreePostgresHTTPAndSDKScopes(t *testing.T) {
	pool := studioTestDatabase(t)
	ctx := context.Background()
	// Extend a normal test-only Core Entity inside this isolated database. The
	// database suite separately verifies authored metadata and schema activation.
	_, err := pool.Exec(ctx, `ALTER TABLE country ADD COLUMN parent_id bigint REFERENCES country(id); CREATE INDEX country_parent_idx ON country(parent_id);
 UPDATE entity SET tree='{"parent-field":"parent","label-field":"code"}' WHERE name='core.country';
 INSERT INTO field(name,entity_id,field_name,label,type,position,options,index) SELECT 'core.country.parent',id,'parent','Parent','link',100,'{"entity":"country"}',true FROM entity WHERE name='core.country'`)
	if err != nil {
		t.Fatal(err)
	}
	store := db.NewRecordStore(pool)
	makeUser := func(email string, admin bool) auth.User {
		t.Helper()
		record, err := store.CreateRecordByIdentity(ctx, "core", "user", db.RecordInput{"email": []byte(fmt.Sprintf("%q", email)), "full-name": []byte(`"Tree test"`), "administrator": []byte(fmt.Sprint(admin))})
		if err != nil {
			t.Fatal(err)
		}
		return auth.User{ID: record["id"].(int64), Email: email, Enabled: true, Administrator: admin}
	}
	admin := makeUser("tree-admin@example.test", true)
	actor := makeUser("tree-user@example.test", false)
	// Country uses manual names. These synthetic labels exercise real route and SDK behavior.
	create := func(name, parent string) int64 {
		t.Helper()
		input := db.RecordInput{"name": []byte(fmt.Sprintf("%q", name)), "code": []byte(fmt.Sprintf("%q", name))}
		if parent != "" {
			input["parent"] = []byte(fmt.Sprintf("%q", parent))
		}
		r, err := store.CreateRecordByIdentity(ctx, "core", "country", input)
		if err != nil {
			t.Fatal(err)
		}
		return r["id"].(int64)
	}
	root := create("Tree Root", "")
	hidden := create("Hidden Ancestor", "Tree Root")
	leaf := create("Tree Leaf", "Hidden Ancestor")
	other := create("Other Tree", "")
	if _, err := pool.Exec(ctx, `UPDATE country SET owner_id=$1 WHERE id<>$2`, actor.ID, hidden); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `WITH r AS (INSERT INTO role(name,label,enabled) VALUES ('tree-reader','Tree Reader',true) RETURNING id) INSERT INTO user_role(name,user_id,role_id) SELECT 'tree-reader-user',$1,id FROM r`, actor.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO permission(name,entity_id,role_id,"read","update","delete","when") SELECT 'tree-reader-country',e.id,r.id,true,true,true,'{"match":"all","conditions":[{"field":"record.owner","equals":"actor.user"}]}' FROM entity e CROSS JOIN role r WHERE e.name='core.country' AND r.name='tree-reader'`)
	if err != nil {
		t.Fatal(err)
	}
	request := func(user auth.User, method, path, body string, status int) []byte {
		t.Helper()
		router := NewRouter(Options{Auth: &fakeAuthStore{user: user}, Records: store, Metadata: db.NewMetadataReader(pool), Permissions: permissions.NewChecker(pool)})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, authenticatedRequest(method, path, body))
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.Bytes()
	}
	base := "/api/v1/records/country/tree/"
	t.Run("HTTP parsing and permission isolation", func(t *testing.T) {
		wire := request(actor, "GET", base+"search?name:contains=Tree+Leaf", "", 200)
		if strings.Contains(string(wire), "Hidden Ancestor") || !strings.Contains(string(wire), `"pathUnavailable":true`) {
			t.Fatalf("private path leaked or absent: %s", wire)
		}
		request(actor, "GET", base+"path?name=Tree+Leaf", "", 403)
		request(actor, "GET", base+"search?exclude-subtree=Hidden+Ancestor", "", 404)
		wire = request(admin, "GET", base+"search?exclude-subtree=Tree+Root&limit=1", "", 200)
		if !strings.Contains(string(wire), "Other Tree") || strings.Contains(string(wire), "Tree Leaf") {
			t.Fatalf("exclusion after limit: %s", wire)
		}
		request(admin, "GET", base+"children", "", 400)
		request(admin, "GET", base+"children?name=Tree+Root&name=Other+Tree", "", 400)
		request(admin, "GET", base+"roots?limit=-1", "", 400)
		request(actor, "PATCH", fmt.Sprintf("/api/v1/records/country/%d", other), `{"data":{"parent":"Hidden Ancestor"}}`, 403)
		request(actor, "DELETE", fmt.Sprintf("/api/v1/records/country/%d", root), "", 409)
		request(admin, "PATCH", fmt.Sprintf("/api/v1/records/country/%d", root), `{"data":{"parent":"Tree Leaf"}}`, 422)
	})
	t.Run("SDK actor and transaction binding", func(t *testing.T) {
		sdk := dygodata.NewRecordData(pool, nil).AsActor(dygo.Actor{UserID: actor.ID, Email: actor.Email})
		result, err := sdk.Tree("core", "country").Search(ctx, dygo.TreeSearchParams{RecordListParams: dygo.RecordListParams{Filters: []dygo.RecordFilter{{Field: "name", Operator: "eq", Value: "Tree Leaf"}}}})
		if err != nil || len(result.Nodes) != 1 || !result.Nodes[0].PathUnavailable {
			t.Fatalf("actor SDK: %+v %v", result, err)
		}
		_, err = sdk.Tree("core", "country").Path(ctx, leaf)
		if err == nil {
			t.Fatal("SDK exposed private path")
		}
		parent := "Hidden Ancestor"
		_, err = sdk.Tree("core", "country").Move(ctx, other, &parent)
		if err == nil {
			t.Fatal("SDK moved to hidden parent")
		}
		system := sdk.AsSystem("test transactional tree move")
		sentinel := errors.New("rollback")
		err = system.Transaction(ctx, func(ctx context.Context, records dygo.RecordData) error {
			parent := "Other Tree"
			if _, err := records.Tree("core", "country").Move(ctx, leaf, &parent); err != nil {
				return err
			}
			path, err := records.Tree("core", "country").Path(ctx, leaf)
			if err != nil {
				return err
			}
			if len(path.Nodes) != 2 || path.Nodes[0].Record["name"] != "Other Tree" {
				return errors.New("tree did not use transaction")
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatal(err)
		}
		r, err := store.GetRecordByIdentity(ctx, "core", "country", leaf)
		if err != nil || r["parent"] != "Hidden Ancestor" {
			t.Fatal("tree move escaped rollback")
		}
		_, err = system.Tree("core", "country").Search(ctx, dygo.TreeSearchParams{ExcludeSubtree: hidden})
		if err != nil {
			t.Fatal(err)
		}
		wire, _ := json.Marshal(result)
		if strings.Contains(string(wire), "Hidden Ancestor") {
			t.Fatal("SDK context leaks hidden parent")
		}
	})
	t.Run("App Hook tree access uses current transaction", func(t *testing.T) {
		observed := false
		registry, err := hooks.NewRecordHookRegistry([]dygo.RecordHookRegistrar{func(r dygo.RecordHookRegistry) error {
			return r.RegisterEntity("core", "country", dygo.RecordAfterUpdate, "read-tree", func(ctx context.Context, hook dygo.RecordHook) error {
				path, err := hook.Records.Tree("core", "country").Path(ctx, hook.RecordID)
				if err != nil {
					return err
				}
				observed = len(path.Nodes) == 2 && path.Nodes[0].Record["name"] == "Other Tree"
				return nil
			})
		}})
		if err != nil {
			t.Fatal(err)
		}
		parent := "Other Tree"
		_, err = dygodata.NewRecordData(pool, registry).AsActor(dygo.Actor{UserID: admin.ID, Email: admin.Email, Administrator: true}).Tree("core", "country").Move(ctx, leaf, &parent)
		if err != nil || !observed {
			t.Fatalf("Hook tree path: observed=%t %v", observed, err)
		}
	})
}
