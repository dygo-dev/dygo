---
name: dygo-studio-development
description: Implement or review dygo Studio behavior in Vue, TypeScript, Pinia, TanStack Query, Vue Router, Reka UI, and framework renderers. Use for Studio application logic, navigation, state, API integration, and shared components.
---

# dygo Studio Development

Treat Studio as the first-party product UI, not as a temporary admin panel.

## Start

Read the affected UI contract and relevant sections of `docs/studio.md`; use `docs/app-model.md` when App ownership is involved. Inspect nearby feature, renderer, shell, store, query, and design-system patterns before adding a new one.

## Rules

- Let Business Apps provide metadata and behavior. Let Studio render supported contracts consistently.
- Prefer shared renderers and design components over feature-local copies.
- Preserve reusable UI components and exports even when they have no current consumers. Remove them only when the user authorizes their removal.
- Keep server state in TanStack Query and cross-feature client state in Pinia only when it has a clear owner.
- Give applied route state, draft filter input, persisted preferences, and server results distinct owners. Do not add another synchronized copy of the same state.
- When a renderer mixes route synchronization, data fetching, actions, import/export, and presentation, separate the affected responsibility with a small contract. Avoid a mechanical file split or a new generic controller.
- Keep route identity aligned with the server route registry and boot payload.
- Enforce permissions on the server. Use client checks only for presentation and guidance.
- Handle loading, empty, error, forbidden, and retry states.
- Use Reka UI primitives for accessible complex interactions where available.
- Use Lucide icons through the shared conventions.
- Preserve keyboard access, focus, responsive behavior, and reduced-motion preferences.
- Do not create a one-off UI pattern when Studio needs a reusable primitive.

Use focused TypeScript or component checks. Exercise the actual flow in a browser when interaction or layout is central to the change.
