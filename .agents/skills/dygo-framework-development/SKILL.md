---
name: dygo-framework-development
description: Design or implement reusable capability in the dygo framework, Core App, Studio App, runtime, or public SDK. Use when changing dygo itself rather than building a Business App.
---

# dygo Framework Development

Build a reusable platform primitive. Keep business-specific behavior in Business Apps.

## Start

Apply `AGENTS.md` and read the affected subsystem contract. Use `docs/doctrine.md` and `docs/app-model.md` for ownership decisions, `docs/sdk.md` for public contracts, and `docs/dir.md` for file placement. Inspect nearby implementations and tests before choosing a new abstraction.

## Rules

- Keep implementation details under `internal/`.
- Expose stable App capability only through `pkg/dygo`.
- Expose reusable App capability used by Core through a supported SDK contract. Keep storage and bootstrap privileges internal.
- Trace one Core caller and one Business App caller through the shared primitive. A public interface alone does not prove dogfooding.
- For a direct SQL or private Core path, identify why the shared primitive is insufficient. Bootstrap metadata writes and worker claim SQL can require separate paths. Ordinary Record changes need an explicit reason to bypass shared behavior.
- Keep each exception small. Document its scope and its permission, Hook, and Activity behavior at the call site.
- Prefer shared contracts and registries for naming, Field types, metadata, permissions, Hooks, queries, and API envelopes.
- Dogfood metadata-backed Records and framework primitives where metadata exists. Extend the canonical owner before adding a parallel service.
- Trace transaction ownership and access mode through Actions, Hooks, and Jobs. Services supplied together do not necessarily share a transaction.
- When a service claims atomic behavior across Records and Jobs, verify that both use the same transaction in each supported entry point.
- Consider permissions, Logs, audit behavior, health, failure states, CLI support, documentation, and Studio visibility.
- Keep public APIs small. A public API is a compatibility promise.
- Prefer boring, explicit implementation over hidden magic.
- Do not turn roadmap notes into current contracts without updating implementation and documentation together.

Verify the changed boundary in proportion to risk. Before a major commit, run the broader framework checks that the affected subsystems require.
