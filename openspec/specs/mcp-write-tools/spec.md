# mcp-write-tools Specification

## Purpose
Defines the create, update, and delete record tools exposed by the gidbams MCP server, including validation and consistency with the derived store.
## Requirements
### Requirement: create_record tool creates a validated record
The MCP server SHALL expose a `create_record` tool that accepts `collection`, `id`, and `fields` (JSON object) and creates the record file when the record validates against the collection schema.

#### Scenario: Valid record is created
- **WHEN** `create_record` is called with a valid collection, a new id, and fields satisfying the schema
- **THEN** the record file is created and the tool returns success

#### Scenario: Invalid record is rejected
- **WHEN** `create_record` is called with fields violating the schema
- **THEN** the tool returns a validation error and no record file is created

#### Scenario: Duplicate id is rejected
- **WHEN** `create_record` is called with an id that already exists
- **THEN** the tool returns an error and the existing record is unchanged

### Requirement: update_record tool updates a validated record
The MCP server SHALL expose an `update_record` tool that accepts `collection`, `id`, and `fields` (JSON object) and updates the record file when the merged result validates against the collection schema.

#### Scenario: Valid update succeeds
- **WHEN** `update_record` is called for an existing record with fields satisfying the schema
- **THEN** the record file is updated and the tool returns success

#### Scenario: Invalid update is rejected
- **WHEN** `update_record` is called with fields violating the schema
- **THEN** the tool returns a validation error and the record file is unchanged

#### Scenario: Missing record is rejected
- **WHEN** `update_record` is called for an id that does not exist
- **THEN** the tool returns a not-found error

### Requirement: delete_record tool removes a record
The MCP server SHALL expose a `delete_record` tool that accepts `collection` and `id` and removes the record file.

#### Scenario: Existing record is deleted
- **WHEN** `delete_record` is called for an existing record
- **THEN** the record file is removed and the tool returns success

#### Scenario: Missing record returns not-found
- **WHEN** `delete_record` is called for an id that does not exist
- **THEN** the tool returns a not-found error

### Requirement: Writes use the store layer
All create/update/delete operations performed by MCP tools SHALL go through the existing store layer so that atomic writes and file locking behavior match the CLI.

#### Scenario: Concurrency safety
- **WHEN** two concurrent MCP write requests target the same collection
- **THEN** file locking prevents corruption (no interleaved/partial writes)

