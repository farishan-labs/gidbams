## Context

See proposal.md — Why. Env var lookups are scattered: `os.Getenv("GIT_DBMS_DB")` appears in `internal/cli/cli.go` (4 sites: build, query, update, delete paths), `internal/mcp/server.go` (3 sites: query, build, and token), and `internal/httpserver/server.go` (1 site). `GIT_DBMS_TOKEN` is read once in `internal/cli/cli.go` for `serveCmd`. There is no central helper.

## Goals / Non-Goals

**Goals:**
- Single source of truth for env var resolution with precedence (new over legacy).
- Backward compatible for one release.

**Non-Goals:**
- No changes to the on-disk names `gitdb.yaml` / `.gitdb/` (those stay).
- No changes to config file format.
- No telemetry/warnings about deprecated usage (could be added later).

## Decisions

- **Add a small helper** `internal/config.EnvOr(newName, legacyName string) string` (or in a new `internal/env` package) that returns `os.Getenv(newName)` if non-empty else `os.Getenv(legacyName)`. Centralizes precedence and makes future removal a one-line change. Alternative: inline `if v := os.Getenv("GIDBAMS_DB"); v != "" { ... } else { os.Getenv("GIT_DBMS_DB") }` at each site — rejected because 8 sites and easy to drift.
- **New name takes precedence** when both are set, per spec.
- **Help text and examples** use only `GIDBAMS_*`; legacy names mentioned in a deprecation note in README/CHANGELOG.
- **Removal plan**: in the next minor release, drop the legacy fallback and the helper's legacy branch; document as breaking.

## Risks / Trade-offs

- [A script sets both names to different values expecting legacy to win] → Mitigation: spec defines new-wins; document in CHANGELOG.
- [Helper location choice] → Put in `internal/config` alongside other root config to avoid a new package; acceptable since config already imports os.

## Migration Plan

1. Add helper.
2. Replace all 8 `GIT_DBMS_DB`/`GIT_DBMS_TOKEN` lookups with helper calls.
3. Update help text/examples/docs.
4. Run `go build ./...`, `go test ./...`, `gofmt`.
5. Rollback: revert commit.