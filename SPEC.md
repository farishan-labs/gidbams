# gidbams Specification (v0.1)

## Purpose
Production-ready git-as-SSOT database engine for Faris Han Labs products.

## Record layout
- Root config: `gitdb.yaml`
- Collections declared with name/path.
- Each collection dir may contain `schema.yaml` (or `.collection/definition.yaml`).
- One record per file: `.md` (YAML frontmatter + body), `.json`, `.yaml`.

## Validation
- `gidbams validate [collection]` validates all records.
- `--from/--to` restricts to git-changed files (incremental).
- Types: string, int, float, bool, array, object, date. Constraints: required, unique, min, max, pattern, enum.

## Derived store
- `gidbams build [db-path]` creates SQLite with one table per collection.
- `.gitdb/data.db` is gitignored.

## MCP
- `gidbams mcp` runs stdio MCP server with tools: list_collections, list_records, get_record, validate, search_records.

## Triggers
- `.gitdb/triggers.yaml` defines shell triggers run via `gidbams triggers run <event>`.

## Licensing
- Code MIT.
EOF