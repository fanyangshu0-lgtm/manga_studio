# Manga Drama Studio Production Pipeline Design

**Status:** Approved on 2026-07-19

## Goal

Turn the mock-only MVP into a single-server production application that stores state in Baota-managed MySQL, uses the user's New API instance for DeepSeek script generation and Seedance 2.0 video generation, protects administration with a login, and produces a downloadable final MP4.

## Scope

- MySQL persistence for projects, workflows, providers, runs, node states, and generated asset metadata.
- A protected single-administrator web application.
- Encrypted New API credentials and configurable model routing.
- DeepSeek V4 Pro story, script, character, and storyboard generation.
- Seedance 2.0 Fast video generation by default, with a per-node quality-model override.
- Asynchronous task polling, resumable per-shot progress, local asset persistence, subtitle generation, and FFmpeg composition.
- Docker deployment against MySQL and New API services running on the same Linux host.

This version does not add multi-tenant accounts, distributed workers, Redis, object storage, collaborative editing, or a separate TTS provider. Seedance native audio is requested for every generated clip. When a returned clip has no audio stream, composition adds silence so the pipeline remains deterministic.

## Architecture

```text
Browser
  | signed HttpOnly admin session
  v
Go HTTP application
  |-- REST + SSE API
  |-- workflow validation and orchestration
  |-- New API client
  |     |-- DeepSeek chat completions
  |     `-- Seedance asynchronous video tasks
  |-- FFmpeg composition
  |-- protected local asset serving
  `-- repository interfaces
          `-- MySQL repository

Persistent volume: /app/data/assets
External services: host.docker.internal:3000 and host.docker.internal:3306
```

The API process continues to own the in-process task scheduler because the target deployment is one Docker instance. Every durable state transition is written to MySQL before an SSE event is published. On restart, runs left in `queued` or `running` become `interrupted`; the administrator can resume them and successful shot assets are reused.

## Configuration

Deployment uses environment variables and never commits credentials:

```env
APP_ADDR=:8080
APP_WEB_DIR=/app/web/dist
APP_ASSET_DIR=/app/data/assets
APP_SECRET_KEY=<random 32-byte-or-longer value>
APP_ADMIN_USERNAME=admin
APP_ADMIN_PASSWORD=<strong password>

DATABASE_HOST=host.docker.internal
DATABASE_PORT=3306
DATABASE_NAME=manga_drama_studio
DATABASE_USER=manga_studio
DATABASE_PASSWORD=<database password>
DATABASE_PARAMS=charset=utf8mb4&parseTime=true&loc=UTC

