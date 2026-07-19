FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml* ./
RUN corepack enable && pnpm install --frozen-lockfile=false
COPY web/ ./
RUN pnpm build

FROM golang:1.24-alpine AS backend
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /studio ./cmd/server

FROM alpine:3.22
RUN addgroup -S studio && adduser -S studio -G studio && mkdir -p /app/data && chown -R studio:studio /app
WORKDIR /app
COPY --from=backend /studio /app/studio
COPY --from=web /src/web/dist /app/web/dist
USER studio
ENV APP_ADDR=:8080 APP_DATA_FILE=/app/data/studio.json APP_WEB_DIR=/app/web/dist
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/studio"]

