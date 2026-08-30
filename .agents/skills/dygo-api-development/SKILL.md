---
name: dygo-api-development
description: Design, implement, or review dygo HTTP APIs and public App SDK contracts. Use for Record APIs, metadata and boot endpoints, query behavior, response envelopes, compatibility, and public Go interfaces.
---

# dygo API Development

Keep network APIs and the trusted App SDK distinct.

## Start

Read `docs/server.md`, `docs/records.md`, and `docs/sdk.md`. Inspect the current router, shared response types, query parser, permission engine, and public package before you design a new surface.

## Rules

- Use App-qualified Entity identity internally. Use documented route slugs only at the user-facing route boundary.
- Enforce authentication and Permissions before Record access.
- Keep response and error envelopes consistent across endpoints.
- Validate filters, sorts, pagination, paths, and payloads at the boundary.
- Preserve deterministic list ordering.
- Return useful client errors without exposing SQL, secrets, or protected data.
- Keep browser and Studio APIs compatible with the documented version contract.
- Put stable trusted-server capability in `pkg/dygo`; keep implementation under `internal/`.
- Do not expose a public SDK type only because an internal package needs it.
- Consider cancellation, timeouts, observability, and audit behavior for material operations.

Use focused handler and contract tests. Verify Permissions and error shapes for data-bearing endpoints.
