---
name: dygo-access-control
description: Design, implement, or review dygo roles, Entity access metadata, Permissions, and permission-aware Business App behavior. Use when access to Records or business actions is central to the task.
---

# dygo Access Control

Make server-side access the source of truth. Default to deny.

## Start

Read `docs/access.md`, `docs/auth.md`, and the current permission actions in `internal/permissions/actions.go`. Inspect the target App's `access/` directory and Core access metadata.

## Rules

- Define App-contributed roles in `access/_roles.yml`.
- Define Entity grants in `access/<entity>.access.yml`.
- Use canonical App and Entity identities. Do not use labels or route slugs as permission identity.
- Keep the built-in action set small. Do not model business lifecycle verbs as global CRUD actions.
- Enforce access on the server before you add UI visibility rules.
- Treat Administrator bypass as a narrow framework rule, not an App authorization shortcut.
- Do not add a second permission engine or a private Core-only permission path.
- Preserve database-owned Studio access Records when the documented sync contract requires it.
- Make denials observable without exposing secrets or protected Record data.

## Check

Use `dygo access validate`, `dygo access show <app>/<entity>`, and `dygo access apply --dry-run`. Test the actual permission boundary when the change can expose or mutate business data.
