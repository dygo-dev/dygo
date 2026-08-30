---
name: dygo-security-engineering
description: Design, implement, or review security-sensitive dygo behavior across auth, sessions, Permissions, secrets, APIs, database writes, files, Jobs, and Studio. Use when security boundaries are a primary concern.
---

# dygo Security Engineering

Protect business data at the server boundary and use secure defaults.

## Review Areas

- identity, session creation, expiry, revocation, and cookie settings;
- Permission checks and Administrator boundaries;
- secret encryption, redaction, environment selection, and key rotation;
- input validation, query construction, routes, and API errors;
- destructive database and CLI operations;
- Job payloads, Logs, files, and audit data;
- Studio exposure of protected metadata and Records.

## Rules

- Default to deny.
- Do not rely on UI hiding for enforcement.
- Keep secrets out of stdout, Logs, errors, fixtures, and committed plaintext.
- Use parameterized queries and canonical identifier validation.
- Make privileged and destructive targets explicit before execution.
- Preserve tenant or actor context when the runtime contract requires it.
- Record useful security events without storing sensitive payloads.
- Do not invent cryptographic protocols. Use the repository's established libraries and formats.
- Treat public SDK and HTTP surfaces as compatibility and trust boundaries.

Use focused adversarial checks for the changed boundary. Report evidence and impact. Do not expand a normal review into a security audit unless the task calls for it.
