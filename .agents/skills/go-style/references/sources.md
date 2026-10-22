# Go style sources and boundaries

Consult the relevant section when a rule is disputed, a Go language version matters, or a new rule is being proposed. These sources inform the skill; they do not override a project's correctness or compatibility requirements. Checked against the linked upstream pages on 2026-10-21.

## Core guidance

| Source | Use and limits |
| --- | --- |
| [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) | Practical Go Wiki guidance on formatting, names, interfaces, errors, contexts, goroutine lifetimes, and more. It supplements Effective Go; treat advice in its actual context. |
| [Google Go Style overview](https://google.github.io/styleguide/go/index) and [Guide](https://google.github.io/styleguide/go/guide) | The Guide is canonical and normative **for Google Go code**; it orders clarity, simplicity, concision, maintainability, and consistency. Its lack of a fixed line length is a deliberate style decision. |
| [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions) | Normative but not canonical for Google reviewers; consult for a specific decision rather than treating every Google rule as a universal Go requirement. |
| [Google Go Style Best Practices](https://google.github.io/styleguide/go/best-practices) | Neither normative nor canonical; context-dependent examples and trade-offs. |
| [Go Test Comments](https://go.dev/wiki/TestComments) | Test failure diagnostics and test structure; table-driven tests fit cases with shared checking logic, while distinct logic can warrant separate tests. |
| [Go Doc Comments](https://go.dev/doc/comment) | Go documentation comments for packages, exported declarations, and examples. |
| [Effective Go](https://go.dev/doc/effective_go) | Foundational idioms; its own introduction notes that the text has not kept pace with features such as modules and generics. |

## Versions, API behavior, and tools

- [Go specification](https://go.dev/ref/spec), [release notes](https://go.dev/doc/devel/release), and [go.mod reference](https://go.dev/doc/modules/gomod-ref): check the language version and relevant build constraints before relying on version-sensitive behavior.
- [Go 1.22 loop variable change](https://go.dev/wiki/LoopvarExperiment): the new per-iteration semantics depend on the module's declared Go language version, not simply the installed compiler.
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors): wrapping with `%w` exposes the underlying error to callers via `errors.Is` and `errors.As`, which may become part of the API contract.
- [gofmt](https://pkg.go.dev/cmd/gofmt) is the standard formatting tool. [goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) also adjusts imports; use it when the project already relies on it rather than adding a dependency for style alone.
