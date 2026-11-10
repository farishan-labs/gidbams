## Purpose

Defines how gidbams resolves the derived SQLite database path and HTTP bearer token from environment variables, including the transition from the legacy `GIT_DBMS_*` names to the new `GIDBAMS_*` names.

## ADDED Requirements

### Requirement: Preferred environment variable names use the gidbams prefix
The system SHALL read the derived SQLite database path from `GIDBAMS_DB` and the HTTP bearer token from `GIDBAMS_TOKEN`.

#### Scenario: New names are used
- **WHEN** `GIDBAMS_DB` is set
- **THEN** the system uses its value as the derived database path

#### Scenario: New token name is used
- **WHEN** `GIDBAMS_TOKEN` is set
- **THEN** the system uses its value as the HTTP bearer token

### Requirement: Legacy names remain supported as deprecated fallbacks
The system SHALL continue to read `GIT_DBMS_DB` and `GIT_DBMS_TOKEN` for one release after the rename, treating them as deprecated aliases.

#### Scenario: Legacy DB path still works
- **WHEN** only `GIT_DBMS_DB` is set
- **THEN** the system uses its value as the derived database path

#### Scenario: Legacy token still works
- **WHEN** only `GIT_DBMS_TOKEN` is set
- **THEN** the system uses its value as the HTTP bearer token

### Requirement: New names take precedence over legacy names
When both the new and legacy environment variables are set, the system SHALL use the new name's value and ignore the legacy value.

#### Scenario: New name wins for DB path
- **WHEN** both `GIDBAMS_DB` and `GIT_DBMS_DB` are set
- **THEN** the system uses the value of `GIDBAMS_DB`

#### Scenario: New name wins for token
- **WHEN** both `GIDBAMS_TOKEN` and `GIT_DBMS_TOKEN` are set
- **THEN** the system uses the value of `GIDBAMS_TOKEN`