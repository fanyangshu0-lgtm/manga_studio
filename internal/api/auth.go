package api

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/domain"
)

const sessionCookieName = "manga_studio_session"

type loginLimiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
}

func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	return &loginLimiter{max: max, window: window, failures: map[string][]time.Time{}}
}

func (l *loginLimiter) blocked(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.trimLocked(key, now)
	return len(l.failures[key]) >= l.max
}

func (l *loginLimiter) fail(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.trimLocked(key, now)
	l.failures[key] = append(l.failures[key], now)
}

func (l *loginLimiter) clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

func (l *loginLimiter) trimLocked(key string, now time.Time) {
	cutoff := now.Add(-l.window)
	items := l.failures[key]
	first := 0
	for first < len(items) && items[first].Before(cutoff) {
		first++
	}
	if first > 0 {
		l.failures[key] = append([]time.Time(nil), items[first:]...)
	}
}

func (s *Server) authRoutes(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 1 {
		writeError(w, http.StatusNotFound, "not_found", "认证接口不存在", nil)
		return
	}
	switch parts[0] {
	case "login":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		s.login(w, r)
	case "session":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		username, ok := s.sessionUsername(r)
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": ok, "username": username})
	case "logout":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.admin.SecureCookie, SameSite: http.SameSiteStrictMode})
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
	default:
		writeError(w, http.StatusNotFound, "not_found", "认证接口不存在", nil)
	}
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	key := requestIP(r)
	now := time.Now().UTC()
	if s.limiter.blocked(key, now) {
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", "登录失败次数过多，请稍后再试", nil)
		return
	}
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	usernameMatch := subtle.ConstantTimeCompare([]byte(input.Username), []byte(s.admin.Username)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(input.Password), []byte(s.admin.Password)) == 1
	if !usernameMatch || !passwordMatch {
		s.limiter.fail(key, now)
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误", nil)
		return
	}
	token, err := s.sessions.Issue(s.admin.Username, now)
	if err != nil {
		writeInternal(w, err)
		return
	}
	s.limiter.clear(key)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", MaxAge: int(s.admin.SessionTTL.Seconds()),
		Expires: now.Add(s.admin.SessionTTL), HttpOnly: true, Secure: s.admin.SecureCookie, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "username": s.admin.Username})
}

func (s *Server) authenticated(r *http.Request) bool {
	_, ok := s.sessionUsername(r)
	return ok
}

func (s *Server) sessionUsername(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	username, err := s.sessions.Verify(cookie.Value, time.Now().UTC())
	return username, err == nil
}

func requestIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func (s *Server) BootstrapProvider(ctx context.Context, cfg config.NewAPI) error {
	if cfg.BaseURL == "" || cfg.Token == "" {
		return nil
	}
	providers, err := s.store.ListProviders(ctx)
	if err != nil {
		return err
	}
	var provider domain.Provider
	for _, item := range providers {
		if item.Kind == "new-api" {
			provider = item
			break
		}
	}
	now := time.Now().UTC()
	if provider.ID == "" {
		provider = domain.Provider{ID: domain.NewID("pvd"), Kind: "new-api", Name: "New API", CreatedAt: now}
	}
	ciphertext, err := s.secret.Encrypt(cfg.Token)
	if err != nil {
		return err
	}
	provider.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	provider.Capabilities = []string{"script", "video"}
	provider.Models = domain.ProviderModels{Script: cfg.ScriptModel, VideoFast: cfg.FastVideoModel, VideoQuality: cfg.QualityVideoModel, VideoDefault: cfg.DefaultVideoModel}
	provider.Weight = 1
	provider.Enabled = true
	provider.SecretCiphertext = ciphertext
	provider.SecretHint = secretHint(cfg.Token)
	provider.UpdatedAt = now
	return s.store.SaveProvider(ctx, provider)
}
