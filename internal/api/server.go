package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"manga-drama-studio/internal/auth"
	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/domain"
	"manga-drama-studio/internal/runner"
	"manga-drama-studio/internal/store"
	"manga-drama-studio/internal/workflow"
)

type Server struct {
	store    store.Repository
	runner   *runner.Runner
	broker   *runner.Broker
	secret   secretCodec
	webDir   string
	sessions *auth.SessionManager
	admin    config.Admin
	limiter  *loginLimiter
}

func New(data store.Repository, runtime *runner.Runner, broker *runner.Broker, cfg config.Config) *Server {
	return &Server{
		store: data, runner: runtime, broker: broker, secret: newSecretCodec(cfg.SecretKey), webDir: cfg.WebDir,
		sessions: auth.NewSessionManager(cfg.SecretKey, cfg.Admin.SessionTTL), admin: cfg.Admin,
		limiter: newLoginLimiter(cfg.Admin.LoginMaxAttempts, cfg.Admin.LoginWindow),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", s.routeAPI)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if s.webDir != "" {
		mux.HandleFunc("/", s.serveWeb)
	}
	return s.withMiddleware(mux)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func (s *Server) routeAPI(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/v1/"))
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在", nil)
		return
	}
	if parts[0] == "auth" {
		s.authRoutes(w, r, parts[1:])
		return
	}
	if !s.authenticated(r) {
		writeError(w, http.StatusUnauthorized, "authentication_required", "请先登录", nil)
		return
	}
	switch parts[0] {
	case "catalog":
		if r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "nodes" {
			writeJSON(w, http.StatusOK, workflow.Catalog())
			return
		}
	case "projects":
		s.projects(w, r, parts[1:])
		return
	case "providers":
		s.providers(w, r, parts[1:])
		return
	case "runs":
		s.runs(w, r, parts[1:])
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "接口不存在", nil)
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			projects, err := s.store.ListProjects(r.Context())
			if err != nil {
				writeInternal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, projects)
		case http.MethodPost:
			var input struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			input.Name = strings.TrimSpace(input.Name)
			if input.Name == "" {
				writeError(w, http.StatusBadRequest, "name_required", "项目名称不能为空", nil)
				return
			}
			now := time.Now().UTC()
			project := domain.Project{ID: domain.NewID("prj"), Name: input.Name, Description: strings.TrimSpace(input.Description), CreatedAt: now, UpdatedAt: now}
			graph := workflow.Example(project.ID)
			if err := s.store.CreateProject(r.Context(), project, graph); err != nil {
				writeInternal(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"project": project, "workflow": graph})
		default:
			methodNotAllowed(w)
		}
		return
	}
	projectID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		project, err := s.store.Project(r.Context(), projectID)
		if handleStoreError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, project)
		return
	}
	if len(parts) == 2 && parts[1] == "workflow" {
		switch r.Method {
		case http.MethodGet:
			graph, err := s.store.Workflow(r.Context(), projectID)
			if handleStoreError(w, err) {
				return
			}
			writeJSON(w, http.StatusOK, graph)
		case http.MethodPut:
			var graph domain.Workflow
			if !decodeJSON(w, r, &graph) {
				return
			}
			current, err := s.store.Workflow(r.Context(), projectID)
			if handleStoreError(w, err) {
				return
			}
			graph.ID = current.ID
			graph.ProjectID = projectID
			graph.Revision = current.Revision + 1
			graph.UpdatedAt = time.Now().UTC()
			if issues := workflow.Validate(graph); len(issues) > 0 {
				writeError(w, http.StatusUnprocessableEntity, "invalid_workflow", "工作流校验失败", issues)
				return
			}
			if err := s.store.SaveWorkflow(r.Context(), graph); err != nil {
				writeInternal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, graph)
		default:
			methodNotAllowed(w)
		}
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "项目资源不存在", nil)
}

