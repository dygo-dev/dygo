package schema

func validateTree(e Entity, fields map[string]Field, problems *[]string) {
	if e.Tree == nil {
		return
	}
	if e.IsSingle || e.IsCollection {
		*problems = append(*problems, "tree is not supported on Single or Collection Entities")
	}
	f, ok := fields[e.Tree.ParentField]
	if !ok || f.Type != "link" || f.Required || f.Unique || f.Default.Kind != 0 || f.Fetch != nil || (f.Options.ForeignKey != nil && !*f.Options.ForeignKey) {
		*problems = append(*problems, "tree parent-field must be a nullable writable Link with a foreign key, without default, fetch, or uniqueness")
	}
	indexed := f.Index
	for _, index := range e.Indexes {
		if len(index.Fields) > 0 && index.Fields[0] == f.Name {
			indexed = true
		}
	}
	for _, constraint := range e.Constraints {
		if constraint.Type == "unique" && len(constraint.Fields) == 1 && constraint.Fields[0] == f.Name {
			*problems = append(*problems, "tree parent-field must not be unique")
		}
	}
	if !indexed {
		*problems = append(*problems, "tree parent-field requires an index")
	}
	if e.Tree.LabelField != "" {
		label, ok := fields[e.Tree.LabelField]
		if !ok || label.Type != "text" {
			*problems = append(*problems, "tree label-field must be a stored text field")
		}
	}
}
