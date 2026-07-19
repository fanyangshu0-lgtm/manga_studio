# Single-server deployment capability

## Requirements

### Requirement: external host services

The Docker deployment SHALL connect to Baota MySQL and New API through `host.docker.internal` mapped to `host-gateway`.

#### Scenario: production startup

- WHEN valid database, administrator, application-secret, and New API variables are supplied
- THEN the application migrates the database, validates FFmpeg, bootstraps the encrypted provider, and becomes ready.

#### Scenario: invalid configuration

- WHEN a required production variable is missing or a dependency is unreachable
- THEN startup fails with a redacted actionable error.

### Requirement: persistent media

The deployment SHALL mount `/app/data` so generated source clips, normalized clips, subtitles, and final MP4 files survive container replacement.

### Requirement: proxy-compatible progress

The application SHALL emit SSE headers and keep-alives compatible with a Baota reverse proxy configured with buffering disabled.
