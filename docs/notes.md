# Maintainer Notes

This page keeps repo-maintenance notes that are useful to review with the codebase. Framework behavior is documented in the focused docs linked from the [Documentation Index](index.md).

## Documentation Practice

- Keep framework documentation in `docs/`, not in a GitHub Wiki.
- Keep docs versioned with code and reviewed in pull requests.
- Prefer concise reference docs over planning transcripts.
- A future docs website can publish these files when the docs are stable enough for a public site.

## CLI And Directory Shape

[CLI](cli.md) and [Directory Shape](dir.md) are the source references for the command surface and filesystem layout.

The current conventions are:

- Centralize path constants and slash-target parsing.
- Use root `dygo.yml`, `config/`, `.dygo/`, and `db/schema.sql`.
- Use canonical Entity bundles at `apps/<app>/entities/<entity>/<entity>.entity.yml`.
- Store Entity fixtures at `apps/<app>/entities/<entity>/fixtures.yml`.
- Use `dygo db migrate`, `dygo db prune`, singular command groups, and `--yes` / `--dry-run` write safety.
- Keep `dygo upgrade` project-only; update binaries out of band through installers.
- Put scaffolding under `dygo generate` and alias `dygo g`.
- Keep hook generation under `dygo generate hook`; use `dygo hook` for inspection and wiring maintenance.
- Keep Entity action generation under `dygo generate action`; `dygo hook sync` also updates action runner wiring.
- Keep route validation filesystem-backed.
- Include route, fixture, hook, Job, Schedule, schema snapshot, config, secrets, database, Studio assets, and first-run setup checks in `dygo doctor`.

## Coming Soon

These are intentionally not part of the current public CLI contract:

- global `--json`
- smart shell completions
- report runtime
- custom page runtime
- Studio schedule UI
- Studio retry and cancel controls for Job Executions
- production secret providers such as KMS or Vault

Naming notes:

- `internal/studio/assets.go` is really Studio bundle source resolution, cache installation, and static handler wiring.
- `internal/fixtures/fixtures.go` may read better split into discovery, validation, and apply files.
