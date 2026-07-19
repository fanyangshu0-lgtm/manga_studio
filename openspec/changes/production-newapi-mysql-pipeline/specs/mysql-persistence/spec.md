# MySQL persistence capability

## Requirements

### Requirement: durable production repository

The system SHALL persist projects, workflows, providers, runs, node attempts, and asset metadata in MySQL when database configuration is present.

#### Scenario: restart

- WHEN the application restarts
- THEN completed projects, workflows, provider configuration, run history, node outputs, and asset metadata remain available.

#### Scenario: interrupted run

- WHEN startup finds a run in queued or running state
- THEN the run is marked interrupted and can be resumed without regenerating verified successful shots.

### Requirement: safe migrations

The system SHALL apply ordered idempotent migrations under a database lock before accepting traffic.

#### Scenario: concurrent startup

- WHEN two processes attempt migration simultaneously
- THEN only one applies migrations and neither observes a partially migrated schema.

### Requirement: transactional state changes

Related aggregate changes SHALL commit atomically.

#### Scenario: project creation

- WHEN a project is created
- THEN the project and initial workflow both commit, or neither is visible.

#### Scenario: node completion

- WHEN a generated asset completes a node
- THEN node output and asset metadata commit in one transaction.
