package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hapyco/dygo/internal/auth"
	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/internal/studiostate"
	"github.com/hapyco/dygo/pkg/dygo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func studioTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DYGO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set DYGO_TEST_DATABASE_URL for isolated PostgreSQL checks")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := fmt.Sprintf("dygo_studio_state_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	config := admin.Config().Copy()
	config.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(ctx, "DROP DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
			t.Error(err)
		}
	})
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SyncMetadataSchema(ctx, pool, root); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestStudioStatePostgresOwnershipAndValidation(t *testing.T) {
	pool := studioTestDatabase(t)
	ctx := context.Background()
	store := studiostate.New(pool)
	makeUser := func(email string, administrator bool) auth.User {
		t.Helper()
		input := db.RecordInput{"email": json.RawMessage(fmt.Sprintf("%q", email)), "full-name": json.RawMessage(`"Studio User"`), "administrator": json.RawMessage(fmt.Sprint(administrator))}
		record, err := db.NewRecordStore(pool).CreateRecordByIdentity(ctx, "core", "user", input)
		if err != nil {
			t.Fatal(err)
		}
		return auth.User{ID: record["id"].(int64), Email: email, Enabled: true, Administrator: administrator}
	}
	alice, bob := makeUser("alice@example.com", true), makeUser("bob@example.com", false)
	request := func(user auth.User, method, path, body string, status int) map[string]json.RawMessage {
		t.Helper()
		router := NewRouter(Options{Auth: &fakeAuthStore{user: user}, StudioState: store, Metadata: db.NewMetadataReader(pool), Records: db.NewRecordStore(pool), Activity: db.NewActivityReader(pool), Permissions: permissions.NewChecker(pool)})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, authenticatedRequest(method, path, body))
		if response.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, response.Code, status, response.Body.String())
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	const prefs = "/api/v1/studio/preferences"
	const saved = "/api/v1/studio/saved-filters"
	t.Run("preferences are owner scoped and strict", func(t *testing.T) {
		request(alice, "PUT", prefs+"/studio.theme", `{"value":"dark"}`, 200)
		request(bob, "PUT", prefs+"/studio.theme", `{"value":"light"}`, 200)
		for _, item := range []struct {
			user  auth.User
			value string
		}{{alice, "dark"}, {bob, "light"}} {
			data := request(item.user, "GET", prefs, "", 200)["data"]
			if string(data) != `{"studio.theme":"`+item.value+`"}` {
				t.Fatalf("owner data: %s", data)
			}
		}
		request(bob, "PUT", prefs+"/studio.theme", `{"value":"dark","user":"alice@example.com"}`, 400)
		request(alice, "PUT", prefs+"/studio.theme", `{}`, 400)
		request(alice, "PUT", prefs+"/studio.theme", `{"value":null}`, 200)
		request(alice, "DELETE", prefs+"/studio.theme", "", 200)
		if string(request(alice, "GET", prefs, "", 200)["data"]) != `{}` {
			t.Fatal("delete failed")
		}
		request(alice, "PUT", prefs+"/studio%2Ftheme", `{"value":true}`, 400)
	})
	var filterID int64
	t.Run("saved filters preserve ownership and reject invalid predicates", func(t *testing.T) {
		body := `{"entity":"core/user","label":"My users","filters":[{"field":"name","operator":"contains","value":"alice"}]}`
		data := request(alice, "POST", saved, body, 201)["data"]
		var item studiostate.SavedFilter
		if err := json.Unmarshal(data, &item); err != nil {
			t.Fatal(err)
		}
		filterID = item.ID
		request(alice, "POST", saved, body, 409)
		request(bob, "GET", saved+"?entity=core/user", "", 403)
		request(bob, "PATCH", fmt.Sprintf("%s/%d", saved, filterID), `{"label":"Stolen"}`, 404)
		request(bob, "DELETE", fmt.Sprintf("%s/%d", saved, filterID), "", 404)
		for _, filters := range []string{`null`, `[{"field":"missing","operator":"eq","value":"x"}]`, `[{"field":"password","operator":"eq","value":"x"}]`, `[{"field":"id","operator":"eq","value":"wrong"}]`, `[{"field":"enabled","operator":"contains","value":"true"}]`} {
			response := httptest.NewRecorder()
			router := NewRouter(Options{Auth: &fakeAuthStore{user: alice}, StudioState: store})
			router.ServeHTTP(response, authenticatedRequest("POST", saved, `{"entity":"core/user","label":"Bad","filters":`+filters+`}`))
			if response.Code != 400 && response.Code != 422 {
				t.Fatalf("invalid predicate accepted: %d %s", response.Code, response.Body.String())
			}
		}
		request(alice, "POST", saved, `{"entity":"core/user","label":" ","filters":[]}`, 400)
		request(alice, "POST", saved, `{"entity":"core/user","label":"Mine","filters":[],"user":"bob@example.com"}`, 400)
	})
	t.Run("field and linked Record access are validated", func(t *testing.T) {
		_, err := pool.Exec(ctx, `WITH new_role AS (INSERT INTO role(name,label,enabled) VALUES ('studio-test-reader','Reader',true) RETURNING id) INSERT INTO user_role(name,user_id,role_id) SELECT 'bob.reader',$1,id FROM new_role`, bob.ID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pool.Exec(ctx, `INSERT INTO permission(name,entity_id,role_id,"read") SELECT e.name||'.reader',e.id,r.id,true FROM entity e CROSS JOIN role r WHERE e.name IN ('core.user','core.user-role') AND r.name='studio-test-reader'`)
		if err != nil {
			t.Fatal(err)
		}
		body := `{"entity":"core/user","label":"Names","filters":[{"field":"full-name","operator":"contains","value":"User"}]}`
		request(bob, "POST", saved, body, 201)
		_, err = pool.Exec(ctx, `UPDATE permission SET field_rules='{"deny-read":["full-name"]}', "when"='{"match":"all","conditions":[{"field":"record.id","equals":"actor.user"}]}' WHERE name='core.user.reader'`)
		if err != nil {
			t.Fatal(err)
		}
		request(bob, "POST", saved, body, 403)
		data := request(bob, "GET", saved+"?entity=core/user", "", 200)["data"]
		var items []studiostate.SavedFilter
		if err := json.Unmarshal(data, &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ValidationError == "" {
			t.Fatalf("field restriction was not reported: %s", data)
		}
		request(bob, "POST", saved, `{"entity":"core/user-role","label":"Other user","filters":[{"field":"user","operator":"eq","value":"alice@example.com"}]}`, 422)
		request(bob, "POST", saved, `{"entity":"core/user-role","label":"Self","filters":[{"field":"user","operator":"eq","value":"bob@example.com"}]}`, 201)
	})
	t.Run("stale filters remain editable and deletable", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE studio_saved_filter SET filters='[{"field":"removed","operator":"eq","value":"old"}]' WHERE id=$1`, filterID); err != nil {
			t.Fatal(err)
		}
		data := request(alice, "GET", saved+"?entity=core/user", "", 200)["data"]
		var items []studiostate.SavedFilter
		if err := json.Unmarshal(data, &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ValidationError == "" || items[0].Filters[0].Field != "removed" {
			t.Fatalf("stale filter lost: %s", data)
		}
		request(alice, "PATCH", fmt.Sprintf("%s/%d", saved, filterID), `{"filters":[]}`, 200)
		request(alice, "PATCH", fmt.Sprintf("%s/%d", saved, filterID), `{"label":"Renamed"}`, 200)
	})
	t.Run("generic routes and Activity cannot expose private state", func(t *testing.T) {
		var name string
		if err := pool.QueryRow(ctx, `SELECT name FROM studio_saved_filter WHERE id=$1`, filterID).Scan(&name); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"", fmt.Sprintf("/%d", filterID), "/name/" + name, "/export", fmt.Sprintf("/%d/activity", filterID)} {
			request(alice, "GET", "/api/v1/records/studio-saved-filter"+path, "", 403)
		}
		request(alice, "POST", "/api/v1/records/studio-preference", `{"data":{"key":"studio.foo","value":"leak"}}`, 403)
		request(alice, "GET", "/api/v1/entities/studio-preference/meta", "", 403)
		if strings.Contains(string(request(alice, "GET", "/api/v1/entities", "", 200)["data"]), "studio-preference") {
			t.Fatal("private metadata visible")
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM activity a JOIN entity e ON e.id=a.entity_id WHERE e.name IN ('studio.preference','studio.saved-filter')`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("private Activity exists: %d %v", count, err)
		}
		if _, err := db.NewRecordStore(pool).GetRecordByIdentity(ctx, "studio", "saved-filter", filterID); err == nil {
			t.Fatal("unscoped Record SDK exposed private state")
		}
		request(alice, "DELETE", fmt.Sprintf("%s/%d", saved, filterID), "", 200)
	})
	t.Run("concurrent first writes converge without duplicates", func(t *testing.T) {
		actor := dygo.Actor{UserID: alice.ID, Email: alice.Email, Administrator: true}
		var wait sync.WaitGroup
		start := make(chan struct{})
		errs := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wait.Add(1)
			go func(i int) {
				defer wait.Done()
				<-start
				errs <- store.PutPreference(ctx, actor, "studio.concurrent", json.RawMessage(fmt.Sprint(i)))
			}(i)
		}
		close(start)
		wait.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Error(err)
			}
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM studio_preference WHERE user_id=$1 AND key='studio.concurrent'`, alice.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("concurrent row count: %d %v", count, err)
		}
	})
	t.Run("authentication is required", func(t *testing.T) {
		response := httptest.NewRecorder()
		NewRouter(Options{Auth: &fakeAuthStore{user: alice}, StudioState: store}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, prefs, nil))
		if response.Code != 401 {
			t.Fatalf("unauthenticated: %d", response.Code)
		}
	})
}
