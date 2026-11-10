## Context

See proposal.md — Why. The MCP server (`internal/mcp/server.go`) already wires store + validate for read tools. The store layer (`internal/store/store.go`) exposes `Create`, `Update`, `Delete` (used by CLI). Validation is `validate.ValidateCollection`/`ValidateRecord`. The MCP `callTool` switch dispatches by name; `toolDefs()` lists schemas.

## Goals / Non-Goals

**Goals:**
- Expose create/update/delete over MCP with schema validation, reusing store + validate.
- Keep tool result format (text JSON) and error reporting consistent with existing tools.

**Non-Goals:**
- No new permission/auth model for MCP (stdio, trusted local agent).
- No automatic derived-DB rebuild on every write (decide below).
- No bulk/batch write tools in this change.

## Decisions

- **Reuse store.Create/Update/Delete** rather than reimplementing file IO, so atomic writes + locking are inherited. Alternative: direct file writes in mcp package — rejected (duplication, risk of drift).
- **Validate before write**: load collection def, parse fields, run the same validation path as CLI (`ValidateCollection` or a single-record validator if available) before calling store.Create/Update. On failure, return the validation error and do not write.
- **Derived DB refresh**: do NOT rebuild automatically on each write (could be slow on large repos and surprising). Instead, document that callers run the `build` tool after writes; optionally add a `rebuild` boolean param to create/update later. Alternative: auto-build — rejected for performance/predictability. The `build` MCP tool already exists for explicit refresh.
- **Tool schemas**: `create_record` requires `collection`, `id`, `fields` (object). `update_record` same. `delete_record` requires `collection`, `id`. Return `{"status":"ok"}` on success (consistent with `build`).
- **Error mapping**: not-found → error with message; validation failure → error listing issues; unknown collection → error.

## Risks / Trade-offs

- [Store API signatures differ from MCP needs] → Inspect store.go during apply; adapt argument passing. If store expects a `*record.Record`, build it from fields.
- [Validation single-record vs collection] → Reuse whatever the CLI create/update path uses; check cli.go during apply to mirror exactly.
- [Partial writes on crash] → Inherited from store's atomic write; acceptable.

## Migration Plan

1. Inspect store.Create/Update/Delete and CLI create/update to mirror validation.
2. Add tool defs + switch cases.
3. Add integration tests in `internal/mcp` (mcp_tool_test.go already exists).
4. Run `go test ./internal/mcp/...`, `go vet`, `gofmt`.
5. Rollback: revert commit; read tools unaffected.