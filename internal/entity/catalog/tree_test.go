package catalog

import (
	"testing"

	"github.com/hapyco/dygo/internal/entity/fieldtype"
	"github.com/hapyco/dygo/internal/entity/schema"
)

func TestTreeParentIdentity(t *testing.T) {
	owner := LoadedEntity{AppName: "org", Entity: schema.Entity{Name: "node", Tree: &schema.Tree{ParentField: "parent"}}}
	field := schema.Field{Name: "parent", Type: "link"}
	for _, tc := range []struct {
		app, entity string
		valid       bool
	}{{"org", "node", true}, {"other", "node", false}, {"org", "other", false}} {
		t.Run(tc.app+"/"+tc.entity, func(t *testing.T) {
			var problems []string
			validateLinkOptions(owner, field, LoadedEntity{AppName: tc.app, Entity: schema.Entity{Name: tc.entity}}, fieldtype.DefaultRegistry(), &problems)
			if (len(problems) == 0) != tc.valid {
				t.Fatalf("tree parent identity: %v", problems)
			}
		})
	}
}
