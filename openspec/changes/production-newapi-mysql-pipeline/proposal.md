# Proposal: Production New API and MySQL Pipeline

## Problem

The MVP currently executes deterministic mock nodes and persists all state in one atomic JSON file. It cannot call real models, produce real media, survive production concurrency, or safely run on a public server.

## Proposed Change

- Add a MySQL repository and idempotent schema migrations.
- Protect the application with a single-administrator authenticated session.
- Add encrypted New API provider configuration.
- Use DeepSeek V4 Pro for script, character, and storyboard generation.
- Use Seedance 2.0 Fast or Quality for asynchronous shot generation.
- Persist generated clips, compose them with FFmpeg, and expose authenticated previews/downloads.
- Add Docker and Baota deployment configuration for external MySQL and New API services.

## Outcome

One Docker application on the Baota server can turn a story idea into a durable, observable, downloadable MP4 while using the deployed New API gateway and Baota-managed MySQL.

## Non-Goals

- Multi-user or tenant isolation.
- Distributed workers, Redis, or horizontal scaling.
- Object storage or CDN integration.
- A separate TTS provider.
