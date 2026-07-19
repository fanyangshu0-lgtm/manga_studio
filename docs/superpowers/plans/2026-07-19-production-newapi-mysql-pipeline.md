# Production New API and MySQL Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert Manga Drama Studio from a mock/JSON MVP into an authenticated, MySQL-backed production application that uses New API with DeepSeek V4 Pro and Seedance 2.0 to produce downloadable final MP4 files.

**Architecture:** Keep one Go HTTP process with an in-process asynchronous runner. Introduce repository interfaces with JSON and MySQL implementations, a signed-cookie administrator session, an encrypted New API provider, typed DeepSeek/Seedance clients, durable assets, and an FFmpeg composer. The Vue application remains same-origin and receives durable run progress through REST and SSE.

**Tech Stack:** Go 1.22, standard library HTTP/database APIs, `github.com/go-sql-driver/mysql`, `github.com/stretchr/testify`, Vue 3, TypeScript, Pinia, Vite, Vitest, MySQL 8, FFmpeg/ffprobe, Docker Compose.

## Global Constraints

- Production database is Baota MySQL at `host.docker.internal:3306`; compose must not create a production MySQL service.
- New API is reached from the container at `http://host.docker.internal:3000`.
- Script model is `deepseek-v4-pro`.
- Fast video model is `doubao-seedance-2-0-fast-260128`; quality model is `doubao-seedance-2-0-260128`; fast is the default.
- Seedance shot concurrency defaults to 2, storyboard shot count is capped at 12, polling defaults to 5 seconds, and each upstream task times out after 30 minutes.
- Provider tokens are AES-GCM encrypted and never serialized or logged.
- Every `/api/v1` resource except authentication bootstrap requires a signed administrator session.
- Generated files live under `/app/data/assets` and are served only through authenticated asset routes.
- The previously disclosed New API token is not reused; deployment requires a newly generated token.
- Existing domain types stay independent from HTTP and storage.
- Go JSON uses the standard library because this repository has no wrapper requirement.
- Frontend TypeScript remains strict and user-facing copy remains Chinese.
- All production behavior follows test-first red-green-refactor cycles.

---

### Task 0: Establish the MVP Source Baseline

**Files:**
- Track: `.env.example`, `.gitignore`, `AGENTS.md`, `Dockerfile`, `Makefile`, `README.md`, `cmd/`, `data/`, `docker-compose.yml`, `go.mod`, `internal/`, `openspec/`, `web/`
- Exclude: `.env`, `server.exe`, `web/node_modules/`, `web/dist/`, generated data and media

**Interfaces:**
- Produces: a clean Git baseline from which implementation changes and red-green steps are reviewable.

- [ ] **Step 1: Verify the current MVP before tracking it**

Run:

```powershell
go test ./...
pnpm --dir web build
```

Expected: Go tests pass and the Vue production build exits 0.

- [ ] **Step 2: Stage the existing MVP without generated files**

Run:

```powershell
git add .env.example .gitignore AGENTS.md Dockerfile Makefile README.md cmd data docker-compose.yml go.mod internal openspec web
git status --short
```

Expected: source files are staged; `.env`, `server.exe`, `web/node_modules`, `web/dist`, and runtime data are absent.

- [ ] **Step 3: Commit the baseline**

```powershell
git commit -m "chore: track manga drama studio mvp"
```

Expected: commit succeeds and remaining status contains only the implementation-plan document until it is committed.

---

### Task 1: Configuration and Repository Boundary

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/store/repository.go`
- Modify: `internal/store/store.go`
- Modify: `internal/api/server.go`
- Modify: `internal/runner/runner.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `config.Load() (config.Config, error)`, `store.Repository`, and a JSON repository conforming to the production repository contract.
- `store.Repository` methods return storage errors instead of hiding failures.

- [ ] **Step 1: Write failing configuration tests**

Create table-driven tests that call `config.LoadFromLookup(func(string) string)` and assert exact defaults plus production validation:

