---
name: dygo-code-quality
description: Improve or review ordinary dygo code changes for simplicity, focused scope, canonical ownership, clear boundaries, and verifiable behavior. Use for normal implementation and refactoring quality; use dygo-thermonuclear-review only for an explicitly severe audit.
---

# dygo Code Quality

Deliver the required behavior with the least maintenance burden.

## Before Editing

- State material assumptions.
- Inspect the current implementation and nearby conventions.
- Define an observable success condition in a real Business App or operator flow.
- Identify the canonical package, metadata contract, SDK boundary, or Studio component that owns the behavior.

## Rules

- Keep every changed line tied to the task.
- Do not clean unrelated code or reformat unrelated files.
- Prefer direct code and standard-library operations over custom algorithms and speculative abstraction.
- Prefer existing framework and native platform capabilities before adding custom code or dependencies.
- Reduce duplicated rules, state, branches, queries, and abstractions. Do not compress readable code or omit required behavior to reduce line count.
- Add abstractions for demonstrated shared contracts, not hypothetical reuse. A small amount of duplication can be simpler than coupling unrelated behavior.
- Avoid unnecessary I/O, repeated queries, allocations, and unbounded work. Support performance claims with measurements appropriate to the affected path.
- For refactoring, identify the duplicated rule, state owner, branch, or concept that the change removes. Moving code into more files is not sufficient.
- Preserve error meaning. Treat only a confirmed not-found result as absence. Do not use a fallback to hide a persistence, permission, or decoding failure.
- Do not add a thin wrapper, flag, optional mode, or generic registry without a real repeated need.
- Keep business logic in Apps and reusable platform capability in the framework.
- Keep internals private until Business Apps need a stable public contract.
- Remove only dead code that the current change creates unless cleanup is in scope.
- Preserve reusable UI components and supported exports even without current callers unless their removal is explicitly requested.
- Add a concise TODO only when a real deferred improvement has a clear boundary and cannot be completed in the current scope.
- Verify behavior in proportion to risk.

If a solution becomes much larger than the behavior it provides, stop and simplify it before finalization.
