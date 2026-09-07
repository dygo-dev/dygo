package fixtures

import (
	"github.com/hapyco/dygo/internal/entity/catalog"
	"strings"
	"testing"
)

func TestSecretFixtureValueRejected(t *testing.T) {
	err := validateFixtureValue("fixture.yml", Value{}, catalog.LoadedEntity{}, fixtureField{Name: "token", Type: "secret"}, catalog.TargetIndex{}, 0)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("secret fixture accepted: %v", err)
	}
}
