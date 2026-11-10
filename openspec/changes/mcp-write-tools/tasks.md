## 1. Inspect existing store/CLI behavior

- [ ] 1.1 Read `internal/store/store.go` Create/Update/Delete signatures and validation usage
- [ ] 1.2 Read CLI create/update/delete handlers in `internal/cli/cli.go` to mirror validation + error handling

## 2. Implement MCP tools

- [ ] 2.1 Add `create_record`, `update_record`, `delete_record` entries to `toolDefs()` in `internal/mcp/server.go`
- [ ] 2.2 Add cases to `callTool` switch for the three new tools
- [ ] 2.3 Implement create: validate then store.Create; return success/error
- [ ] 2.4 Implement update: validate then store.Update; return success/error (not-found handling)
- [ ] 2.5 Implement delete: store.Delete; return success/error (not-found handling)

## 3. Tests and quality

- [ ] 3.1 Add integration tests in `internal/mcp` covering create/update/delete success and validation/not-found errors
- [ ] 3.2 Run `gofmt -w .`, `go vet ./...`, `go test ./internal/mcp/...`
- [ ] 3.3 Update README MCP tools list and openspec/specs/mcp-server spec