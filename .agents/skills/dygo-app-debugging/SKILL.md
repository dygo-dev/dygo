---
name: dygo-app-debugging
description: Diagnose a dygo Business App that does not validate, boot, route, render, authorize, migrate, or execute background work correctly. Use for evidence-first investigation rather than feature implementation.
---

# dygo App Debugging

Find the failing boundary before you change code.

## Workflow

1. Reproduce the smallest observable failure.
2. Run `dygo doctor` or the narrow validator for that boundary.
3. Inspect resolved configuration, metadata identity, routes, generated runner wiring, and environment selection.
4. Inspect persisted Logs, Job Executions, schema state, and Records when runtime state is involved.
5. Trace the failure to App metadata, App code, public SDK behavior, framework runtime, Studio, or deployment configuration.
6. Report the cause and evidence. Do not implement a fix unless the user asked for one.

## Rules

- Redact database URLs, secrets, session values, and protected business data.
- Distinguish source metadata from persisted Core metadata.
- Distinguish `dygo serve` from `dygo worker` behavior.
- Distinguish current behavior from proposed documentation.
- Use `dygo entity show`, `entity graph`, `route resolve`, `access show`, `hook list`, and Job inspection when relevant.
- Do not reset or prune a database as a diagnostic shortcut.

Use focused verification after a fix. Expand checks only when the cause crosses subsystem boundaries.
