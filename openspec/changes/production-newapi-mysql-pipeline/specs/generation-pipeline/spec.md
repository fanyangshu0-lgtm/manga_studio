# Production generation pipeline capability

## Requirements

### Requirement: structured DeepSeek planning

The system SHALL use the configured DeepSeek model to produce validated script, character, and storyboard structures before submitting video tasks.

#### Scenario: invalid model response

- WHEN DeepSeek returns invalid or incomplete JSON
- THEN the text node fails with a safe diagnostic and no Seedance task is created.

#### Scenario: excessive shots

- WHEN a storyboard contains more than 12 shots
- THEN validation rejects it before video submission.

### Requirement: selectable Seedance quality

The system SHALL default to `doubao-seedance-2-0-fast-260128` and SHALL allow a node to select `doubao-seedance-2-0-260128`.

#### Scenario: default generation

- WHEN no video model override is configured
- THEN the fast model is submitted through New API.

#### Scenario: quality override

- WHEN the video node selects quality mode
- THEN the quality model is submitted without changing provider defaults.

### Requirement: durable asynchronous video tasks

The system SHALL persist each public New API task ID before polling and SHALL limit concurrent video tasks to the configured value.

#### Scenario: successful shot

- WHEN a Seedance task succeeds
- THEN its result is downloaded atomically, checksummed, registered as an asset, and marked reusable.

#### Scenario: timeout or cancellation

- WHEN a task times out or the run is cancelled
- THEN polling stops, no new shots are submitted, and completed shot assets remain reusable.

#### Scenario: retry

- WHEN a failed run is resumed
- THEN verified successful shots are reused and only missing or failed shots are submitted again.

### Requirement: final MP4 composition

The system SHALL normalize generated clips, preserve available audio, add silence when necessary, burn UTF-8 subtitles, and concatenate clips into a final MP4.

#### Scenario: mixed clip inputs

- WHEN some clips contain audio and others do not
- THEN FFmpeg produces one playable H.264/AAC MP4 with shots in storyboard order.

#### Scenario: composition failure

- WHEN FFmpeg fails
- THEN the run records a bounded stderr diagnostic while retaining source clips for retry.
