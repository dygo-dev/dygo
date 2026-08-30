---
name: dygo-observability
description: Add or review dygo Logs, Activity, Audit Logs, health checks, diagnostics, metrics, traces, and operator-visible failure states. Use when runtime behavior must become inspectable or diagnosable.
---

# dygo Observability

Make important framework and business behavior visible without exposing sensitive data.

## Start

Read `docs/logs.md` and the relevant runtime document. Inspect existing Log helpers, health handlers, CLI diagnostics, Job Execution data, and Studio surfaces.

## Rules

- Use persisted Logs for operational diagnostics that must survive the request or process.
- Use Activity for human-friendly Record history.
- Reserve Audit Logs for compliance and security-grade history.
- Choose a canonical source and type. Keep messages concise and actionable.
- Include stable context such as App, Entity, Record, Job, execution, operation, or correlation identity when available.
- Put optional high-cardinality context in metadata until a real query workflow needs a first-class Field.
- Redact secrets, credentials, tokens, session data, and sensitive payloads.
- Make health checks bounded and meaningful. Do not make them perform destructive recovery.
- Avoid duplicate logging at every layer for the same failure.
- Give operators a clear next action when the system can provide one.

Verify that the event is emitted at the correct boundary and can be inspected through the intended CLI, API, or Studio path.
