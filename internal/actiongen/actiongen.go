// Package actiongen generates Entity action scaffolds and project runner wiring.
package actiongen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hapyco/dygo/internal/app/manifest"
	"github.com/hapyco/dygo/internal/entity/catalog"
	"github.com/hapyco/dygo/internal/project"
	"github.com/hapyco/dygo/internal/runnergen"
	"github.com/hapyco/dygo/internal/shape"
)

// Result describes files touched by Entity action scaffold generation.
type Result struct {
	AppName string
	Entity  string

	ActionFile string
	RunnerFile string

	ActionFileStatus string
	RunnerFileStatus string

	ActionFileCreated bool
	RunnerFileWritten bool
}

// GenerateOptions controls Entity action scaffold generation.
type GenerateOptions struct {
	Root       string
	AppName    string
	EntityName string
	DryRun     bool
	Force      bool
}

// Generate creates an Entity action scaffold and updates generated runner wiring.
func Generate(root string, appName string, entityName string) (Result, error) {
	return GenerateWithOptions(GenerateOptions{Root: root, AppName: appName, EntityName: entityName})
}

// GenerateWithOptions creates or previews an Entity action scaffold and runner wiring.
func GenerateWithOptions(options GenerateOptions) (Result, error) {
	root := filepath.Clean(options.Root)
	appName := strings.TrimSpace(options.AppName)
	entityName := strings.TrimSpace(options.EntityName)
	if appName == "" {
		return Result{}, fmt.Errorf("app name is required")
	}
	if entityName == "" {
		return Result{}, fmt.Errorf("entity name is required")
	}
	if err := runnergen.RequireGeneratedProjectRoot(root); err != nil {
		return Result{}, err
	}

	modulePath, err := runnergen.ReadModulePath(root)
	if err != nil {
		return Result{}, err
	}
	metadata, err := project.LoadMetadata(root)
	if err != nil {
		return Result{}, err
	}

	app, ok := findApp(metadata.Apps, appName)
	if !ok {
		return Result{}, fmt.Errorf("app %q not found", appName)
	}
	if !runnergen.IsProjectOwnedApp(root, app.Dir) {
		return Result{}, fmt.Errorf("app %q is not a project-owned app under apps/", appName)
	}
	entity, ok := findEntity(metadata.Entities, appName, entityName)
	if !ok {
		return Result{}, fmt.Errorf("entity %q not found in app %q", entityName, appName)
	}
	if entity.IsCollection() {
		return Result{}, fmt.Errorf("entity %q in app %q is a collection; generate actions for the parent Entity that owns collection row usage", entityName, appName)
	}

	entityDir := filepath.Dir(entity.Path)
	actionDir := filepath.Join(entityDir, shape.EntityActionsDir)
	actionFile := filepath.Join(actionDir, shape.EntityActionsFile)
	runnerFile := filepath.Join(root, "cmd", "dygo", "main.go")

	if err := preflightPath(entityDir, wantDirectory); err != nil {
		return Result{}, err
	}
	if err := preflightPath(actionDir, wantDirectory); err != nil {
		return Result{}, err
	}
	if err := preflightPath(actionFile, wantRegularFile); err != nil {
		return Result{}, err
	}
	if exists, hasRegister, err := runnergen.InspectFunctionFile(actionFile, "Register"); err != nil {
		return Result{}, err
	} else if exists && !hasRegister {
		return Result{}, fmt.Errorf("%s exists but does not expose Register(registry dygo.EntityActionRegistry) error", actionFile)
	}
	if err := runnergen.PreflightGeneratedFile(runnerFile, runnergen.ActionManualSnippet(root, modulePath, appName, entityName, actionDir)); err != nil {
		return Result{}, err
	}

	actionSource, err := renderEntityActionSource(appName, entityName)
	if err != nil {
		return Result{}, err
	}
	runnerUpdate, err := runnergen.Render(root, runnergen.RenderOptions{ActionTarget: runnergen.ActionTarget{AppName: appName, EntityName: entityName}})
	if err != nil {
		return Result{}, err
	}

	actionStatus, err := actionFileStatus(actionFile, options.DryRun)
	if err != nil {
		return Result{}, err
	}
	runnerStatus, err := runnergen.GeneratedFileStatus(runnerFile, runnerUpdate.Source, options.DryRun)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		AppName:           appName,
		Entity:            entityName,
		ActionFile:        actionFile,
		RunnerFile:        runnerFile,
		ActionFileStatus:  actionStatus,
		RunnerFileStatus:  runnerStatus,
		ActionFileCreated: actionStatus == "created",
		RunnerFileWritten: runnerStatus == "created" ||
			runnerStatus == "updated",
	}
	if options.DryRun {
		return result, nil
	}
	if err := os.MkdirAll(actionDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create Entity action directory %s: %w", actionDir, err)
	}
	if actionStatus == "created" {
		if err := os.WriteFile(actionFile, actionSource, 0o644); err != nil {
			return Result{}, fmt.Errorf("write Entity action file %s: %w", actionFile, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(runnerFile), 0o755); err != nil {
		return Result{}, fmt.Errorf("create runner directory %s: %w", filepath.Dir(runnerFile), err)
	}
	written, err := runnergen.WriteFileIfChanged(runnerFile, runnerUpdate.Source)
	if err != nil {
		return Result{}, err
	}
	result.RunnerFileWritten = written
	if result.RunnerFileStatus == "" {
		result.RunnerFileStatus = writeStatus(written)
	}
	return result, nil
}

func findApp(apps []manifest.LoadedApp, appName string) (manifest.LoadedApp, bool) {
	for _, app := range apps {
		if app.Manifest.Name == appName {
			return app, true
		}
	}
	return manifest.LoadedApp{}, false
}

func findEntity(entities []catalog.LoadedEntity, appName string, entityName string) (catalog.LoadedEntity, bool) {
	for _, entity := range entities {
		if entity.AppName == appName && entity.Entity.Name == entityName {
			return entity, true
		}
	}
	return catalog.LoadedEntity{}, false
}

type pathExpectation int

const (
	wantDirectory pathExpectation = iota
	wantRegularFile
)

func preflightPath(path string, want pathExpectation) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", path, err)
	}
	switch want {
	case wantDirectory:
		if !info.IsDir() {
			return fmt.Errorf("%s must be a directory", path)
		}
	case wantRegularFile:
		if info.IsDir() || !info.Mode().IsRegular() {
			return fmt.Errorf("%s must be a regular file", path)
		}
	}
	return nil
}

