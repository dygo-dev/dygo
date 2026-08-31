package permissions

import (
	"fmt"
	"strings"

	"github.com/hapyco/dygo/internal/db"
	"github.com/hapyco/dygo/internal/shape"
)

type actionSpec struct {
	Action Action
	Column string
}

var actionSpecs = []actionSpec{
	{Action: ActionRead, Column: `"read"`},
	{Action: ActionCreate, Column: `"create"`},
	{Action: ActionUpdate, Column: `"update"`},
	{Action: ActionDelete, Column: `"delete"`},
	{Action: ActionExport, Column: `"export"`},
	{Action: ActionPrint, Column: `"print"`},
}

// SupportedActions returns the stable permission actions supported by dygo.
func SupportedActions() []Action {
	actions := make([]Action, len(actionSpecs))
	for index, spec := range actionSpecs {
		actions[index] = spec.Action
	}
	return actions
}

// ParseAction normalizes and validates a permission action name.
func ParseAction(value string) (Action, error) {
	action := Action(strings.TrimSpace(value))
	if err := shape.ValidateMetadataName("permission action", string(action)); err != nil {
		return "", err
	}
	return action, nil
}

func actionColumn(action Action) (string, bool) {
	for _, spec := range actionSpecs {
		if spec.Action == action {
			return spec.Column, true
		}
	}
	return "", false
}

// IsBuiltInAction reports whether action is stored in a dedicated Permission column.
func IsBuiltInAction(action Action) bool {
	_, ok := actionColumn(action)
	return ok
}

// ValidateMetadata verifies that core.permission metadata supports the runtime action catalog.
func ValidateMetadata(meta db.MetadataEntityMeta) error {
	fields := db.MetadataFieldsByName(meta)
	for _, spec := range actionSpecs {
		field, ok := db.RecordAddressableFieldByName(fields, string(spec.Action))
		if !ok {
			return fmt.Errorf("permission action field %q is missing", spec.Action)
		}
		if field.Type != "boolean" {
			return fmt.Errorf("permission action field %q must be boolean", spec.Action)
		}
	}
	retired, ok := db.RecordAddressableFieldByName(fields, "retired")
	if !ok {
		return fmt.Errorf("permission retired field is missing")
	}
	if retired.Type != "boolean" {
		return fmt.Errorf("permission retired field must be boolean")
	}
	actions, ok := db.RecordAddressableFieldByName(fields, "actions")
	if !ok {
		return fmt.Errorf("permission actions field is missing")
	}
	if actions.Type != "json" {
		return fmt.Errorf("permission actions field must be json")
	}
	for _, name := range []string{"when", "field-rules"} {
		field, ok := db.RecordAddressableFieldByName(fields, name)
		if !ok {
			return fmt.Errorf("permission %s field is missing", name)
		}
		if field.Type != "json" {
			return fmt.Errorf("permission %s field must be json", name)
		}
	}
	return nil
}
