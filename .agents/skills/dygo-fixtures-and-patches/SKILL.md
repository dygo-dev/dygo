---
name: dygo-fixtures-and-patches
description: Create or review dygo Fixtures and explicit Patches for reference data, setup Records, backfills, renames, and unsafe lifecycle transitions. Use when metadata sync alone cannot express the required data change.
---

# dygo Fixtures And Patches

Choose the mechanism by intent.

- Use a Fixture for repeatable reference, setup, or demo Records.
- Use a Patch for a one-time ordered lifecycle transition.
- Use metadata sync for additive schema changes that metadata can infer safely.

## Source Contracts

Read `docs/fixtures.md`, `docs/patches.md`, and the database lifecycle in `docs/database.md`.

## Rules

- Keep Fixtures app-owned and Entity-local.
- Use stable match fields and explicit dependencies.
- Do not use Fixtures as an uncontrolled production data migration system.
- Prefer structured Patch operations.
- Use the SQL escape hatch only when structured operations cannot express a safe transition.
- Place a Patch in the correct pre-sync or post-sync phase.
- Make Patch execution transactional and ledgered according to the runtime contract.
- Never hide destructive cleanup inside additive migration. Use explicit prune or a reviewed Patch.
- Preview writes before application when a dry-run path exists.

## Check

Use fixture validation and dry-run plans. For a Patch, verify the intended before and after state, failure rollback, and repeat-run ledger behavior.
