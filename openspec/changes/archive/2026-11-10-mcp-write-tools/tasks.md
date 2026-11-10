## 1. Inspect existing store/CLI behavior

- [x] 1.1 Read `internal/store/store.go` Create/Update/Delete signatures and validation usage
- [x] 1.2 Read CLI create/update/delete handlers in `internal/cli/cli.go` to mirror validation + error handling

## 2. Implement MCP tools

- [x] 2.1 Add `create_record`, `update_record`, `delete_record` entries to `toolDefs()` in `internal/mcp/server.go`
- [x] 2.2 Add cases to `callTool` switch for the three new tools
- [x] 2.3 Implement create: validate then store.Create; return success/error
- [x] 2.4 Implement update: validate then store.Update; return success/error (not-found handling)
- [x] 2.5 Implement delete: store.Delete; return success/error (not-found handling)

## 3. Tests and quality

- [x] 3.1 Add integration tests in `internal/mcp` covering create/update/delete success and validation/not-found errors
- [x] 3.2 Run `gofmt -w .`, `go vet ./...`, `go test ./internal/mcp/...`
- [x] 3.3 Update README MCP tools list and openspec/specs/mcp-server spec