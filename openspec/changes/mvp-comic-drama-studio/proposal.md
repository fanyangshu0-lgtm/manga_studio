# Change: MVP comic-drama studio

## Why

Creators need a repeatable pipeline rather than disconnected prompt forms. A visual DAG makes dependencies, reuse, partial failure, and provider selection visible.

## What changes

- Add project and workflow persistence APIs.
- Add a typed node graph editor.
- Add graph validation and an asynchronous generation runtime.
- Add SSE run events.
- Add masked provider-token management.
- Add a runnable local and Docker development experience.

## Impact

This is the initial product baseline. Real model adapters are intentionally behind an executor boundary and are not part of this change.

