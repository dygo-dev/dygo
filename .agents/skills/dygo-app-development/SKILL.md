---
name: dygo-app-development
description: Build or extend a dygo Business App using the supported project layout, generators, metadata, SDK, and validation workflow. Use for general app work that is not primarily Entity modeling, access, hooks, Jobs, fixtures, or patches.
---

# dygo App Development

Build business behavior in an App and let dygo provide the platform foundation.

## Start

1. Find the project root from `dygo.yml`.
2. Read `docs/doctrine.md`, `docs/app-model.md`, and `docs/dir.md` when they exist in the checkout.
3. Inspect the target App and nearby examples before you choose a structure.
4. Confirm that the requested behavior is business-specific. Put reusable platform behavior in the framework instead.

## Rules

- Use the terms in `docs/nomenclature.md`.
- Prefer an App manifest, Entity metadata, access metadata, Hooks, Fixtures, Jobs, Schedules, and Patches over one-off infrastructure.
- Use `pkg/dygo` from App-owned Go code. Do not import framework `internal/` packages.
- Start with the smallest valid App shape. Add Pages, reports, or custom behavior only when metadata-driven Studio rendering is insufficient.
- Keep generated files predictable. Do not overwrite developer-owned Hook or Job logic.
- Treat server-side permissions and observable failure behavior as part of the feature.
- Do not implement behavior that current documentation marks as proposed or coming soon unless the task is to develop that framework capability.

## Check

Use focused commands such as `dygo app validate`, `dygo entity validate`, `dygo hook validate`, and `dygo doctor`. Use broader tests only when the change risk justifies them.
