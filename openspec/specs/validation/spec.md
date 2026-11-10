## Purpose

Defines record validation rules and the incremental validation mode for gidbams collections.

## Requirements

### Requirement: Validate all records in a collection
The system SHALL validate all records in a named collection when running `gidbams validate <collection>`.

#### Scenario: Valid collection passes
- **WHEN** `gidbams validate <collection>` is run on a collection whose records satisfy the schema
- **THEN** validation succeeds with no errors

#### Scenario: Invalid record reported
- **WHEN** a record violates a schema constraint
- **THEN** the command reports the record, field, and reason

### Requirement: Incremental validation via git range
The system SHALL support `--from` and `--to` flags to validate only records changed between two git commits.

#### Scenario: Changed files only
- **WHEN** `gidbams validate <collection> --from <commit> --to <commit>` is run
- **THEN** only records changed in that range are validated

### Requirement: Supported field types
The schema SHALL support field types: string, int, float, bool, array, object, and date.

#### Scenario: Type mismatch rejected
- **WHEN** a field value does not match its declared type
- **THEN** validation reports a type error for that field

### Requirement: Supported constraints
The schema SHALL support constraints: required, unique, min, max, pattern, and enum.

#### Scenario: Required field missing
- **WHEN** a required field is absent from a record
- **THEN** validation reports a required-field error

#### Scenario: Unique violation
- **WHEN** two records share the same value for a unique field
- **THEN** validation reports a uniqueness error

#### Scenario: Pattern/enum/min/max enforced
- **WHEN** a field value violates pattern, enum, min, or max
- **THEN** validation reports the corresponding constraint error