# AGENTS.md - Quick Reference for AI Coding Agents

## Commands
```bash
# Run single test: go test ./path/to/package -run TestName
go test ./internal/api -run TestCompletionHandler

# Before every commit (MANDATORY):
gofmt -w . && golangci-lint run && go mod tidy && go vet ./... && go test ./...
```

## Code Style
- **Imports**: Group stdlib, external, internal with blank lines. Use `goimports` for ordering.
- **Formatting**: Run `gofmt -w .` before commit. No manual formatting.
- **Types**: Use named struct fields (`&Config{Enabled: true}` not `&Config{true}`). Pointer receivers for methods that mutate.
- **Naming**: `camelCase` for unexported, `PascalCase` for exported. Acronyms all caps (`HTTPServer`, `apiKey`).
- **Error Handling**: ALWAYS handle errors with `fmt.Errorf("context: %w", err)`. Never ignore errors.
- **Context**: Pass `context.Context` as first param to all service methods. Use `c.UserContext()` in handlers.
- **Comments**: No comments unless required. Code should be self-documenting.

## Architecture
- Use `ResolveConfig()` to merge YAML + request overrides before processing (CRITICAL).
- Initialize format adapter singletons (`format_adapter.InitAdapters()`).
- Share circuit breakers across endpoints (created once in `main.go`).
- Buffer pools: `bufferpool.Get()` / `bufferpool.Put()` for streaming.
- Tests adjacent to impl: `foo.go` → `foo_test.go`, use table-driven tests.

See CLAUDE.md for full architecture details.
