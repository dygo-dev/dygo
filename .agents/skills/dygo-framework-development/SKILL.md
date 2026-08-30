---
name: dygo-framework-development
description: Design or implement reusable capability in the dygo framework, Core App, Studio App, runtime, or public SDK. Use when changing dygo itself rather than building a Business App.
---

# dygo Framework Development

Build a reusable platform primitive. Keep business-specific behavior in Business Apps.

## Start

Read `AGENTS.md`, `docs/doctrine.md`, `docs/app-model.md`, `docs/sdk.md`, `docs/dir.md`, and the relevant subsystem document. Inspect nearby implementations and tests before you choose a new abstraction.

## Rules

- Keep implementation details under `internal/`.
- Expose stable App capability only through `pkg/dygo`.
- Any reusable capability used by Core must be available to Business Apps through a supported SDK contract when practical.
- Make bootstrap exceptions explicit, small, and documented in the code path.
- Prefer shared contracts and registries for naming, Field types, metadata, permissions, Hooks, queries, and API envelopes.
- Dogfood metadata-backed Records and framework primitives where metadata exists.
- Consider permissions, Logs, audit behavior, health, failure states, CLI support, documentation, and Studio visibility.
- Keep public APIs small. A public API is a compatibility promise.
- Prefer boring, explicit implementation over hidden magic.
- Do not turn roadmap notes into current contracts without updating implementation and documentation together.

Verify the changed boundary in proportion to risk. Before a major commit, run the broader framework checks that the affected subsystems require.
