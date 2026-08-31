package imports

import (
	"testing"

	"github.com/hapyco/dygo/pkg/dygo"
)

func TestValidateHeadersRejectsDuplicateAndBlankColumns(t *testing.T) {
	for name, headers := range map[string][]string{
		"blank":     {"email", " "},
		"duplicate": {"email", "email"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateHeaders(headers); err == nil {
				t.Fatal("validateHeaders() error = nil")
			} else {
				var actionErr dygo.ActionError
				if !asActionError(err, &actionErr) || actionErr.Code != "invalid_request" {
					t.Fatalf("validateHeaders() error = %v, want invalid_request", err)
				}
			}
		})
	}
}

func TestValidateHeadersAcceptsStableCSVColumns(t *testing.T) {
	if err := validateHeaders([]string{"email", "full-name", "department"}); err != nil {
		t.Fatalf("validateHeaders() error = %v, want nil", err)
	}
}

func asActionError(err error, target *dygo.ActionError) bool {
	value, ok := err.(dygo.ActionError)
	if ok {
		*target = value
	}
	return ok
}
