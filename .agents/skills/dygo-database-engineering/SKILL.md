---
name: dygo-database-engineering
description: Develop or review dygo PostgreSQL connectivity, schema planning, additive migration, prune, metadata persistence, Records, transactions, and schema snapshots. Use for framework database behavior and high-risk data lifecycle changes.
---

# dygo Database Engineering

Make database changes reviewable, transactional, and safe for business data.

## Start

Read `docs/database.md`, `docs/patches.md`, and the relevant Record or metadata contract. Inspect the planner, executor, naming policy, and current schema tests before you edit behavior.

## Rules

- Keep `dygo db migrate` additive and safe.
- Use explicit prune for metadata-orphaned tables, columns, indexes, and constraints.
- Keep destructive targets concrete and visible in the plan.
- Use metadata as source of truth for dygo-managed schema.
- Preserve deterministic plans and schema snapshots.
- Keep related state changes atomic when partial application would violate an invariant.
- Use parameterized SQL. Do not interpolate business values or identifiers without approved quoting rules.
- Respect transaction ownership and PostgreSQL locking behavior.
- Make concurrent claims and uniqueness rules enforceable in the database when correctness depends on them.
- Keep Core metadata persistence on the same framework contracts where practical.
- Return actionable database errors without exposing credentials.

## Check

Use focused planner and database tests. Verify both the intended schema result and meaningful data preservation. Development migrations require elevated execution because the database socket is outside the normal sandbox.