```go
func TestLoadFromLookupProductionRequiresSecretsAndDatabase(t *testing.T) {
    values := map[string]string{"APP_ENV": "production"}
    _, err := LoadFromLookup(func(key string) string { return values[key] })
    require.ErrorContains(t, err, "APP_SECRET_KEY")
}

func TestLoadFromLookupUsesApprovedModels(t *testing.T) {
    cfg, err := LoadFromLookup(func(string) string { return "" })
    require.NoError(t, err)
    assert.Equal(t, "deepseek-v4-pro", cfg.NewAPI.ScriptModel)
    assert.Equal(t, "doubao-seedance-2-0-fast-260128", cfg.NewAPI.FastVideoModel)
    assert.Equal(t, "doubao-seedance-2-0-260128", cfg.NewAPI.QualityVideoModel)
    assert.Equal(t, 2, cfg.Video.MaxConcurrency)
}
```

- [ ] **Step 2: Verify configuration tests fail because the package is absent**

Run: `go test ./internal/config -v`

Expected: FAIL because `LoadFromLookup` and configuration types do not exist.

- [ ] **Step 3: Implement typed configuration**

Define focused structs and duration/int parsing:

```go
type Config struct {
    Environment string
    Address string
    WebDir string
    AssetDir string
    SecretKey string
    Admin Admin
    Database Database
    NewAPI NewAPI
    Video Video
}

type Database struct { Host, Port, Name, User, Password, Params string }
type NewAPI struct { BaseURL, Token, ScriptModel, FastVideoModel, QualityVideoModel, DefaultVideoModel string }
type Video struct { MaxConcurrency int; PollInterval, TaskTimeout time.Duration }
```

`APP_ENV=production` must require administrator credentials, a 32-character-or-longer application secret, complete database values, New API URL/token, and both approved video models. Error messages name the missing variable but never include its value.

Add `github.com/stretchr/testify` for deterministic `require`/`assert` tests used throughout the backend plan.

- [ ] **Step 4: Write the repository interface and refactor tests to compile against it**

Define:

```go
type Repository interface {
    CreateProject(context.Context, domain.Project, domain.Workflow) error
    ListProjects(context.Context) ([]domain.Project, error)
    Project(context.Context, string) (domain.Project, error)
    Workflow(context.Context, string) (domain.Workflow, error)
    SaveWorkflow(context.Context, domain.Workflow) error
    ListProviders(context.Context) ([]domain.Provider, error)
    SaveProvider(context.Context, domain.Provider) error
    Provider(context.Context, string) (domain.Provider, error)
    DeleteProvider(context.Context, string) error
    SaveRun(context.Context, domain.Run) error
    Run(context.Context, string) (domain.Run, error)
    UpdateRun(context.Context, string, func(*domain.Run)) (domain.Run, error)
    MarkActiveRunsInterrupted(context.Context) error
    Close() error
}
```

Update JSON store tests to exercise the interface through a reusable `runRepositoryContract` fixture.

- [ ] **Step 5: Verify interface refactor fails before production changes**

Run: `go test ./internal/store ./internal/api ./internal/runner`

Expected: FAIL at call sites that still use the old no-context/no-error methods.

- [ ] **Step 6: Refactor JSON storage, API, runner, and server composition**

Pass request/run contexts into repository methods, return storage failures through existing API error handling, and make `api.Server` and `runner.Runner` depend on `store.Repository`. Add `InterruptedAt *time.Time` and `RunInterrupted` to the run domain so JSON restart recovery is observable.

- [ ] **Step 7: Run focused and full tests**

Run:

