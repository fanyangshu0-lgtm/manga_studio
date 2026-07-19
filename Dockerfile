FROM node:22-alpine AS web
WORKDIR /src/web
RUN npm config set registry https://registry.npmmirror.com && corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.24-alpine AS backend
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /studio ./cmd/server

FROM alpine:3.22
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates ffmpeg font-noto-cjk tzdata \
    && addgroup -S studio \
    && adduser -S studio -G studio \
    && mkdir -p /app/data/assets \
    && chown -R studio:studio /app
WORKDIR /app
COPY --from=backend /studio /app/studio
COPY --from=web /src/web/dist /app/web/dist
USER studio
ENV APP_ADDR=:8080 APP_WEB_DIR=/app/web/dist APP_ASSET_DIR=/app/data/assets TZ=Asia/Shanghai
EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=5 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/studio"]
