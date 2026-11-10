# gidbams

Production-ready **git-as-a-source-of-truth database engine**. Treats a Git repository as the authoritative store: records are plain text files (Markdown with YAML frontmatter, JSON, or YAML), reviewed via PRs, versioned by commits, queried via a derived SQLite cache and an MCP server.

Built for Faris Han Labs products (providers, forager, provider-intel, etc.) where **git is the SSOT** and local AI agents need fast, queryable access.

## Problem

Many product datasets need a store that is:

- **Reviewable and versioned** — changes go through PRs and commits, not opaque database writes.
- **Agent-queryable** — local AI agents need fast SQL/MCP access without a running database server.
- **Text-first** — git should hold only plain text; derived indexes are rebuildable artifacts.

`gidbams` fills that gap: **git is the SSOT, SQLite is the derived index, MCP/CLI are the access layer.**

## Alternatives and similar tools

| Tool | Paradigm | Why not (or when to use) |
|---|---|---|
| **inGitDB** | Git-files-first DB with typed schemas, CLI CRUD, triggers/subscribers spec | Closest analogue. Borrow its collection layout, incremental validation, and trigger patterns; it has no shipped MCP server and low maturity, and uses YAML/JSON rather than Markdown frontmatter. |
| **Dolt / Doltgres / DoltLite** | Version-controlled SQL database (DB is SSOT, Git-like history on tables) | Different paradigm (DB-first). Pick it when you need real merge semantics, concurrent writes, or a live SQL server; heavier than text records need. |
| **TerminusDB** | Graph database with git-like version control (branches/merges/time-travel) | Graph/versioned-DB paradigm; overkill for flat record collections. |
| **Astro content collections** | Build-time content layer (Zod schemas over md/mdx/json/yaml in git) | Best fit for the public site layer; complements `gidbams` (the site reads derived data). |
| **Keystatic / Decap / TinaCMS** | Git-based CMS with admin UI over files in git | Optional contributor-UI layer on top of a git SSOT; not a replacement. |
| **gitrows / gh-as-db / @naturalcycles/github-db** | Single or sharded JSON file per collection in GitHub | Reference patterns (SHA optimistic locking, sharding, transactions); tiny adoption — treat as reference only. |
| **Generic Git MCP servers** (modelcontextprotocol git, cyanheads/git-mcp) | Expose raw git operations as MCP tools | Generic git ops, no record/schema/DB semantics. `gidbams` adds collections, validation, derived SQLite, and SQL. |
| **DVC / LakeFS / Nessie** | Data versioning for data lakes / ML pipelines | Different audience (ML/data lakes), not record-level databases. |

See `research/topics/gidbams/` and `research/topics/git-as-ssot-db/` for the full comparison and evidence.

## Features

- Collections with typed column schemas (`schema.yaml` or `.collection/definition.yaml`)
- Records: one file per record (Markdown frontmatter / JSON / YAML)
- Validation: required, types, min/max, pattern, enum, unique; incremental via git diff
- Derived SQLite build (`gidbams build`) — git holds only text
- Read-only SQL query (`gidbams query`)
- MCP server over stdio (`gidbams mcp`) for AI agents
- CRUD CLI (`list`, `get`, `create`, `update`, `delete`)
- Stats/export helpers
- Triggers (`.gitdb/triggers.yaml`)
- Doctor health check
- Go library (`pkg/gidbams`) for embedding

## Quick start

```sh
gidbams init
# add collections under data/<name>/ with schema.yaml
gidbams validate
gidbams build
gidbams mcp
```

## Shell completion

```sh
gidbams completion bash > /etc/bash_completion.d/gidbams
```

## Commands

| Command | Purpose |
|---|---|
| `init` | Initialize repository |
| `validate [collection] [--from --to]` | Validate records |
| `list <collection>` | List record ids |
| `get <collection> <id>` | Print record |
| `create/update/delete` | CRUD |
| `build [db-path]` | Build derived SQLite |
| `query <sql>` | Read-only SQL |
| `stats` | Record counts |
| `export <collection>` | JSON export |
| `mcp` | MCP server (stdio) |
| `collections` | List collections |
| `triggers run <event>` | Run triggers |
| `doctor` | Health check |
| `version` | Version |

## Layout

```
gitdb.yaml
data/
  providers/
    schema.yaml
    openai.md
.gitdb/
  data.db  (gitignored)
```

## Status

v0.1.0. See SPEC.md.

## License

MIT
EOF