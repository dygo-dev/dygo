package dygo

import (
	"context"
	"encoding/json"
)

// ActionSelection describes how many Records an Entity action accepts.
type ActionSelection string

const (
	ActionSelectionRecord     ActionSelection = "record"
	ActionSelectionSelection  ActionSelection = "selection"
	ActionSelectionCollection ActionSelection = "collection"
)

// EntityActionDefinition describes one callable action owned by an Entity.
type EntityActionDefinition struct {
	Name      string          `json:"name"`
	Label     string          `json:"label"`
	Selection ActionSelection `json:"selection"`
}

// EntityActionCall contains one authorized, transactional action invocation.
type EntityActionCall struct {
	Actor         Actor
	RecordIDs     []int64
	Input         json.RawMessage
	Records       RecordData
	Jobs          JobData
	Files         FileData
	Timeline      TimelineData
	Notifications NotificationData
}

// EntityActionFunc handles one Entity action invocation.
type EntityActionFunc func(context.Context, EntityActionCall) (any, error)

// EntityActionRegistry is the public app-facing Entity action registration API.
type EntityActionRegistry interface {
	RegisterEntity(appName string, entity string, definition EntityActionDefinition, fn EntityActionFunc) error
}

// EntityActionRegistrar registers compiled app Entity actions with dygo.
type EntityActionRegistrar func(EntityActionRegistry) error

// ActionError is a stable App error that can cross the HTTP boundary.
type ActionError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e ActionError) Error() string { return e.Message }
