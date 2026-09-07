package schema

import (
	"github.com/hapyco/dygo/internal/entity/fieldtype"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestSecretMetadataRestrictions(t *testing.T) {
	for _, option := range []string{"", "index: true", "unique: true", "default: private-value"} {
		var entity Entity
		err := yaml.Unmarshal([]byte("label: Connection\nfields:\n  - name: token\n    label: Token\n    type: secret\n    "+option+"\n"), &entity)
		if err != nil {
			t.Fatal(err)
		}
		entity.Name = "connection"
		entity.Naming = Naming{Strategy: NamingStrategyRandom}
		err = entity.Validate(fieldtype.DefaultRegistry())
		if (err == nil) != (option == "") {
			t.Fatalf("option %q: %v", option, err)
		}
	}
}
