package schema

import (
	"strings"
	"testing"

	"github.com/hapyco/dygo/internal/entity/fieldtype"
)

func TestTreeMetadata(t *testing.T) {
	valid := `label: Node
name: {strategy: manual}
tree: {parent-field: parent, label-field: title}
fields:
  - {name: parent, label: Parent, type: link, index: true, options: {entity: node}}
  - {name: title, label: Title, type: text}
`
	for _, tc := range []struct {
		name, old, replacement string
		collection             bool
	}{
		{name: "valid"},
		{name: "required parent", old: "index: true", replacement: "index: true, required: true"},
		{name: "unique parent", old: "index: true", replacement: "index: true, unique: true"},
		{name: "missing index", old: "index: true", replacement: "index: false"},
		{name: "disabled foreign key", old: "entity: node", replacement: "entity: node, foreign-key: false"},
		{name: "default parent", old: "index: true", replacement: "index: true, default: null"},
		{name: "computed parent", old: "index: true", replacement: "index: true, fetch: {from: parent.parent}"},
		{name: "invalid label", old: "label-field: title", replacement: "label-field: parent"},
		{name: "missing parent", old: "parent-field: parent", replacement: "parent-field: absent"},
		{name: "single", old: "name: {strategy: manual}", replacement: "is-single: true"},
		{name: "collection", old: "name: {strategy: manual}", collection: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := valid
			if tc.old != "" {
				source = strings.Replace(source, tc.old, tc.replacement, 1)
			}
			_, err := DecodeWithOptions([]byte(source), fieldtype.DefaultRegistry(), DecodeOptions{IsCollection: tc.collection})
			if (err == nil) != (tc.name == "valid") {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}
