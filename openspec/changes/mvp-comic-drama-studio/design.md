# Design

## Components

```text
Vue workspace -> REST/SSE API -> workflow validator -> runner -> node executor
                         |              |               |
                         +---------- JSON store --------+
```

## Data model

- Project owns one editable workflow in the MVP.
- Workflow contains typed nodes, ports, edges and canvas viewport.
- Run snapshots a workflow to avoid edits changing an active execution.
- RunNode records state, progress, logs, timing and outputs.
- Provider stores routing metadata and an encrypted credential.

## Runtime

The API validates and snapshots the graph, then queues a run in-process. The runner topologically orders nodes and publishes immutable events. A process restart marks previously running work as interrupted. The executor interface receives node configuration and upstream outputs; the MVP mock executor produces deterministic sample assets.

## Security

Secrets are encrypted with AES-GCM using `APP_SECRET_KEY` and are never serialized to clients. The local MVP has no login and must not be exposed to the public internet. Authentication, per-user ownership and a managed secret vault are required before production deployment.

## Trade-offs

Atomic JSON persistence provides a zero-dependency local MVP and easy inspection but is not suitable for concurrent multi-instance deployment. Repository interfaces make migration to SQLite/PostgreSQL an isolated follow-up.

