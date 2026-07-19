# Generation runtime capability

## Requirements

### Requirement: asynchronous run

The system SHALL snapshot a valid workflow and return a run identifier without waiting for generation to finish.

#### Scenario: observe progress

- WHEN a client subscribes to a run event stream
- THEN it receives run and node state changes until a terminal state.

#### Scenario: node failure

- WHEN an executor fails
- THEN the node and run become failed, downstream nodes do not start, and the error is recorded.

### Requirement: cancellation

The system SHALL allow a queued or running run to be cancelled and SHALL stop scheduling additional nodes.