```powershell
go test ./internal/config ./internal/store ./internal/api ./internal/runner
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```powershell
git add internal/config internal/store internal/api/server.go internal/runner/runner.go internal/domain/types.go cmd/server/main.go
git commit -m "refactor: introduce production configuration and repository boundary"
```

---

### Task 2: MySQL Schema, Repository, and Recovery

**Files:**
- Modify: `go.mod`
- Create: `internal/store/mysql.go`
- Create: `internal/store/migrations.go`
- Create: `internal/store/migrations/001_initial.sql`
- Create: `internal/store/mysql_test.go`
- Create: `internal/store/contract_test.go`
- Create: `docker-compose.test.yml`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Consumes: `config.Database`, `store.Repository`.
- Produces: `store.OpenMySQL(context.Context, config.Database) (store.Repository, error)` and migration/recovery behavior.

- [ ] **Step 1: Write repository contract and migration tests first**

The MySQL test must call the same contract as JSON and add transaction/restart checks:

```go
func TestMySQLRepositoryContract(t *testing.T) {
    dsn := os.Getenv("TEST_MYSQL_DSN")
    if dsn == "" { t.Skip("TEST_MYSQL_DSN is required") }
    repo := openIsolatedMySQLRepository(t, dsn)
    runRepositoryContract(t, repo)
}

func TestMySQLMarksActiveRunsInterrupted(t *testing.T) {
    repo := openTestMySQLRepository(t)
    seedRun(t, repo, domain.RunRunning)
    require.NoError(t, repo.MarkActiveRunsInterrupted(context.Background()))
    got, err := repo.Run(context.Background(), testRunID)
    require.NoError(t, err)
    assert.Equal(t, domain.RunInterrupted, got.Status)
    assert.NotNil(t, got.InterruptedAt)
}
```

- [ ] **Step 2: Start MySQL and verify tests fail**

Create a test compose service using `mysql:8.0`, database `manga_drama_studio_test`, user `manga_test`, password `manga_test_password`, healthcheck `mysqladmin ping`, and host port `13307`.

Run:

```powershell
docker compose -f docker-compose.test.yml up -d --wait
$env:TEST_MYSQL_DSN='manga_test:manga_test_password@tcp(127.0.0.1:13307)/manga_drama_studio_test?charset=utf8mb4&parseTime=true&loc=UTC&multiStatements=true'
go test ./internal/store -run MySQL -v
```

Expected: FAIL because MySQL repository and migrations are absent.

- [ ] **Step 3: Add MySQL driver and initial migration**

Add `github.com/go-sql-driver/mysql` and embed migrations. `001_initial.sql` creates `schema_migrations`, `projects`, `workflows`, `providers`, `runs`, `run_nodes`, and `assets` using InnoDB, `utf8mb4`, foreign keys, indexed status/project fields, `JSON` documents, and `DATETIME(6)` timestamps.

Acquire `GET_LOCK('manga_drama_studio_migrations', 30)`, apply missing versions in transactions, and always call `RELEASE_LOCK`.

- [ ] **Step 4: Implement MySQL repository methods**

Use `database/sql`, context timeouts, `errors.Is(err, sql.ErrNoRows)` mapping to `store.ErrNotFound`, explicit transactions, and JSON marshal/unmarshal for aggregate documents. `UpdateRun` uses `SELECT ... FOR UPDATE` followed by an update in the same transaction.

- [ ] **Step 5: Select repository at startup**

`cmd/server/main.go` opens MySQL when `cfg.Database.Host != ""`; otherwise it opens JSON storage. Production configuration cannot reach the JSON branch. Call `MarkActiveRunsInterrupted` before creating the runner and defer `Close`.

- [ ] **Step 6: Verify MySQL contract and all Go tests**

Run:

```powershell
go test ./internal/store -run MySQL -v
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add go.mod go.sum internal/store cmd/server/main.go docker-compose.test.yml
git commit -m "feat: add durable mysql repository"
```

---

### Task 3: Administrator Sessions and Encrypted Provider Bootstrap

**Files:**
- Create: `internal/auth/session.go`
- Create: `internal/auth/session_test.go`
- Modify: `internal/api/secret.go`
- Create: `internal/api/secret_test.go`
- Modify: `internal/api/server.go`
- Create: `internal/api/auth_test.go`
- Modify: `internal/domain/types.go`
- Modify: `web/src/api/client.ts`
- Modify: `web/src/stores/studio.ts`
- Create: `web/src/views/LoginView.vue`
- Modify: `web/src/App.vue`

**Interfaces:**
- Produces: signed admin sessions, protected APIs/assets, provider decryption, and New API bootstrap settings.
- Session cookie name: `manga_studio_session`; lifetime: 12 hours.

- [ ] **Step 1: Write failing session and secret tests**

```go
func TestSessionRoundTripAndTamperRejection(t *testing.T) {
    manager := NewSessionManager("01234567890123456789012345678901", 12*time.Hour)
    token, err := manager.Issue("admin", time.Unix(1000, 0))
    require.NoError(t, err)
    assert.Equal(t, "admin", requireSession(t, manager, token, time.Unix(1001, 0)))
    _, err = manager.Verify(token+"x", time.Unix(1001, 0))
    require.Error(t, err)
}

