## Purpose

Defines the stdio MCP server exposing gidbams collections, records, validation, and queries to AI agents.

## Requirements

### Requirement: Stdio MCP server
The system SHALL run an MCP server over stdio when invoking `gidbams mcp`.

#### Scenario: Server starts
- **WHEN** `gidbams mcp` is run
- **THEN** the process serves MCP requests over stdin/stdout

### Requirement: MCP tools
The MCP server SHALL expose tools for listing collections, listing records, getting a record, validating, searching records (per-collection and global), running read-only SQL queries, rebuilding the derived database, returning stats, running doctor checks, showing a collection schema, and returning the build version.

#### Scenario: Tool listed
- **WHEN** an MCP client calls `tools/list`
- **THEN** the response includes the supported tool definitions with input schemas

#### Scenario: Tool invocation
- **WHEN** an MCP client calls a supported tool with valid arguments
- **THEN** the server returns the tool result as text content

### Requirement: MCP resources
The MCP server SHALL expose each collection as a resource with URI `gitdb://<collection>`.

#### Scenario: Resource read
- **WHEN** an MCP client reads `gitdb://<collection>/<id>`
- **THEN** the server returns the record's fields as JSON