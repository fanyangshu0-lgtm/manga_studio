package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"manga-drama-studio/internal/config"
	"manga-drama-studio/internal/runner"
	"manga-drama-studio/internal/store"
)

func TestAdminLoginProtectsStudioAPI(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "studio.json"))
	require.NoError(t, err)
	broker := runner.NewBroker()
	server := New(repository, runner.New(repository, runner.MockExecutor{}, broker), broker, config.Config{
		SecretKey: "01234567890123456789012345678901",
		Admin:     config.Admin{Username: "admin", Password: "strong-password", SessionTTL: 12 * time.Hour, LoginMaxAttempts: 10, LoginWindow: 10 * time.Minute},
	})

	protected := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	protectedResult := httptest.NewRecorder()
	server.Handler().ServeHTTP(protectedResult, protected)
	assert.Equal(t, http.StatusUnauthorized, protectedResult.Code)

	login := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"strong-password"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginResult, login)
	require.Equal(t, http.StatusOK, loginResult.Code)
	require.NotEmpty(t, loginResult.Result().Cookies())
	assert.True(t, loginResult.Result().Cookies()[0].HttpOnly)

	protected = httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	protected.AddCookie(loginResult.Result().Cookies()[0])
	protectedResult = httptest.NewRecorder()
	server.Handler().ServeHTTP(protectedResult, protected)
	assert.Equal(t, http.StatusOK, protectedResult.Code)
}

func TestSessionEndpointAllowsUnauthenticatedBootstrap(t *testing.T) {
	repository, err := store.Open(filepath.Join(t.TempDir(), "studio.json"))
	require.NoError(t, err)
	broker := runner.NewBroker()
	server := New(repository, runner.New(repository, runner.MockExecutor{}, broker), broker, config.Config{
		SecretKey: "01234567890123456789012345678901",
		Admin:     config.Admin{SessionTTL: 12 * time.Hour, LoginMaxAttempts: 10, LoginWindow: 10 * time.Minute},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"authenticated":false`)
}
