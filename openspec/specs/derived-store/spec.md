## Purpose

Defines the derived SQLite store built from git-tracked records.

## Requirements

### Requirement: Build derived SQLite database
The system SHALL create a SQLite database with one table per collection when running `gidbams build [db-path]`.

#### Scenario: Build succeeds
- **WHEN** `gidbams build` is run in a valid repository
- **THEN** a SQLite database is created containing one table per collection

#### Scenario: Custom db path
- **WHEN** `gidbams build <db-path>` is run
- **THEN** the database is written to the specified path

### Requirement: Derived database is gitignored
The derived database file (`.gitdb/data.db` by default) SHALL be gitignored and treated as a rebuildable artifact; git remains the source of truth.

#### Scenario: Gitignored artifact
- **WHEN** the derived database is created
- **THEN** it is not tracked by git (listed in .gitignore)

### Requirement: Read-only SQL query
The system SHALL support read-only SQL queries against the derived database via `gidbams query <sql>`.

#### Scenario: Query returns results
- **WHEN** `gidbams query` is run against a built database
- **THEN** the query results are printed without modifying the database