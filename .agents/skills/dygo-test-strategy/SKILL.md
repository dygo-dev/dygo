---
name: dygo-test-strategy
description: Select, write, or review proportionate tests and verification for dygo Go packages, PostgreSQL behavior, metadata, CLI contracts, Studio, integrations, and releases. Use when test scope or confidence strategy is the main decision.
---

# dygo Test Strategy

Test the meaningful invariant at the lowest useful boundary.

## Choose The Boundary

- Use a focused Go package test for pure logic and package contracts.
- Use PostgreSQL-backed tests for transactions, schema state, constraints, concurrency, and Record behavior.
- Use CLI tests for arguments, streams, plans, prompts, redaction, and exit behavior.
- Use metadata fixtures for parsing, validation, identity, and deterministic diagnostics.
- Use Studio unit tests for stores, queries, renderers, and component behavior.
- Use browser checks for interaction, navigation, accessibility, and layout acceptance.
- Use release smoke tests for archives, installers, bundled assets, and reported versions.

## Rules

- Test business or framework behavior, not exact implementation wording.
- Inspect existing coverage before adding tests. Add a case for a meaningful behavior or failure mode that is not adequately covered, not merely for a new function or file.
- Extend an existing case when it covers the same contract clearly. Test at multiple layers only when each protects a distinct failure mode.
- Prefer observable results over source patterns or incidental markup. Keep source and file assertions when generated output, ownership markers, wire formats, or bundled assets are the contract.
- Before removing a test, map its unique assertions and input cases to retained coverage, a stronger replacement, or a confirmed obsolete contract. Resolve fixture helpers before calling cases duplicates.
- Consolidate repeated setup with named cases when this stays clear. Preserve distinct failure modes and useful diagnostics; fewer lines do not necessarily mean fewer executions.
- Make assertions prove the named behavior: presence does not prove order, a preset mock decision does not prove authorization, and any error does not prove the intended rejection. Redaction checks must introduce sensitive input.
- Add a regression test when a defect can recur and the boundary is stable.
- Do not add tests that mirror a reversible, low-impact edit when existing validation proves the result.
- Do not chase an arbitrary coverage percentage.
- Keep database tests deterministic and isolate their state. SQL-construction tests do not replace PostgreSQL behavior checks; report skipped integration tests and their execution requirements.
- Include failure and rollback cases for high-risk writes.
- Include authorization cases for data-bearing APIs.
- Use race or concurrency checks when ownership, workers, claims, or goroutines changed.
- Run the narrowest relevant checks during development. Run broader checks before a major commit or release.
- After relevant and required checks pass, repeat or broaden them only for new changes, failures, or unresolved concerns.
