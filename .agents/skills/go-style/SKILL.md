---
name: go-style
description: >-
  Use when writing, changing, debugging, refactoring, or testing Go code or Go examples.
  Apply idiomatic Go style during authorized work using project conventions, Go Code Review
  Comments, and Google Go Style guidance. Also supports explicit Go style assessments;
  merely reading Go files does not authorize edits.
license: MIT
metadata:
  version: 0.1.0
---

# Go Style

Apply Go conventions while doing the requested work. This is guidance for implementation, not a separate review or scorecard. Do not expand an authorized Go change into repository-wide cleanup.

## When to apply

Use for new or changed `.go` files, tests, public Go APIs, and Go code examples. For an explicit Go style review, identify concrete findings without editing unless asked. Reading or explaining Go code alone does not authorize edits.

## Decide what governs

1. Check the task's behavior and compatibility requirements, nearby code, project instructions and tools, and the applicable Go language version (`go.mod` and build constraints where present). Correctness and existing public contracts take priority over style preferences.
2. Use [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and related Go documentation as the default source of idioms. Use the [Google Go Style Guide](https://google.github.io/styleguide/go/guide) for clarity, simplicity, concision, and maintainability. Its [Decisions](https://google.github.io/styleguide/go/decisions) are normative within Google's codebase, and its [Best Practices](https://google.github.io/styleguide/go/best-practices) are situational. Apply Google-specific prescriptions to another project only when they fit its conventions or it has adopted them.
3. When a point is disputed, version-sensitive, or materially affects an API, read the relevant entry in [sources and boundaries](references/sources.md) and check the current upstream text. Explain a conflict between project conventions, correctness, and external guidance rather than silently changing the contract.

## Write Go code

- Run `gofmt` on Go files you change; manage imports with the project's existing tooling. Do not format unrelated files. Use short, clear package names and Go `MixedCaps` identifiers; choose local names to fit their scope, and avoid repeating the package name at call sites.
- Prefer clear control flow, useful zero values, and the standard library where appropriate. Add an interface, generic type, helper, or layer when an actual use or boundary justifies it; do not create abstraction solely to satisfy a style checklist.
- Handle returned errors and add useful context at a boundary. Wrap with `%w` only when callers may rely on the underlying error as part of the API; otherwise avoid exposing an implementation detail. Close owned resources, including response bodies and files, and account for close errors when they affect the result. Do not panic for ordinary failures or log the same error at every layer.
- Propagate an existing `context.Context` across request or cancellation boundaries; pass it first when a function needs it. Do not store it in a struct or add it to every function by default. Make goroutine ownership, cancellation, and exit conditions clear.
- Write helpful doc comments for exported declarations and explain non-obvious reasons in other comments. Choose tests for changed behavior and plausible regressions; use table-driven tests when cases share testing logic, and make failures identify inputs and results. Do not add tests that only mirror a formatting change.
- Check version-dependent assumptions before applying older advice (for example, loop variable capture semantics depend on the module's Go language version, not simply the installed compiler). Preserve observable distinctions such as `nil` versus empty slices at JSON boundaries. Edit a generator's source instead of hand-editing generated code.

## Finish the task

Keep changes within the authorized scope. Format changed Go files and run the project's relevant tests, vet, or lint checks where feasible; consider whether a command may download dependencies or have other side effects. State what ran and what remains unverified. Report style trade-offs only when they matter to the result; do not produce a separate style report for ordinary coding work.
