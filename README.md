# Manga Drama Studio

一套可直接部署的 AI 漫剧生产工作台：DeepSeek 负责中文剧本、角色和分镜，豆包 Seedance 2.0 负责带原生音频的视频镜头，FFmpeg 自动拼接并烧录字幕。服务端使用 Go，界面使用 Vue 3，生产数据保存到 MySQL。

## 已实现

- 管理员登录、HttpOnly 签名会话和登录限速。
- MySQL 自动建表、项目/工作流/任务/资产持久化。
- New API Token 使用 AES-GCM 加密保存，接口永不返回明文。
- `deepseek-v4-pro` 生成结构化剧本、角色档案和最多 12 个分镜。
- `doubao-seedance-2-0-fast-260128` 默认生成视频，可切换 `doubao-seedance-2-0-260128` 质量模型。
- 两路并发生成、任务 ID 持久化、失败/取消/服务重启后恢复。
- 视频校验下载、SHA-256 校验、FFmpeg 拼接、静音轨补齐和中文字幕。
- Docker 单容器部署；MySQL 和 New API 使用宝塔宿主机已有服务。

## 宝塔部署

### 1. 创建 MySQL 数据库

在宝塔「数据库」中创建：

- 数据库：`manga_drama_studio`
- 用户：`manga_studio`
- 访问权限：所有人（MySQL 中对应 `%`，用于允许 Docker 网桥访问）
- 字符集：`utf8mb4`

也可以用 root 执行：

```sql
CREATE DATABASE manga_drama_studio CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'manga_studio'@'%' IDENTIFIED BY '换成新的强密码';
GRANT ALL PRIVILEGES ON manga_drama_studio.* TO 'manga_studio'@'%';
FLUSH PRIVILEGES;
```

确认 MySQL 监听宿主机接口，而不只是 `127.0.0.1`。无需在云安全组向公网开放 3306；只需允许 Docker 网桥访问宿主机 3306。

### 2. 上传代码并配置环境变量

将项目上传到例如 `/www/wwwroot/manga-drama-studio`：

```bash
cd /www/wwwroot/manga-drama-studio
cp .env.example .env
openssl rand -hex 32
```

编辑 `.env`，至少替换以下值：

```dotenv
APP_SECRET_KEY=上一步生成的64位随机字符串
APP_ADMIN_USERNAME=你的管理员账号
APP_ADMIN_PASSWORD=新的强密码
DATABASE_PASSWORD=宝塔数据库密码
NEW_API_TOKEN=在New API后台新建的令牌
```

项目已经预设：

```dotenv
NEW_API_BASE_URL=http://host.docker.internal:3000
SCRIPT_MODEL=deepseek-v4-pro
VIDEO_DEFAULT_MODEL=doubao-seedance-2-0-fast-260128
VIDEO_QUALITY_MODEL=doubao-seedance-2-0-260128
DATABASE_HOST=host.docker.internal
```

不要继续使用曾经发到聊天、终端历史或截图里的 Token；请在 New API 后台撤销它并创建新 Token。

### 3. 构建和启动

```bash
docker compose config --quiet
docker compose up -d --build
docker compose ps
docker compose logs --tail=200 manga-drama-studio
curl http://127.0.0.1:3001/healthz
```

成功时健康检查返回：

```json
{"status":"ok"}
```

浏览器访问 `http://服务器公网IP:3001`。这是 HTTP 地址，不要直接写成 `https://IP:3001`。

### 4. 宝塔反向代理 HTTPS（推荐）

在宝塔创建网站和 SSL 证书，把域名反向代理到：

```text
http://127.0.0.1:3001
```

确认代理支持长连接/SSE，然后把 `.env` 改为：

```dotenv
APP_SESSION_SECURE=true
```

重启：

```bash
docker compose up -d
```

## 更新

上传或拉取新代码后执行：

```bash
docker compose up -d --build
docker compose ps
docker compose logs --tail=100 manga-drama-studio
```

MySQL 表结构由服务启动时自动迁移；Docker 命名卷 `studio-data` 保存生成的视频，重新构建容器不会删除数据。

## 本地验证

```bash
go test ./...
pnpm --dir web install --frozen-lockfile
pnpm --dir web build
```

生产镜像包含 FFmpeg 和 Noto CJK 字体。`Dockerfile` 已配置国内 Go、npm 和 Alpine 镜像源，以减少服务器访问海外源超时。
