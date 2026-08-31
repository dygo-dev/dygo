// Package accesspolicy defines the validated conditional-access AST shared by
// access metadata, SQL scopes, proposed Record evaluation, and explanations.
package accesspolicy

// When limits a role grant to Records that match its conditions.
type When struct {
	Match      string      `json:"match"`
	Conditions []Condition `json:"conditions"`
}

// Condition compares one Record field path with actor context or assignment Records.
type Condition struct {
	Field  string      `json:"field"`
	Equals string      `json:"equals,omitempty"`
	In     *Membership `json:"in,omitempty"`
}

// Membership checks a Record value against an ordinary assignment Entity.
type Membership struct {
	Entity string            `json:"entity"`
	Value  string            `json:"value"`
	Where  map[string]string `json:"where"`
}

// Fields denies selected fields for one role grant.
type Fields struct {
	DenyRead  []string `json:"deny-read,omitempty"`
	DenyWrite []string `json:"deny-write,omitempty"`
}
