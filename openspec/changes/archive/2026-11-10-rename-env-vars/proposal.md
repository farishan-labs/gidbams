## Why

After renaming the project from `git-dbms` to `gidbams`, the user-facing environment variables still use the old `GIT_DBMS_*` prefix (`GIT_DBMS_DB`, `GIT_DBMS_TOKEN`). This is inconsistent with the new binary/module name and confusing for users and docs. The rename should be completed for all user-facing identifiers, while keeping existing scripts working for at least one release.

## What Changes

- Introduce `GIDBAMS_DB` (derived SQLite path) and `GIDBAMS_TOKEN` (HTTP bearer token) as the preferred environment variable names.
- Continue to read the legacy `GIT_DBMS_DB` / `GIT_DBMS_TOKEN` names as deprecated fallbacks for one release, with the new names taking precedence when both are set.
- Update CLI help text, examples, and docs to use the new names.
- (Optional, next release) Drop legacy fallbacks and document the breaking change.

## Capabilities

### New Capabilities
- `configuration`: environment variable naming and precedence for derived DB path and HTTP bearer token.

### Modified Capabilities
(none)

## Impact

- Affected files: `internal/cli/cli.go`, `internal/mcp/server.go`, `internal/httpserver/server.go`, docs/examples.
- Affected behavior: env var lookup precedence (new name wins over legacy).
- Backward compatible: legacy names still work for one release.
- Risk: low. Existing scripts keep working; new name is additive.