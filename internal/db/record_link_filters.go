package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func (s RecordStore) validateFilteredLinks(ctx context.Context, layout recordLayout, input RecordInput, base Record) error {
	for _, field := range layout.Fields {
		if field.Type != "link" || len(field.Options.Filters) == 0 {
			continue
		}
		selected := fetchInputValue(field, input, base)
		if rawIsNull(selected) {
			continue
		}
		var name string
		if err := json.Unmarshal(selected, &name); err != nil || strings.TrimSpace(name) == "" {
			return recordError(RecordErrorValidation, "link field must be a record name", map[string]any{"entity": layout.Entity, "field": field.Name}, err)
		}
		target, err := s.fetchTargetLayout(ctx, layout, field)
		if err != nil {
			return err
		}
		filters := []RecordFilter{{Field: "name", Operator: "eq", Value: name}}
		for _, filter := range field.Options.Filters {
			filterValue := filter.Value
			if filter.From != "" {
				filterValue = "$" + filter.From
			}
			value, resolved, err := resolveLinkFilterValue(filterValue, input, base)
			if err != nil {
				return recordError(RecordErrorValidation, "link filter value is invalid", map[string]any{"entity": layout.Entity, "field": field.Name, "filter": filter.Field}, err)
			}
			operator := filter.Operator
			if operator == "" {
				operator = "eq"
			}
			if !resolved && operator != "empty" && operator != "not-empty" {
				continue
			}
			filters = append(filters, RecordFilter{Field: filter.Field, Operator: operator, Value: value})
		}
		unscoped := s
		unscoped.scope = nil
		result, err := unscoped.listRecords(ctx, target, target.Entity, RecordListParams{Limit: 1, Filters: filters})
		if err != nil {
			return err
		}
		if len(result.Records) == 0 {
			return recordError(RecordErrorValidation, "link target does not match field filters", map[string]any{"entity": layout.Entity, "field": field.Name, "target": name}, nil)
		}
	}
	return nil
}

func resolveLinkFilterValue(value any, input RecordInput, base Record) (string, bool, error) {
	if token, ok := value.(string); ok && strings.HasPrefix(token, "$") {
		name := strings.Trim(strings.TrimPrefix(token, "$"), "{}")
		raw, exists := input[name]
		if !exists {
			if current, found := base[name]; found {
				return linkFilterString(current), true, nil
			}
			return "", false, nil
		}
		if rawIsNull(raw) {
			return "", false, nil
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return "", false, err
		}
		return linkFilterString(decoded), true, nil
	}
	return linkFilterString(value), value != nil, nil
}

func linkFilterString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}