func (s *Server) providers(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			providers, err := s.store.ListProviders(r.Context())
			if err != nil {
				writeInternal(w, err)
				return
			}
			writeJSON(w, http.StatusOK, providers)
		case http.MethodPost:
			var input struct {
				Name, Kind, BaseURL, Token string
				Capabilities               []string
				Models                     domain.ProviderModels
				Weight                     int
			}
			if !decodeJSON(w, r, &input) {
				return
			}
			if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Kind) == "" || strings.TrimSpace(input.Token) == "" {
				writeError(w, http.StatusBadRequest, "invalid_provider", "名称、类型和 Token 必填", nil)
				return
			}
			ciphertext, err := s.secret.Encrypt(input.Token)
			if err != nil {
				writeError(w, http.StatusBadRequest, "secret_key_required", err.Error(), nil)
				return
			}
			if input.Weight <= 0 {
				input.Weight = 1
			}
			now := time.Now().UTC()
			provider := domain.Provider{ID: domain.NewID("pvd"), Name: strings.TrimSpace(input.Name), Kind: input.Kind, BaseURL: input.BaseURL, Capabilities: input.Capabilities, Models: input.Models, Weight: input.Weight, Enabled: true, SecretCiphertext: ciphertext, SecretHint: secretHint(input.Token), CreatedAt: now, UpdatedAt: now}
			if err := s.store.SaveProvider(r.Context(), provider); err != nil {
				writeInternal(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, provider)
		default:
			methodNotAllowed(w)
		}
		return
	}
	id := parts[0]
	switch r.Method {
	case http.MethodPut:
		var input struct {
			Name, Kind, BaseURL, Token string
			Capabilities               []string
			Models                     domain.ProviderModels
			Weight                     int
			Enabled                    *bool `json:"enabled"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		provider, err := s.store.Provider(r.Context(), id)
		if handleStoreError(w, err) {
			return
		}
		if strings.TrimSpace(input.Name) != "" {
			provider.Name = strings.TrimSpace(input.Name)
		}
		if strings.TrimSpace(input.Kind) != "" {
			provider.Kind = strings.TrimSpace(input.Kind)
		}
		if strings.TrimSpace(input.BaseURL) != "" {
			provider.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
		}
		if input.Capabilities != nil {
			provider.Capabilities = input.Capabilities
		}
		provider.Models = input.Models
		if input.Weight > 0 {
			provider.Weight = input.Weight
		}
		if input.Enabled != nil {
			provider.Enabled = *input.Enabled
		}
		if input.Token != "" {
			provider.SecretCiphertext, err = s.secret.Encrypt(input.Token)
			if err != nil {
				writeInternal(w, err)
				return
			}
			provider.SecretHint = secretHint(input.Token)
		}
		provider.UpdatedAt = time.Now().UTC()
		if err := s.store.SaveProvider(r.Context(), provider); err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, provider)
	case http.MethodPatch:
		var input struct {
			Enabled *bool `json:"enabled"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		if input.Enabled == nil {
			writeError(w, http.StatusBadRequest, "enabled_required", "enabled 字段必填", nil)
			return
		}
		provider, err := s.store.Provider(r.Context(), id)
		if handleStoreError(w, err) {
			return
		}
		provider.Enabled = *input.Enabled
		provider.UpdatedAt = time.Now().UTC()
		if err := s.store.SaveProvider(r.Context(), provider); err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusOK, provider)
	case http.MethodDelete:
		if err := s.store.DeleteProvider(r.Context(), id); handleStoreError(w, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 && r.Method == http.MethodPost {
		var input struct {
			ProjectID string `json:"projectId"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		graph, err := s.store.Workflow(r.Context(), input.ProjectID)
		if handleStoreError(w, err) {
			return
		}
		if issues := workflow.Validate(graph); len(issues) > 0 {
			writeError(w, http.StatusUnprocessableEntity, "invalid_workflow", "工作流校验失败", issues)
			return
		}
		nodes := map[string]domain.NodeRun{}
		for _, node := range graph.Nodes {
			nodes[node.ID] = domain.NodeRun{NodeID: node.ID, NodeName: node.Name, Status: domain.RunQueued, Message: "等待执行"}
		}
		run := domain.Run{ID: domain.NewID("run"), ProjectID: input.ProjectID, Status: domain.RunQueued, Message: "任务已入队", Workflow: graph, Nodes: nodes, CreatedAt: time.Now().UTC()}
		if err := s.runner.Start(run); err != nil {
			writeInternal(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, run)
		return
	}
	if len(parts) == 0 {
		methodNotAllowed(w)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		run, err := s.store.Run(r.Context(), id)
		if handleStoreError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, run)
		return
	}
	if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
		if err := s.runner.Cancel(id); handleStoreError(w, err) {
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "cancelling"})
		return
	}
	if len(parts) == 2 && parts[1] == "events" && r.Method == http.MethodGet {
		s.events(w, r, id)
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "运行任务不存在", nil)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request, runID string) {
	channel, unsubscribe := s.broker.Subscribe(runID)
	defer unsubscribe()
	run, err := s.store.Run(r.Context(), runID)
	if handleStoreError(w, err) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "服务器不支持事件流", nil)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	writeEvent(w, domain.RunEvent{Type: "snapshot", RunID: run.ID, Status: run.Status, Progress: run.Progress, Message: run.Message, Timestamp: time.Now().UTC()})
	flusher.Flush()
	if terminal(run.Status) {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-channel:
			writeEvent(w, event)
			flusher.Flush()
			if event.Type == "run" && terminal(event.Status) {
				return
			}
		case <-time.After(20 * time.Second):
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.webDir, filepath.Clean(r.URL.Path))
	if info, err := filepath.Abs(path); err == nil {
		root, _ := filepath.Abs(s.webDir)
		if strings.HasPrefix(info, root) {
			if stat, err := filepath.Glob(info); err == nil && len(stat) > 0 {
				http.ServeFile(w, r, info)
				return
			}
		}
	}
	http.ServeFile(w, r, filepath.Join(s.webDir, "index.html"))
}

func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	if len(raw) == 1 && raw[0] == "" {
		return nil
	}
	return raw
}
func terminal(status domain.RunStatus) bool {
	return status == domain.RunSucceeded || status == domain.RunFailed || status == domain.RunCancelled || status == domain.RunInterrupted
}
func writeEvent(w io.Writer, event domain.RunEvent) {
	data, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "请求 JSON 无效: "+err.Error(), nil)
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message, "details": details}})
}
func writeInternal(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "服务器内部错误", nil)
}
func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "请求方法不支持", nil)
}
func handleStoreError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "资源不存在", nil)
	} else {
		writeInternal(w, err)
	}
	return true
}