func TestSecretCodecDecryptsAndRejectsWrongKey(t *testing.T) {
    encoded, err := newSecretCodec("first-secret").Encrypt("sk-private")
    require.NoError(t, err)
    value, err := newSecretCodec("first-secret").Decrypt(encoded)
    require.NoError(t, err)
    assert.Equal(t, "sk-private", value)
    _, err = newSecretCodec("second-secret").Decrypt(encoded)
    require.Error(t, err)
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./internal/auth ./internal/api -run 'Session|Secret|Auth' -v`

Expected: FAIL because session manager, decrypt, and auth routes do not exist.

- [ ] **Step 3: Implement signed session manager and route middleware**

Encode `{username, expiresAt}` as base64url JSON and sign it with HMAC-SHA256. Verify signature with `hmac.Equal`. Add login/session/logout handlers, HttpOnly and SameSite=Strict cookies, production `Secure` cookies, and an in-memory per-IP failed-login limiter of 10 attempts per 10 minutes.

Require authentication for all project/provider/run/asset paths. Keep `/healthz`, `/api/v1/auth/login`, and `/api/v1/auth/session` reachable for bootstrap; unauthenticated session returns `{authenticated:false}`.

- [ ] **Step 4: Extend provider settings and safe updates**

Add:

```go
type ProviderModels struct {
    Script string `json:"script"`
    VideoFast string `json:"videoFast"`
    VideoQuality string `json:"videoQuality"`
    VideoDefault string `json:"videoDefault"`
}
```

Store it on `domain.Provider`. Add `PUT /api/v1/providers/{id}`; empty token preserves ciphertext. Implement `secretCodec.Decrypt`. Startup creates or updates one enabled `new-api` provider from environment values without returning its secret.

- [ ] **Step 5: Add frontend login/session flow**

Set fetch `credentials: 'same-origin'`. Add typed `session`, `login`, and `logout` calls. Store authentication state in Pinia, render `LoginView` when unauthenticated, and clear in-memory project/provider/run data on logout.

- [ ] **Step 6: Verify backend and frontend**

Run:

```powershell
go test ./internal/auth ./internal/api ./...
pnpm --dir web typecheck
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add internal/auth internal/api internal/domain/types.go cmd/server/main.go web/src
git commit -m "feat: protect studio with admin authentication"
```

---

### Task 4: DeepSeek New API Client and Structured Planning

**Files:**
- Create: `internal/newapi/client.go`
- Create: `internal/newapi/client_test.go`
- Create: `internal/generation/types.go`
- Create: `internal/generation/planner.go`
- Create: `internal/generation/planner_test.go`
- Create: `internal/runner/production_executor.go`
- Create: `internal/runner/production_executor_test.go`
- Modify: `internal/api/server.go`
- Modify: `cmd/server/main.go`

**Interfaces:**
- Produces: `newapi.Client.ChatJSON`, validated generation structs, and real execution for `script`, `character`, and `storyboard` nodes.

- [ ] **Step 1: Write failing New API request/response tests**

Use `httptest.Server` to assert method/path/auth/model and return a fixture:

```go
func TestChatJSONUsesApprovedModelAndBearerToken(t *testing.T) {
    upstream := newRecordedServer(t, func(r *http.Request, body chatRequest) {
        assert.Equal(t, "/v1/chat/completions", r.URL.Path)
        assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
        assert.Equal(t, "deepseek-v4-pro", body.Model)
        assert.Equal(t, "json_object", body.ResponseFormat.Type)
    }, `{"choices":[{"message":{"content":"{\"title\":\"测试\"}"}}]}`)
    var out struct{ Title string `json:"title"` }
    require.NoError(t, New(upstream.URL, "sk-test", upstream.Client()).ChatJSON(context.Background(), "deepseek-v4-pro", []Message{{Role:"user", Content:"write"}}, &out))
    assert.Equal(t, "测试", out.Title)
}
```

Add cases for non-2xx response redaction, empty choices, fenced JSON, invalid JSON, cancellation, and response body size limits.

- [ ] **Step 2: Verify client tests fail**

Run: `go test ./internal/newapi -v`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the typed New API client**

Normalize base URLs by removing trailing `/v1` and `/`. Send bearer auth, `Content-Type`, finite timeouts, `response_format:{type:"json_object"}`, and bounded response decoding. Error values expose safe status/code/request ID but implement `Error()` without token, headers, or full response bodies.

- [ ] **Step 4: Write failing planner validation tests**

Define `Script`, `CharacterSheet`, `Storyboard`, and `Shot`. Tests assert required title/scenes/characters, unique ordered shot IDs, duration bounds, ratio allowlist, non-empty prompts/subtitles, and rejection above 12 shots.

- [ ] **Step 5: Implement planner prompts and validators**

Planner methods accept upstream node outputs and node config, use DeepSeek V4 Pro, request Chinese JSON matching the exact structs, and validate before returning. Prompts preserve character appearance and include dialogue in each Seedance shot prompt.

- [ ] **Step 6: Wire production text execution**

Create a production executor with planner/video/composer dependencies. `story-input` remains deterministic. `script`, `character`, and `storyboard` call planner methods and return typed JSON-compatible outputs. A startup `APP_EXECUTOR=mock` option is allowed only outside production; production always uses the real executor.

- [ ] **Step 7: Verify**

Run:

```powershell
go test ./internal/newapi ./internal/generation ./internal/runner -v
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```powershell
git add internal/newapi internal/generation internal/runner cmd/server/main.go internal/api/server.go
git commit -m "feat: generate structured scripts with deepseek"
```

---

### Task 5: Seedance Tasks, Durable Assets, and Resume

**Files:**
- Modify: `internal/newapi/client.go`
- Modify: `internal/newapi/client_test.go`
- Create: `internal/assets/service.go`
- Create: `internal/assets/service_test.go`
- Modify: `internal/domain/types.go`
- Modify: `internal/store/repository.go`
- Modify: `internal/store/store.go`
- Modify: `internal/store/mysql.go`
- Modify: `internal/runner/production_executor.go`
- Create: `internal/runner/video_executor_test.go`
- Modify: `internal/api/server.go`

**Interfaces:**
- Produces: `SubmitVideo`, `VideoTask`, asset registration/opening, per-shot durable task state, authenticated asset streaming, cancellation, and resume.

- [ ] **Step 1: Write failing Seedance client tests**

Assert submission payload and polling:

```go
func TestSubmitVideoRequestsAudioAndFastModel(t *testing.T) {
    client := fixtureClient(t, func(r *http.Request, body VideoRequest) {
        assert.Equal(t, "/v1/video/generations", r.URL.Path)
        assert.Equal(t, "doubao-seedance-2-0-fast-260128", body.Model)
        assert.Equal(t, true, body.Metadata["generate_audio"])
        assert.Equal(t, false, body.Metadata["watermark"])
    }, `{"id":"task-public-1","status":"queued"}`)
    task, err := client.SubmitVideo(context.Background(), approvedVideoRequest())
    require.NoError(t, err)
    assert.Equal(t, "task-public-1", task.ID)
}
```

Add polling fixtures for queued/running/succeeded/failed, cancellation, timeout, and missing result URL.

- [ ] **Step 2: Write failing asset download tests**

Tests require HTTPS/HTTP URLs, reject non-video MIME, cap download size, write via `.part`, compute SHA-256, atomically rename, and clean partial files on cancellation.

- [ ] **Step 3: Verify tests fail**

Run: `go test ./internal/newapi ./internal/assets ./internal/runner -run 'Video|Asset|Resume' -v`

Expected: FAIL because task and asset services are absent.

- [ ] **Step 4: Implement video methods and asset service**

Use `POST /v1/video/generations` and `GET /v1/video/generations/{id}`. Treat `queued`/`running` as nonterminal, `completed`/`succeeded` as success, and `failed`/`cancelled` as terminal errors. Asset paths are generated server-side as `<project>/<run>/<node>/<asset-id>.mp4`; callers never provide filesystem paths.

- [ ] **Step 5: Extend repository and schema for assets/node attempts**

Add repository methods:

```go
SaveNodeAttempt(context.Context, string, domain.NodeRun) error
SaveAssetWithNode(context.Context, domain.Asset, string, domain.NodeRun) error
Asset(context.Context, string) (domain.Asset, error)
ListRunAssets(context.Context, string) ([]domain.Asset, error)
```

Persist `UpstreamTaskID`, `Attempt`, shot identity, outputs, and asset checksum. Add contract tests for atomic node+asset completion.

- [ ] **Step 6: Implement concurrent shot execution and reuse**

Use `errgroup`-equivalent standard-library goroutines with a buffered semaphore sized by `VIDEO_MAX_CONCURRENCY`, cancellation-safe result collection, and deterministic shot ordering. Before submission, reuse an existing successful asset only when its metadata, file, size, and checksum agree.

- [ ] **Step 7: Add resume and authenticated asset routes**

Add `POST /api/v1/runs/{id}/resume`, `GET /api/v1/runs/{id}/assets`, and `GET /api/v1/assets/{id}` with range-request support through `http.ServeContent`. Resume is allowed only for failed, cancelled, or interrupted runs.

- [ ] **Step 8: Verify**

Run:

```powershell
go test ./internal/newapi ./internal/assets ./internal/runner ./internal/api -v
go test ./...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```powershell
git add internal/newapi internal/assets internal/domain internal/store internal/runner internal/api
git commit -m "feat: generate and resume seedance video shots"
```

---

### Task 6: FFmpeg Final Composition

**Files:**
- Create: `internal/media/ffmpeg.go`
- Create: `internal/media/ffmpeg_test.go`
- Create: `internal/media/subtitle.go`
- Create: `internal/media/subtitle_test.go`
- Modify: `internal/runner/production_executor.go`
- Modify: `internal/domain/types.go`

**Interfaces:**
- Produces: `media.Composer.Check`, `media.Composer.Compose`, UTF-8 subtitle generation, final asset registration, and bounded diagnostics.

- [ ] **Step 1: Write failing subtitle tests**

Use exact SRT output assertions for multi-shot timings, escaping, empty subtitles, and millisecond formatting:

```go
func TestBuildSRTUsesCumulativeShotTimings(t *testing.T) {
    got := BuildSRT([]Cue{{Text:"第一句", Duration:5*time.Second}, {Text:"第二句", Duration:4*time.Second}})
    assert.Equal(t, "1\n00:00:00,000 --> 00:00:05,000\n第一句\n\n2\n00:00:05,000 --> 00:00:09,000\n第二句\n", got)
}
```

- [ ] **Step 2: Write failing FFmpeg integration tests**

Generate two one-second synthetic MP4 fixtures with FFmpeg: one with sine-wave audio and one without audio. Compose them, run `ffprobe -of json`, and assert H.264 video, AAC audio, requested dimensions, ordered duration, and final-file existence.

- [ ] **Step 3: Verify media tests fail**

Run: `go test ./internal/media -v`

Expected: FAIL because composer and subtitle packages are absent. If FFmpeg is missing locally, install it before continuing; do not skip the integration test used for completion evidence.

- [ ] **Step 4: Implement bounded command execution and normalization**

`Composer.Check` runs `ffmpeg -version` and `ffprobe -version`. `Compose` probes audio streams, normalizes clips to 30 FPS H.264/AAC with scale/pad filters, adds `anullsrc` for silent clips, creates an ordered concat list, burns subtitles using the bundled Chinese font, writes `final.mp4.part`, validates it with ffprobe, then atomically renames it.

Capture only the final 32 KiB of stderr and wrap it in a typed redacted error.

- [ ] **Step 5: Wire compose node and final asset**

The `compose` node consumes ordered clip assets and storyboard cues, emits one `video` asset ID/URL, and stores final asset metadata transactionally with node completion.

- [ ] **Step 6: Verify**

Run:

```powershell
go test ./internal/media -v
go test ./internal/runner ./internal/api ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add internal/media internal/runner internal/domain internal/store
git commit -m "feat: compose generated shots into final mp4"
```

---

### Task 7: Production Workflow and Vue Run Experience

**Files:**
- Modify: `internal/workflow/catalog.go`
- Modify: `internal/workflow/example.go`
- Modify: `internal/workflow/validate_test.go`
- Modify: `web/package.json`
- Modify: `web/pnpm-lock.yaml`
- Create: `web/vitest.config.ts`
- Create: `web/src/components/GenerationInspector.vue`
- Create: `web/src/components/RunDetails.vue`
- Create: `web/src/components/ProviderSettings.vue`
- Create: `web/src/components/__tests__/GenerationInspector.test.ts`
- Create: `web/src/components/__tests__/RunDetails.test.ts`
- Modify: `web/src/components/InspectorPanel.vue`
- Modify: `web/src/components/ProviderDrawer.vue`
- Modify: `web/src/components/RunDock.vue`
- Modify: `web/src/stores/studio.ts`
- Modify: `web/src/api/client.ts`
- Modify: `web/src/types/index.ts`
- Modify: `web/src/App.vue`
- Modify: `web/src/style.css`

**Interfaces:**
- Produces: a default real pipeline and typed UI controls/results matching backend request and response types.

- [ ] **Step 1: Write failing workflow validation tests**

Update the default graph contract to `story-input -> script -> character/storyboard -> video -> compose`; video requires storyboard plus character inputs and emits `video-sequence`. Compose requires `video-sequence` and emits final `video`. Assert invalid model mode, ratio, resolution, duration, and shot count are rejected server-side.

- [ ] **Step 2: Write failing frontend tests**

Add Vitest and Vue Test Utils. Tests mount structured controls and assert:

- fast mode serializes `videoModelMode: "fast"`;
- quality mode serializes `videoModelMode: "quality"`;
- shot count cannot exceed 12;
- run details order shots, show retry only for failed/interrupted runs, and expose final asset playback/download only after composition succeeds;
- provider settings never render token values returned from APIs.

- [ ] **Step 3: Verify tests fail**

Run:

```powershell
go test ./internal/workflow -v
pnpm --dir web test --run
```

Expected: FAIL because the production graph and UI components are absent.

- [ ] **Step 4: Implement production workflow definitions**

Replace mock image/voice routing in the default example with a typed video node. Preserve old saved workflows by retaining catalog definitions for legacy nodes but mark them as legacy in metadata; new projects receive only the production graph.

- [ ] **Step 5: Implement structured inspector and provider controls**

Render typed controls for story prompt, episode duration, shot count, ratio, resolution, fast/quality model, subtitle toggle, and retry policy. Keep raw JSON available only in a collapsed advanced panel. Provider settings edit base URL/model names and accept a write-only replacement token.

- [ ] **Step 6: Implement run and asset experience**

Show stage status, safe error messages, shot task IDs, previews, retry/resume, composition progress, final `<video>` playback, and authenticated download. Present a confirmation with model mode and shot count before starting paid Seedance tasks.

- [ ] **Step 7: Verify frontend and backend**

Run:

```powershell
go test ./internal/workflow ./internal/api ./...
pnpm --dir web test --run
pnpm --dir web typecheck
pnpm --dir web build
```

Expected: PASS.

- [ ] **Step 8: Commit**

```powershell
git add internal/workflow web
git commit -m "feat: add production generation workflow experience"
```

---

### Task 8: Docker, Baota Guide, and Release Verification

**Files:**
- Modify: `.env.example`
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `Makefile`
- Modify: `README.md`
- Create: `docs/baota-deployment.md`
- Modify: `openspec/changes/production-newapi-mysql-pipeline/tasks.md`

**Interfaces:**
- Produces: a deployable image, external-host compose configuration, exact Baota MySQL/proxy instructions, and recorded verification evidence.

- [ ] **Step 1: Write deployment assertions before Docker changes**

Add `scripts/verify-compose.ps1` that runs `docker compose config --format json` and exits nonzero unless it finds:

- one production service named `studio`;
- port `8080`;
- `/app/data` volume;
- `host.docker.internal:host-gateway`;
- required environment variable references;
- no production MySQL service;
- a healthcheck at `/healthz`.

Run: `powershell -ExecutionPolicy Bypass -File scripts/verify-compose.ps1`

Expected: FAIL against the current compose file.

- [ ] **Step 2: Update runtime image and compose**

Install FFmpeg, ffprobe, CA certificates, `font-noto-cjk`, and timezone data in the Alpine runtime. Keep the non-root user and ensure `/app/data/assets` ownership. Add `.env` loading, healthcheck, persistent volume, `extra_hosts`, restart policy, and production configuration to compose.

Use Chinese-accessible build mirrors through overridable build args (`GOPROXY` and Alpine repository mirror) while retaining official defaults.

- [ ] **Step 3: Write exact Baota deployment documentation**

Document:

- creating database `manga_drama_studio` and user `manga_studio`;
- allowing the Docker subnet/host and granting only this database;
- generating `APP_SECRET_KEY` and a new New API token;
- setting New API/model environment values;
- building and starting compose;
- health/log/database checks;
- Baota reverse proxy, HTTPS, websocket/SSE buffering settings;
- backups for MySQL and `/app/data`;
- token rotation and rollback.

- [ ] **Step 4: Run fresh release verification**

Run:

```powershell
go test ./...
pnpm --dir web test --run
pnpm --dir web typecheck
pnpm --dir web build
powershell -ExecutionPolicy Bypass -File scripts/verify-compose.ps1
docker compose build
docker compose up -d
curl.exe -f http://127.0.0.1:8080/healthz
docker compose ps
```

Expected: every command exits 0, health returns `{"status":"ok"}`, and `studio` is healthy.

- [ ] **Step 5: Perform a safe real-provider smoke test**

Using a newly generated New API token and the test project, run one DeepSeek planning request and one 5-second Seedance Fast shot, poll it, compose it, and confirm the final asset plays. Record task/run IDs but not credentials in `docs/verification-2026-07-19.md`.

- [ ] **Step 6: Mark OpenSpec tasks complete only where evidence exists**

Update each checkbox in `openspec/changes/production-newapi-mysql-pipeline/tasks.md` only after its corresponding command or smoke check succeeds.

- [ ] **Step 7: Commit release configuration and evidence**

```powershell
git add .env.example Dockerfile docker-compose.yml Makefile README.md docs scripts openspec/changes/production-newapi-mysql-pipeline/tasks.md
git commit -m "docs: add baota production deployment and verification"
```

---

## Final Review Checklist

- [ ] Compare every OpenSpec requirement and scenario to a passing test or recorded smoke result.
- [ ] Confirm no plaintext database, administrator, application-secret, or New API credential is tracked or logged.
- [ ] Confirm the final Git diff contains no generated media, `.env`, database files, binaries, or frontend build output.
- [ ] Run `git status --short`, `git log --oneline --decorate -10`, and the full release verification commands once more before declaring completion.
