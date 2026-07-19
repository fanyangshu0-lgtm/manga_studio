# Workflow editor capability

## Requirements

### Requirement: edit a typed graph

The system SHALL let a creator add, remove, position, connect and configure supported node types.

#### Scenario: compatible connection

- WHEN an output port is connected to an input port with the same media type
- THEN the edge is accepted and can be persisted.

#### Scenario: invalid graph

- WHEN a graph contains a cycle, missing required input, unknown port, duplicate edge or incompatible types
- THEN saving or running returns structured validation issues.

### Requirement: persistent workflow

The system SHALL persist the nodes, edges and viewport for a project and return the latest saved revision.

