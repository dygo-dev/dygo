---
name: dygo-documentation
description: Write or update dygo instructions, reference documentation, task descriptions, CLI help, metadata examples, or architecture notes. Use when documentation quality and contract accuracy are central to the task.
---

# dygo Documentation

Write accurate technical information in ASD-STE100 style.

## Rules

- Use short sentences and active voice.
- Give one instruction per sentence when practical.
- Use the exact terms from `docs/nomenclature.md`.
- Call the main UI Studio. Use Entity, Record, Field, App, Hook, Fixture, Patch, Job, and Schedule with their defined meanings.
- Separate current behavior from proposed behavior and coming-soon behavior.
- For SDK or runtime changes, check adjacent guides for stale claims about access modes, transaction scope, and available services. Verify examples against the actual service wiring, not only the public interface.
- When code and documentation disagree during a review, report the conflict. Do not silently select the more permissive contract.
- Verify commands, flags, paths, metadata keys, and examples against current code.
- Put source-of-truth details in the canonical document and link to them from related documents.
- Do not copy the same contract into many files.
- Use realistic examples that match the current schema and CLI.
- State safety requirements before destructive or privileged instructions.
- Do not describe framework internals as supported Business App APIs.

## Check

Read the completed text as a new framework user. Confirm that each procedure has a clear outcome, prerequisites, safe command order, and no unsupported promise.
