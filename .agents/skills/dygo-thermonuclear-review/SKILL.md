---
name: dygo-thermonuclear-review
description: Run an explicitly requested, unusually strict dygo maintainability audit that seeks structural simplification, removes accidental complexity, and challenges weak abstractions without changing behavior.
---

# dygo Thermonuclear Review

Use this skill only when the user asks for a thermonuclear, severe, exhaustive, or especially harsh code-quality review.

## Standard

Search for a structural move that deletes concepts, branches, modes, wrappers, duplication, or special cases. Do not stop at local style comments.

Challenge:

- framework capability hidden in a private Core-only path;
- Business App logic inside reusable framework packages;
- duplicated metadata decoding, naming, permission, query, or API contracts;
- files and components that became too large to reason about;
- scattered conditionals that indicate a missing model or policy;
- pass-through abstractions and generic mechanisms that hide simple data flow;
- optionality, casts, maps, or fallbacks that conceal an invariant;
- non-atomic related updates and avoidable sequential orchestration;
- one-off Studio patterns outside the design system;
- public SDK surface that has not earned a compatibility promise.

## Boundaries

- Keep an audit read-only unless implementation is authorized. Refactoring must preserve behavior; change behavior only within an authorized feature or fix, and explain the difference.
- For parallel reviews, assign distinct subsystems or concerns and require code evidence. Reconcile overlapping findings before applying changes.
- Do not reward novelty. Prefer direct and boring structure.
- Support every finding with concrete code evidence and maintenance impact.
- Distinguish a defect from an ambitious redesign opportunity.
- Do not require broad rewrites when a smaller structural correction removes the same complexity.
