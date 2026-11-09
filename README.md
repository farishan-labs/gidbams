# git-dbms

Production-ready **git-as-a-source-of-truth database engine**. Treats a Git repository as the authoritative store: records are plain text files (Markdown with YAML frontmatter, JSON, or YAML), reviewed via PRs, versioned by commits, queried via a derived SQLite cache and an MCP server.

Built for Faris Han Labs products (providers, forager, provider-intel, etc.) where **git is the SSOT** and local AI agents need fast, queryable access.

## Features

- Collections with typed column schemas (`.collection/definition.yaml` or `schema.yaml`)
- Records: one file per record (Markdown frontmatter / JSON / YAML)
- Validation: required, types, min/max, pattern, enum, unique; incremental via git diff
- Derived SQLite build (`git-dbms build`) — git holds only text
- MCP server over stdio (`git-dbms mcp`) for AI agents
- CRUD CLI (`list`, `get`, `create`, `update`, `delete`)
- Go library (`pkg/gitdbms`) for embedding in other tools
- Triggers/subscribers hooks (config-driven) for staleness rebuilds

## Quick start

```sh
git-dbms init
# add collections under data/<name>/ with schema.yaml
git-dbms validate
git-dbms build
git-dbms mcp
```

## Layout

```
gitdb.yaml                 # root config
data/
  providers/
    schema.yaml            # collection definition
    openai.md              # one record per file
.gitdb/
  data.db                  # derived (gitignored)
```

## Status

v0.1.0 — core engine. See SPEC.md.

## License

MIT
EOF