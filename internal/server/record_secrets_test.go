package server

import (
	"context"
	"github.com/hapyco/dygo/internal/permissions"
	"github.com/hapyco/dygo/pkg/dygo"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type secretStatusStore struct {
	*fakeRecordStore
	calls int
}

func (s *secretStatusStore) SecretStatus(context.Context, string, int64) (dygo.SecretStatus, error) {
	s.calls++
	return dygo.SecretStatus{Fields: map[string]bool{"token": true}}, nil
}
func TestSecretStatusRouteRequiresPermission(t *testing.T) {
	for _, denied := range []bool{false, true} {
		store := &secretStatusStore{fakeRecordStore: &fakeRecordStore{}}
		checker := &fakePermissionChecker{}
		if denied {
			checker.err = permissions.Error{Code: permissions.ErrorDenied, Message: "permission denied"}
		}
		response := httptest.NewRecorder()
		NewRouter(Options{Auth: validFakeAuthStore(), Records: store, Permissions: checker}).ServeHTTP(response, authenticatedRequest(http.MethodGet, "/api/v1/records/user/1/secret-status", ""))
		if denied {
			if response.Code != http.StatusForbidden || store.calls != 0 {
				t.Fatal("status accessed without permission")
			}
		} else if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"token":true`) {
			t.Fatalf("status response: %s", response.Body.String())
		}
	}
}
