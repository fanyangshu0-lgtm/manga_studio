# 绘界 · Manga Drama Studio

一个面向 AI 漫剧生产的节点式工作台。后端使用 Go，前端使用 Vue 3，交互参考 ComfyUI，但领域模型围绕剧本、角色、分镜、画面、配音和视频合成设计。

![status](https://img.shields.io/badge/status-MVP-f06445) ![Go](https://img.shields.io/badge/Go-1.22+-00ADD8) ![Vue](https://img.shields.io/badge/Vue-3-42b883)

## 当前能力

- 可拖动、连线、增删和配置的类型化节点画布。
- 服务端 DAG 校验：必填输入、端口类型、未知节点/端口、重复边与环。
- 项目和工作流持久化，运行前自动保存并快照。
- 异步工作流执行、节点级进度、SSE 实时更新和取消。
- Provider Token 加密保存、掩码展示、启停、权重和能力标签。
- 开箱即用的 mock 执行器，无需模型 Token 也能演示完整流程。
- Docker 单容器部署，Go 同时托管生产前端。

> 这是经过明确范围控制的 MVP。真实 LLM、ComfyUI、TTS 和 FFmpeg 适配器已预留执行器边界，但尚未内置。详见 [产品脑暴](docs/brainstorm.md) 和 [OpenSpec 变更](openspec/changes/mvp-comic-drama-studio)。

## 本地开发

要求：Go 1.22+、Node.js 20+、pnpm 9+。

```bash
cp .env.example .env
pnpm --dir web install

# 终端 1
go run ./cmd/server

# 终端 2
pnpm --dir web dev
```

打开 <http://localhost:5173>。开发服务器会把 `/api` 代理到 Go 服务的 `8080` 端口。

若要保存渠道 Token，必须先设置 `APP_SECRET_KEY`。PowerShell 示例：

```powershell
$env:APP_SECRET_KEY = "replace-with-a-long-random-secret"
go run ./cmd/server
```

## Docker

```bash
docker compose up --build
```

打开 <http://localhost:8080>。生产环境务必通过外部环境变量设置一个随机的 `APP_SECRET_KEY`，不要使用 compose 中的开发默认值。

## API 概览

| 方法 | 路径 | 用途 |
|---|---|---|
| GET/POST | `/api/v1/projects` | 列出/创建项目 |
| GET/PUT | `/api/v1/projects/{id}/workflow` | 读取/保存工作流 |
| GET | `/api/v1/catalog/nodes` | 节点类型与端口目录 |
| POST | `/api/v1/runs` | 创建运行任务 |
| GET | `/api/v1/runs/{id}/events` | SSE 运行事件 |
| POST | `/api/v1/runs/{id}/cancel` | 取消任务 |
| GET/POST | `/api/v1/providers` | 列出/创建模型渠道 |
| PATCH/DELETE | `/api/v1/providers/{id}` | 启停/删除渠道 |

## 架构

```text
Vue 3 / Vue Flow
       │ REST + SSE
Go HTTP API
       ├── workflow validator / node catalog
       ├── async runner / event broker
       ├── executor interface ── mock executor (MVP)
       └── repository ── atomic JSON store (MVP)
```

JSON 存储是为本地零依赖体验做的有意取舍，不支持多实例并发。准备生产化时，应优先实现 SQLite/PostgreSQL repository、登录与租户隔离、任务队列、对象存储和托管密钥服务。

## 验证

```bash
go test ./...
pnpm --dir web build
```

## 目录

```text
cmd/server/       Go 入口
internal/api/     REST/SSE 与密钥加密
internal/domain/  领域模型
internal/runner/  任务运行器与执行器接口
internal/store/   原子 JSON repository
internal/workflow 节点目录、示例图和 DAG 校验
web/              Vue 3 工作台
docs/             产品脑暴
openspec/         提案、设计、规格与任务清单
```

