package fieldtype

import (
	"errors"
	"fmt"
	"strings"
)

// Options contains type-specific field settings from Entity metadata.
type Options struct {
	Values       []string     `yaml:"values,omitempty"`
	App          string       `yaml:"app,omitempty"`
	Entity       string       `yaml:"entity,omitempty"`
	DisplayField string       `yaml:"display-field,omitempty"`
	Filters      []LinkFilter `yaml:"filters,omitempty"`
	ForeignKey   *bool        `yaml:"foreign-key,omitempty"`
}

// LinkFilter limits the records offered by a link editor. A value beginning
// with "$" is resolved from the current parent form by Studio.
type LinkFilter struct {
	Field    string `yaml:"field" json:"field"`
	Operator string `yaml:"operator" json:"operator"`
	Value    any    `yaml:"value" json:"value"`
	From     string `yaml:"from,omitempty" json:"from,omitempty"`
}

// NoOptions rejects type-specific field options.
func NoOptions(options Options) error {
	var problems []string
	if len(options.Values) > 0 {
		problems = append(problems, "values are not supported")
	}
	if options.App != "" {
		problems = append(problems, "app is not supported")
	}
	if options.Entity != "" {
		problems = append(problems, "entity is not supported")
	}
	if options.ForeignKey != nil {
		problems = append(problems, "foreign-key is not supported")
	}
	if options.DisplayField != "" {
		problems = append(problems, "display-field is not supported")
	}
	if len(options.Filters) > 0 {
		problems = append(problems, "filters are not supported")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// SelectOptions validates select field options.
func SelectOptions(options Options) error {
	var problems []string
	if options.App != "" {
		problems = append(problems, "app is not supported")
	}
	if options.Entity != "" {
		problems = append(problems, "entity is not supported")
	}
	if options.ForeignKey != nil {
		problems = append(problems, "foreign-key is not supported")
	}
	if options.DisplayField != "" {
		problems = append(problems, "display-field is not supported")
	}
	if len(options.Filters) > 0 {
		problems = append(problems, "filters are not supported")
	}
	if len(options.Values) == 0 {
		problems = append(problems, "values are required")
	}
	seen := map[string]struct{}{}
	for _, value := range options.Values {
		if strings.TrimSpace(value) == "" {
			problems = append(problems, "values must not be empty")
			continue
		}
		if _, ok := seen[value]; ok {
			problems = append(problems, fmt.Sprintf("duplicate value %q", value))
		}
		seen[value] = struct{}{}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// LinkOptions validates link field options.
func LinkOptions(options Options) error {
	return entityOptions(options, true)
}

// EntityOptions validates entity-target field options.
func EntityOptions(options Options) error {
	return entityOptions(options, false)
}

func entityOptions(options Options, allowForeignKey bool) error {
	var problems []string
	if len(options.Values) > 0 {
		problems = append(problems, "values are not supported")
	}
	if !allowForeignKey && options.ForeignKey != nil {
		problems = append(problems, "foreign-key is not supported")
	}
	if strings.TrimSpace(options.App) != "" && !IsName(options.App) {
		problems = append(problems, fmt.Sprintf("app %q must be kebab-case", options.App))
	}
	if strings.TrimSpace(options.Entity) == "" {
		problems = append(problems, "entity is required")
	} else if !IsName(options.Entity) {
		problems = append(problems, fmt.Sprintf("entity %q must be kebab-case", options.Entity))
	}
	if options.DisplayField != "" && !IsName(options.DisplayField) {
		problems = append(problems, fmt.Sprintf("display-field %q must be kebab-case", options.DisplayField))
	}
	for _, filter := range options.Filters {
		if !IsName(filter.Field) {
			problems = append(problems, fmt.Sprintf("filter field %q must be kebab-case", filter.Field))
		}
		if filter.From != "" && !IsName(filter.From) {
			problems = append(problems, fmt.Sprintf("filter source %q must be kebab-case", filter.From))
		}
		if filter.Value == nil && filter.From == "" {
			problems = append(problems, fmt.Sprintf("filter %s value is required", filter.Field))
		} else if filter.Value != nil && filter.From != "" {
			problems = append(problems, fmt.Sprintf("filter %s must use value or from, not both", filter.Field))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
