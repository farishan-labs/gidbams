## Purpose

Defines shell triggers configured per repository and executed on events.

## Requirements

### Requirement: Trigger configuration
The repository SHALL support a `.gitdb/triggers.yaml` file defining shell triggers.

#### Scenario: Config present
- **WHEN** `.gitdb/triggers.yaml` exists
- **THEN** the system loads the trigger definitions

### Requirement: Run triggers by event
The system SHALL execute configured triggers for a named event via `gidbams triggers run <event>`.

#### Scenario: Event triggers run
- **WHEN** `gidbams triggers run <event>` is invoked
- **THEN** the configured shell commands for that event are executed