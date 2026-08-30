---
name: dygo-pages-and-studio
description: Define dygo App Pages and metadata-driven Studio experiences for Business Apps. Use when an App needs a Page, navigation entry, renderer choice, or Studio presentation beyond default Entity routes.
---

# dygo Pages And Studio

Use metadata to select a framework-owned renderer. Do not place UI implementation inside Page metadata.

## Start

Read `docs/pages.md`, `docs/studio.md`, and the Page-related parts of `docs/app-model.md` and `docs/dir.md`. Confirm which Page features exist now.

## Rules

- Keep the Page bundle app-owned under the documented `pages/` path.
- Use a stable Page key and a supported renderer.
- Let Studio own rendering, layout, permissions, loading, errors, and shared interaction patterns.
- Prefer normal Entity and Record surfaces when they solve the task.
- Add a Page only when it represents a useful business Space or cross-Entity entry point.
- Keep navigation labels and actions in dygo vocabulary.
- Do not encode arbitrary frontend code, SQL, or unvalidated behavior in metadata.
- Do not implement proposed custom UI contracts as if they are supported.

## Check

Validate Page metadata through the current project validation path. Confirm that boot data, navigation, route resolution, permissions, and renderer registration agree.
