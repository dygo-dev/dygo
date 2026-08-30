---
name: dygo-jobs-and-schedules
description: Build or review dygo durable Jobs, Job Executions, queues, workers, retries, idempotency, and Schedules. Use for background or recurring business work.
---

# dygo Jobs And Schedules

Use Jobs for durable work outside a user request. Use Schedules to create recurring Job Executions.

## Start

Read the relevant sections of `docs/jobs.md`, `docs/schedule.md`, `docs/queues.md`, and `docs/sdk.md`. Inspect the target App's Jobs and queue configuration.

## Rules

- Identify a Job by `<app>/<job>`. Do not use its label as identity.
- Keep the Job payload explicit, stable, and safe to persist.
- Use an idempotency key when duplicate execution can harm the business.
- Make retry behavior match the failure type. Do not retry permanent validation failures.
- Keep handlers observable through result data, errors, timing, and persisted Logs.
- Keep queue and concurrency choices explicit when ordering or resource limits matter.
- Remember that `dygo serve` does not process Jobs. Production needs a separate worker process.
- Treat file-defined, Studio-defined, and system-owned Schedules according to the documented source contract.
- Do not add a second background execution mechanism inside an App.

## Check

Validate registration and runner wiring. Use focused execution inspection. Confirm worker behavior when changes affect claims, retries, Schedules, or concurrency.
