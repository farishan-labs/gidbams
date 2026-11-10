## Purpose

Defines the on-disk record and collection layout for a gidbams repository.

## Requirements

### Requirement: Root configuration file
The repository SHALL contain a root configuration file named `gitdb.yaml` declaring collections by name and path.

#### Scenario: Root config present
- **WHEN** a repository is initialized with `gidbams init`
- **THEN** a `gitdb.yaml` file exists at the repository root

### Requirement: Collection schema
Each collection directory MAY contain a `schema.yaml` or `.collection/definition.yaml` file defining typed columns.

#### Scenario: Schema loaded
- **WHEN** a collection directory contains `schema.yaml`
- **THEN** the system loads its column definitions for validation and SQLite build

### Requirement: One record per file
Each record SHALL be stored as a single file with extension `.md` (YAML frontmatter plus body), `.json`, or `.yaml`.

#### Scenario: Markdown record
- **WHEN** a collection contains a `.md` record file
- **THEN** its YAML frontmatter is parsed as fields and the remainder as body
