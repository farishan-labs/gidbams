## Why

`gidbams mcp` currently exposes read-only tools (list/get/validate/search/query/build/stats/doctor/schema/version). Agents that need to add or fix records must shell out to the CLI or edit files directly, which defeats the purpose of an agent-facing MCP server and bypasses validation/derived-DB refresh. Adding write tools makes the MCP server a complete CRUD + validation + build interface for agents.

## What Changes

- Add MCP tools `create_record`, `update_record`, and `delete_record` mirroring the existing CLI CRUD commands.
- Each write tool validates the record against the collection schema before writing and returns validation errors otherwise.
- `create_record`/`update_record` refresh the derived SQLite database (or document that `build` must be called) — decision in design.
- `delete_record` removes the record file.
- All writes go through the existing store layer (atomic writes + file locking) to stay consistent with CLI behavior.

## Capabilities

### New Capabilities
- `mcp-write-tools`: create/update/delete records over MCP with schema validation.

### Modified Capabilities
(none)

## Impact

- Affected files: `internal/mcp/server.go` (toolDefs + callTool switch), possibly `internal/store/store.go` (reuse existing Create/Update/Delete).
- Affected APIs: MCP tool list gains 3 tools; tool result format unchanged (text JSON).
- Backward compatible: additive only; existing read tools unchanged.
- Risk: medium. Must ensure validation + derived DB consistency and avoid partial writes; reuse store layer to inherit locking.