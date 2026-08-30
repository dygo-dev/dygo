---
name: dygo-app-review
description: Review a dygo Business App for framework conventions, model quality, access control, SDK boundaries, observability, and safe lifecycle behavior. Use for App audits and pre-merge reviews, not ordinary implementation.
---

# dygo App Review

Review the App as business software, not only as source code.

## Review Order

1. Confirm App ownership and directory shape.
2. Review Entity identity, Fields, Collections, names, routes, indexes, and constraints.
3. Review roles and server-side Permissions.
4. Review Hook transaction boundaries and Job durability.
5. Review Fixtures, Patches, and migration safety.
6. Review SDK usage and reject imports from framework `internal/` packages.
7. Review Logs, failure states, and operator visibility.
8. Review Studio consistency and avoid unnecessary custom surfaces.

## Finding Standard

- Report only behavior with concrete evidence and impact.
- Separate defects from optional improvements.
- Point to the canonical dygo contract that the App violates.
- Prefer the smallest correction that restores the framework convention.
- Do not demand speculative abstractions, universal test coverage, or unsupported future features.
- Treat missing access enforcement, unsafe data transitions, and silent durable-work failures as high-risk findings.

Use validation and focused tests when needed to confirm a finding. Do not mutate the App during a read-only review.
