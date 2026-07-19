# Manga Drama Studio conventions

## Architecture

- Backend: Go 1.22+, standard library HTTP server.
- Frontend: Vue 3, TypeScript, Vite, Pinia, Vue Flow.
- Layers: HTTP API -> services -> domain/store.
- Persistence is behind repository interfaces. The MVP uses an atomic JSON file so local startup has zero infrastructure dependencies.

## Rules

- Keep domain types independent from HTTP and storage.
- Validate every workflow before saving or running it.
- Never return provider secrets from an API response; only return a masked hint.
- New node kinds must declare input/output port types and validation rules.
- Use TypeScript strictly. Avoid `any`.
- Keep UI copy in Chinese for this MVP and ensure controls have accessible labels.
- Run `go test ./...` and `pnpm --dir web build` before delivery.

