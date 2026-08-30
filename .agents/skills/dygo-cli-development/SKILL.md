---
name: dygo-cli-development
description: Design, implement, or review the dygo framework CLI in Go with Cobra. Use for commands, flags, help, prompts, plans, output contracts, completion, and project-aware CLI behavior. Do not use for operating a deployed business through the reserved dygo-cli operator skill.
---

# dygo CLI Development

Develop the CLI implementation. This skill does not perform business operations on a deployment.

## Structure

- Keep `cmd/dygo/main.go` small and call the CLI runner.
- Put command implementation under `internal/cli`.
- Construct the root with injected context, stdin, stdout, and stderr.
- Keep `SilenceUsage` and `SilenceErrors` enabled.
- Use `RunE` and explicit `Args` validation.
- Use project-root discovery for commands that read project files.
- Keep command logic small. Put reusable behavior in the owning framework package.

## Command Contract

- Follow the resource-first command surface in `docs/cli.md`.
- Send stable normal output to stdout. Send prompts, warnings, and diagnostics to stderr.
- Keep output plain, concise, and usable by people and agents.
- Redact secrets and credentials.
- For material writes, show the target and plan before the prompt.
- Support `--dry-run` when planning is meaningful and `--yes` when non-interactive use is safe.
- Make destructive commands explicit. Never hide destructive work inside an additive command.
- Do not add Viper until dygo needs shared flag, environment, file, and default precedence.
- Keep completion fast, local, and read-only.

## Check

Use focused `internal/cli` tests and inspect help for changed command trees. Test injected streams, invalid arguments, nested project discovery, and safety behavior when relevant.
