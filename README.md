# git-dbms

Production-ready **git-as-a-source-of-truth database engine**. Treats a Git repository as the authoritative store: records are plain text files (Markdown with YAML frontmatter, JSON, or YAML), reviewed via PRs, versioned by commits, queried via a derived SQLite cache and an MCP server.

Built for Faris Han Labs products (providers, forager, provider-intel, etc.) where **git is the SSOT** and local AI agents need fast, queryable access.

## Features

- Collections with typed column schemas (`schema.yaml` or `.collection/definition.yaml`)
- Records: one file per record (Markdown frontmatter / JSON / YAML)
- Validation: required, types, min/max, pattern, enum, unique; incremental via git diff
- Derived SQLite build (`git-dbms build`) — git holds only text
- Read-only SQL query (`git-dbms query`)
- MCP server over stdio (`git-dbms mcp`) for AI agents
- CRUD CLI (`list`, `get`, `create`, `update`, `delete`)
- Stats/export helpers
- Triggers (`.gitdb/triggers.yaml`)
- Doctor health check
- Go library (`pkg/gitdbms`) for embedding

## Quick start

```sh
git-dbms init
# add collections under data/<name>/ with schema.yaml
git-dbms validate
git-dbms build
git-dbms mcp
```

## Shell completion

```sh
git-dbms completion bash > /etc/bash_completion.d/git-dbms
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