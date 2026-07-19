package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromLookupUsesApprovedDefaults(t *testing.T) {
	cfg, err := LoadFromLookup(func(string) string { return "" })

	require.NoError(t, err)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, ":8080", cfg.Address)
	assert.Equal(t, "./web/dist", cfg.WebDir)
	assert.Equal(t, "./data/assets", cfg.AssetDir)
	assert.Equal(t, "deepseek-v4-pro", cfg.NewAPI.ScriptModel)
	assert.Equal(t, "doubao-seedance-2-0-fast-260128", cfg.NewAPI.FastVideoModel)
	assert.Equal(t, "doubao-seedance-2-0-260128", cfg.NewAPI.QualityVideoModel)
	assert.Equal(t, "doubao-seedance-2-0-fast-260128", cfg.NewAPI.DefaultVideoModel)
	assert.Equal(t, 2, cfg.Video.MaxConcurrency)
	assert.Equal(t, 5*time.Second, cfg.Video.PollInterval)
	assert.Equal(t, 30*time.Minute, cfg.Video.TaskTimeout)
}

func TestLoadFromLookupProductionRequiresSecretsAndDatabase(t *testing.T) {
	values := map[string]string{"APP_ENV": "production"}

	_, err := LoadFromLookup(func(key string) string { return values[key] })

	require.Error(t, err)
	assert.ErrorContains(t, err, "APP_SECRET_KEY")
	assert.NotContains(t, err.Error(), "DATABASE_PASSWORD=")
}

func TestLoadFromLookupParsesProductionConfiguration(t *testing.T) {
	values := map[string]string{
		"APP_ENV":                 "production",
		"APP_SECRET_KEY":          "01234567890123456789012345678901",
		"APP_ADMIN_USERNAME":      "admin",
		"APP_ADMIN_PASSWORD":      "strong-password",
		"DATABASE_HOST":           "host.docker.internal",
		"DATABASE_PORT":           "3306",
		"DATABASE_NAME":           "manga_drama_studio",
		"DATABASE_USER":           "manga_studio",
		"DATABASE_PASSWORD":       "db-secret",
		"NEW_API_BASE_URL":        "http://host.docker.internal:3000",
		"NEW_API_TOKEN":           "sk-new-token",
		"VIDEO_MAX_CONCURRENCY":   "4",
		"VIDEO_POLL_INTERVAL":     "3s",
		"VIDEO_TASK_TIMEOUT":      "45m",
		"VIDEO_DEFAULT_MODEL":     "doubao-seedance-2-0-260128",
		"APP_SESSION_SECURE":      "true",
		"APP_SESSION_TTL":         "8h",
		"APP_LOGIN_MAX_ATTEMPTS":  "7",
		"APP_LOGIN_WINDOW":        "15m",
		"APP_MAX_DOWNLOAD_BYTES":  "1073741824",
	}

	cfg, err := LoadFromLookup(func(key string) string { return values[key] })

	require.NoError(t, err)
	assert.True(t, cfg.Admin.SecureCookie)
	assert.Equal(t, 8*time.Hour, cfg.Admin.SessionTTL)
	assert.Equal(t, 7, cfg.Admin.LoginMaxAttempts)
	assert.Equal(t, 15*time.Minute, cfg.Admin.LoginWindow)
	assert.Equal(t, 4, cfg.Video.MaxConcurrency)
	assert.Equal(t, 3*time.Second, cfg.Video.PollInterval)
	assert.Equal(t, 45*time.Minute, cfg.Video.TaskTimeout)
	assert.Equal(t, int64(1073741824), cfg.Video.MaxDownloadBytes)
	assert.Equal(t, "doubao-seedance-2-0-260128", cfg.NewAPI.DefaultVideoModel)
	assert.Equal(t, "db-secret", cfg.Database.Password)
}

func TestLoadFromLookupRejectsInvalidNumericAndDurationValues(t *testing.T) {
	tests := []struct {
		name, key, value string
	}{
		{name: "concurrency", key: "VIDEO_MAX_CONCURRENCY", value: "0"},
		{name: "poll interval", key: "VIDEO_POLL_INTERVAL", value: "instant"},
		{name: "task timeout", key: "VIDEO_TASK_TIMEOUT", value: "-1s"},
		{name: "download limit", key: "APP_MAX_DOWNLOAD_BYTES", value: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFromLookup(func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return ""
			})
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.key)
		})
	}
}
