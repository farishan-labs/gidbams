## 1. Add resolution helper

- [ ] 1.1 Add `EnvDB()` / `EnvToken()` helpers (or `EnvOr`) to `internal/config` that prefer `GIDBAMS_*` and fall back to `GIT_DBMS_*`
- [ ] 1.2 Add unit tests for precedence and fallback

## 2. Replace call sites

- [ ] 2.1 Update `internal/cli/cli.go` (build/query/update/delete/serve paths) to use the helper
- [ ] 2.2 Update `internal/mcp/server.go` (query/build/token paths) to use the helper
- [ ] 2.3 Update `internal/httpserver/server.go` to use the helper

## 3. Docs and cleanup

- [ ] 3.1 Update help text/examples to reference `GIDBAMS_*`
- [ ] 3.2 Add deprecation note for `GIT_DBMS_*` in README/CHANGELOG
- [ ] 3.3 Run `gofmt -w .`, `go vet ./...`, `go test ./...`, `go build ./...`