NEW_API_BASE_URL=http://host.docker.internal:3000
NEW_API_TOKEN=<new New API token>
SCRIPT_MODEL=deepseek-v4-pro
VIDEO_MODEL_FAST=doubao-seedance-2-0-fast-260128
VIDEO_MODEL_QUALITY=doubao-seedance-2-0-260128
VIDEO_DEFAULT_MODEL=doubao-seedance-2-0-fast-260128
VIDEO_MAX_CONCURRENCY=2
VIDEO_POLL_INTERVAL=5s
VIDEO_TASK_TIMEOUT=30m
```

`NEW_API_TOKEN` only bootstraps or updates the managed `new-api` provider. Its stored token is AES-GCM encrypted with `APP_SECRET_KEY`. Changing `APP_SECRET_KEY` without re-entering the token intentionally makes existing ciphertext unreadable.

## Persistence Model

The MySQL repository uses `database/sql` and `github.com/go-sql-driver/mysql`. Startup runs idempotent numbered migrations under a database advisory lock.

- `schema_migrations`: applied migration versions.
- `projects`: project identity, name, description, and timestamps.
- `workflows`: one current workflow per project, revision, and JSON document.
- `providers`: kind, base URL, capabilities JSON, model settings JSON, weight, enabled state, encrypted secret, hint, and timestamps.
- `runs`: workflow snapshot JSON, status, progress, message, timestamps, and resumability state.
- `run_nodes`: per-node status, progress, message, input/output JSON, upstream task ID, attempt count, and timestamps.
- `assets`: project/run/node ownership, asset kind, relative path, MIME type, byte size, checksum, and creation time.

JSON columns store versioned workflow and run snapshots; indexed lifecycle fields remain normal columns. Transactions cover project plus initial workflow creation, run plus node state transitions, and node completion plus asset registration.

The atomic JSON store remains only as a local-development fallback when `DATABASE_HOST` is empty. Production compose requires MySQL and fails fast when it cannot connect.

## Authentication and Secrets

`POST /api/v1/auth/login` validates environment-configured credentials with constant-time comparisons and issues a signed, expiring, HttpOnly, SameSite=Strict cookie. Session and logout endpoints support frontend bootstrap and expiry.

All `/api/v1` endpoints except login/session require authentication. `/healthz` stays public. Assets use authenticated `/api/v1/assets/{id}` routes. Production CORS is same-origin and login attempts are rate-limited per client address.

Provider APIs never return plaintext or encrypted tokens. Provider updates treat an empty token as “keep the existing secret.”

## Generation Pipeline

### Structured text planning

DeepSeek V4 Pro performs three stages: `script`, `character`, and `storyboard`. Storyboard output is strict JSON with at most 12 ordered shots. Each shot includes visual prompt, dialogue or narration, subtitle, duration, aspect ratio, continuity guidance, and safety-safe negative guidance. Invalid or truncated JSON fails before paid video jobs begin.

### Seedance video generation

Each shot is submitted to New API `POST /v1/video/generations`. The default model is `doubao-seedance-2-0-fast-260128`; a node can select `doubao-seedance-2-0-260128`. Requests include bounded duration, ratio, resolution, `metadata.generate_audio=true`, and `metadata.watermark=false`.

The public New API task ID is saved before polling `GET /v1/video/generations/{id}`. At most two tasks run concurrently. Polling honors cancellation, a five-second interval, and a 30-minute task timeout. Successful URLs are downloaded with size/MIME validation to temporary files and atomically renamed.

Retry reuses completed shot files whose checksum still matches, and resubmits only missing or failed shots.

### Final composition

FFmpeg normalizes clips to the selected resolution, 30 FPS H.264 video, and AAC stereo audio. Clips without audio receive silence. UTF-8 subtitles are generated from shot timings and burned into the timeline. Clips are concatenated in storyboard order and atomically published as `final.mp4`.

The final file and every source clip are registered in MySQL and exposed as authenticated stream/download URLs. FFmpeg stderr is stored only as a bounded diagnostic tail.

## User Interface

- Administrator login and session bootstrap.
- New API provider settings with base URL, masked token, script model, fast/quality video models, and connection checks.
- Structured node controls for models, ratio, resolution, duration, shot count, and subtitles.
- Run details for text results, per-shot tasks, retries, clip previews, composition progress, and final playback/download.
- A cost warning showing the number of Seedance shots before submission.

## Error Handling

- Startup validates configuration, database access, secret handling, and FFmpeg availability without logging secrets.
- Upstream errors retain safe status, code, request ID, and redacted message.
- Text-stage failure prevents video submission.
- Shot failures preserve successful siblings and assets.
- Cancellation stops new submissions and local polling/downloads but retains completed work.
- Database writes use timeouts and transactions; SSE is emitted only after durable updates.

## Verification Strategy

Implementation follows test-first development:

- Repository contract tests run against MySQL 8 and cover transactions, restart recovery, provider secrets, retries, and assets.
- `httptest` contract tests cover DeepSeek requests/JSON validation and Seedance submission, polling, cancellation, timeout, redaction, download validation, and reuse.
- FFmpeg tests create tiny synthetic clips, normalize mixed audio/no-audio inputs, burn subtitles, concatenate, and inspect output with `ffprobe`.
- Authentication tests cover login, failure, expiry, logout, protected resources, cookie attributes, and rate limiting.
- Frontend tests cover login, provider/model configuration, progress, retry, and final downloads.
- Release checks run `go test ./...`, MySQL integration tests, `pnpm --dir web build`, `docker compose config`, and a container smoke test.

## Deployment

The image includes the Go binary, Vue assets, FFmpeg, CA certificates, and Chinese subtitle fonts. Compose mounts `/app/data`, publishes 8080, maps `host.docker.internal` to `host-gateway`, loads `.env`, and does not create MySQL.

Baota creates database `manga_drama_studio` and user `manga_studio`, permits that user from the Docker host/subnet, and grants access only to this database. The application migrates on first start. Baota reverse proxy terminates HTTPS and disables buffering for SSE.

The previously disclosed New API token must be revoked; deployment uses a new token entered only in server `.env`.
