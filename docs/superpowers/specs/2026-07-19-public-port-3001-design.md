# Public Port 3001 Design

## Decision

Change the default host-published port from `8080` to `3001`. Keep the application and Docker container listening on internal port `8080`.

## Changes

- Set `APP_PUBLIC_PORT=3001` in `.env.example`.
- Change the Compose fallback mapping to `${APP_PUBLIC_PORT:-3001}:8080`.
- Update public health-check, browser, and Baota reverse-proxy examples in `README.md` to host port `3001`.
- Leave `APP_ADDR=:8080`, Docker `EXPOSE 8080`, and the container health check unchanged.

## Verification

- Search deployment files to ensure host-facing examples use `3001`.
- Confirm internal service references still use `8080`.
- Run `git diff --check` before committing and pushing.
