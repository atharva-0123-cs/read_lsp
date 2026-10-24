# Repository Guidelines

## Project Structure & Module Organization

This repository is a small Go Language Server Protocol server. The entry point
is `main.go`, which reads LSP messages from stdin, dispatches requests, and
writes JSON-RPC responses. Core packages are:

- `rpc/`: message framing, encoding, decoding, and scanner splitting.
- `lsp/`: LSP request, response, notification, diagnostic, completion, hover,
  definition, references, document symbol, folding range, and code action types.
- `analysis/`: document state and Markdown language behavior used by LSP handlers,
  including navigation and folding features.
- `diagnostics/`: Markdown document model, parser, and syntax diagnostics.
- `markdown_lsp.lua`: local Neovim helper script for manual LSP testing.
- `DOCS.md`: architecture and internals for the Markdown LSP.
- `log.txt`: runtime log output; do not rely on it as source data.

Tests currently live next to package code using Go's standard `*_test.go`
convention, for example `rpc/rpc_test.go`.

## Build, Test, and Development Commands

- `make build`: builds `main.go` using the repository Makefile.
- `go build ./...`: verifies all packages compile.
- `go test ./...`: runs the full test suite.
- `go test ./rpc`: runs only RPC package tests.
- `gofmt -w <files>`: formats edited Go files before committing.

For manual editor testing, build the server first, then use `markdown_lsp.lua`
from Neovim.

## Coding Style & Naming Conventions

Use standard Go formatting and idioms. Keep indentation and alignment controlled
by `gofmt`; do not hand-align code. Package names should stay short and
lowercase (`rpc`, `lsp`, `analysis`). Exported types and functions should use
clear PascalCase names, while unexported helpers should use camelCase.

Prefer small request-specific handlers and typed structs over unstructured maps
for LSP payloads. Keep JSON field tags consistent with the protocol shape.

## Testing Guidelines

Use Go's built-in `testing` package. Name test functions `TestXxx` and place
them near the code under test. Add table-driven tests for multiple LSP methods,
message formats, or edge cases.

Before submitting changes, run `go test ./...`. For protocol changes, include
tests for successful messages and malformed input where practical.

## Commit & Pull Request Guidelines

Recent commits use short, imperative summaries such as `remove logs` and
`emoji completion`. Keep commit subjects concise and focused on one change.

Pull requests should include a brief description, affected package or LSP
behavior, and test command run. Link issues when available. Include manual
Neovim testing notes when changing initialization, completion, hover,
definition, references, document symbols, folding ranges, diagnostics, or code
actions.

## Agent-Specific Instructions

Keep edits scoped to the requested behavior. Do not reformat unrelated files or
delete user-generated logs unless explicitly asked. When modifying protocol
types, verify callers in `main.go`, `analysis/`, and `lsp/` remain consistent.
Update `DOCS.md` when changing Markdown parsing, JSON-RPC, or LSP design.
