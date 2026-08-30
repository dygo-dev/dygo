---
name: dygo-go-development
description: Write, refactor, or review Go code in the dygo framework or App SDK using dygo package boundaries and runtime conventions. Use for substantial Go implementation that is not primarily Cobra CLI work.
---

# dygo Go Development

Write direct Go that makes framework boundaries easy to see.

## Rules

- Follow the Go version and dependencies in `go.mod`.
- Keep `cmd/` entry points small. Put framework implementation in focused `internal/` packages.
- Keep the public App contract in `pkg/dygo` and avoid leaking internal types.
- Accept `context.Context` for blocking, database, request, and background operations.
- Wrap errors with a short operation description and `%w` when callers need the cause.
- Keep interfaces small and define them near the consumer when practical.
- Prefer explicit data flow and typed boundaries over reflection, global state, and generic maps.
- Preserve transaction ownership. Do not start hidden nested transactions.
- Bound goroutines and make cancellation and ownership clear.
- Use existing registries and shared contracts before you create a parallel mechanism.
- Add exported documentation where the API needs a stable contract.

## Check

Run `gofmt` on changed Go files. Use focused package tests first. Use vet, race checks, or the full suite when concurrency, public APIs, database state, release code, or a major final change justifies the cost.
