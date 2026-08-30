---
name: dygo-record-hooks
description: Implement or review dygo Record lifecycle Hooks in Business Apps or the Hook runtime. Use for validation, transactional Record reactions, and lifecycle side effects tied to Record changes.
---

# dygo Record Hooks

Keep Hook behavior small, transactional, and explicit.

## Source Contracts

Read `docs/record-hooks.md` and the Hook sections in `docs/sdk.md`. Inspect generated Hook files and the current registry contract before you edit code.

## Rules

- Put Entity Hook code in the owning Entity bundle.
- Register through the supported `pkg/dygo` contract.
- Choose the narrowest lifecycle event that matches the behavior.
- Use validation events for business validation. Return actionable errors.
- Remember that Hooks run inside the current Record transaction.
- Keep slow, retryable, or external work out of the transaction. Enqueue a durable Job instead.
- Use `hook.Records` for transactional Record access.
- Account for the documented rule that Hook Record writes run framework Hooks but do not re-enter App Hooks.
- Add useful persisted Logs for failures that operators must inspect.
- Do not import framework `internal/` packages from Business Apps.

## Check

Use `dygo hook validate` and focused tests for the changed event or registry. Confirm rollback behavior when failure must keep related Record changes atomic.
