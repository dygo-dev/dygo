---
name: dygo-entity-modeling
description: Design or change dygo Entity metadata, Fields, Record Collections, links, naming, routes, indexes, and constraints. Use when a business data model or metadata contract is the main task.
---

# dygo Entity Modeling

Model the business domain explicitly in metadata before you add custom code.

## Source Contracts

Read the relevant parts of:

- `docs/entity-metadata.md`
- `docs/app-model.md`
- `docs/nomenclature.md`
- `docs/metadata-authoring.md`

Use `schemas/entity.schema.json` for editor guidance. Treat Go validation as authoritative.

## Rules

- Use a singular, file-derived Entity key.
- Keep Entity identity app-scoped. Use `<app>/<entity>` where an app-qualified identity is required.
- Keep the user route slug separate from Entity identity. Make route slugs globally unique.
- Use Record `name` as the stable system or business identifier. Keep numeric `id` internal.
- Use a Record Collection for repeating parent-owned structure. Do not give collection rows routes, standalone permissions, links to normal Entities, or Hooks.
- Use Field-level `index` and `unique` only for single-Field shorthands.
- Use top-level indexes and constraints for composite or structured rules.
- Use structured checks. Do not add raw SQL to Entity metadata.
- Prefer metadata that a human can review without knowing storage internals.

## Check

Run the narrowest useful checks: `dygo entity validate`, `dygo entity show <app>/<entity>`, `dygo entity graph`, and `dygo route validate`.
