# Design

The authoritative detailed design is [the approved Superpowers design](../../../docs/superpowers/specs/2026-07-19-production-newapi-mysql-pipeline-design.md).

## Components

```text
Vue UI -> authenticated REST/SSE -> workflow runner
                                      |-- New API / DeepSeek
                                      |-- New API / Seedance
                                      |-- downloader + FFmpeg
                                      `-- MySQL repository
```

## Key Decisions

- Keep one Go process and an in-process scheduler for the one-instance deployment.
- Use repository interfaces and MySQL in production; retain JSON only as a local fallback.
- Bootstrap New API configuration from environment variables and store the token encrypted.
- Generate and validate structured text before any paid video task is submitted.
- Persist every Seedance task ID and shot result so retries reuse successful work.
- Request Seedance native audio and add silence when a clip lacks audio.
- Serve generated assets through authenticated API routes.
- Include FFmpeg in the application image for deterministic final composition.