func actionFileStatus(path string, dryRun bool) (string, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if dryRun {
				return "would create", nil
			}
			return "created", nil
		}
		return "", fmt.Errorf("stat Entity action file %s: %w", path, err)
	}
	return "existing", nil
}

func renderEntityActionSource(appName string, entityName string) ([]byte, error) {
	name := runnergen.ExportedIdentifier(entityName)
	actionName := entityName + "-action"
	source := fmt.Sprintf(`package actions

import (
	"context"

	"github.com/hapyco/dygo/pkg/dygo"
)

func Register(registry dygo.EntityActionRegistry) error {
	return registry.RegisterEntity(%[2]q, %[3]q, dygo.EntityActionDefinition{
		Name:      %[4]q,
		Label:     %[5]q,
		Selection: dygo.ActionSelectionRecord,
	}, run%[1]sAction)
}

func run%[1]sAction(ctx context.Context, call dygo.EntityActionCall) (any, error) {
	// TODO(%[2]s/%[3]s): implement action behavior.
	return nil, nil
}
`, name, appName, entityName, actionName, labelForName(actionName))
	return runnergen.FormatGoSource([]byte(source))
}

func labelForName(name string) string {
	parts := strings.Split(name, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func writeStatus(written bool) string {
	if written {
		return "updated"
	}
	return "unchanged"
}
