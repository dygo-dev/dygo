---
name: dygo-metadata-engineering
description: Develop or review dygo metadata loading, YAML decoding, JSON Schemas, validation, naming, Field types, registries, routes, and persisted metadata contracts. Use for framework-level metadata behavior rather than authoring one Business App Entity.
---

# dygo Metadata Engineering

Keep metadata explicit, readable, and deterministic.

## Start

Read `docs/metadata-authoring.md`, `docs/entity-metadata.md`, `docs/app-manifest.md`, and the affected metadata document. Inspect shared packages such as `internal/yamlmeta`, Entity schema and catalog packages, naming, reserved words, routes, and Core metadata Records.

## Rules

- Keep YAML files as source of truth and persisted Core metadata as the runtime registry.
- Use one decoding and validation contract for equivalent metadata.
- Reject unknown, conflicting, reserved, or ambiguous identities early.
- Keep App-qualified internal identity separate from user-facing routes and storage names.
- Centralize Field-type behavior, naming, reserved names, and route rules in shared registries.
- Keep JSON Schemas aligned with current accepted metadata, but treat runtime Go validation as authoritative.
- Preserve deterministic ordering in validation, plans, and persisted metadata.
- Return diagnostics with the source file and exact metadata path when possible.
- Do not add required metadata only to serve display or storage derivations that dygo can compute.
- Update documentation, schema files, generators, runtime readers, and validation together when a public metadata contract changes.

Use focused contract and round-trip tests for changed metadata behavior.
