package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment string
	Address     string
	DataFile    string
	WebDir      string
	AssetDir    string
	SecretKey   string
	Admin       Admin
	Database    Database
	NewAPI      NewAPI
	Video       Video
}

type Admin struct {
	Username         string
	Password         string
	SecureCookie     bool
	SessionTTL       time.Duration
	LoginMaxAttempts int
	LoginWindow      time.Duration
}

type Database struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	Params   string
}

type NewAPI struct {
	BaseURL           string
	Token             string
	ScriptModel       string
	FastVideoModel    string
	QualityVideoModel string
	DefaultVideoModel string
}

type Video struct {
	MaxConcurrency   int
	PollInterval     time.Duration
	TaskTimeout      time.Duration
	MaxDownloadBytes int64
}

func Load() (Config, error) {
	return LoadFromLookup(os.Getenv)
}

func LoadFromLookup(lookup func(string) string) (Config, error) {
	value := func(key, fallback string) string {
		if current := strings.TrimSpace(lookup(key)); current != "" {
			return current
		}
		return fallback
	}

	parseBool := func(key string, fallback bool) (bool, error) {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			return fallback, nil
		}
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return false, fmt.Errorf("%s must be a boolean", key)
		}
		return parsed, nil
	}
	parsePositiveInt := func(key string, fallback int) (int, error) {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			return fallback, nil
		}
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}
		return parsed, nil
	}
	parsePositiveInt64 := func(key string, fallback int64) (int64, error) {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			return fallback, nil
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer", key)
		}
		return parsed, nil
	}
	parsePositiveDuration := func(key string, fallback time.Duration) (time.Duration, error) {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			return fallback, nil
		}
		parsed, err := time.ParseDuration(raw)
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("%s must be a positive duration", key)
		}
		return parsed, nil
	}

	secureCookie, err := parseBool("APP_SESSION_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := parsePositiveDuration("APP_SESSION_TTL", 12*time.Hour)
	if err != nil {
		return Config{}, err
	}
	loginMaxAttempts, err := parsePositiveInt("APP_LOGIN_MAX_ATTEMPTS", 10)
	if err != nil {
		return Config{}, err
	}
	loginWindow, err := parsePositiveDuration("APP_LOGIN_WINDOW", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}
	maxConcurrency, err := parsePositiveInt("VIDEO_MAX_CONCURRENCY", 2)
	if err != nil {
		return Config{}, err
	}
	pollInterval, err := parsePositiveDuration("VIDEO_POLL_INTERVAL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	taskTimeout, err := parsePositiveDuration("VIDEO_TASK_TIMEOUT", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	maxDownloadBytes, err := parsePositiveInt64("APP_MAX_DOWNLOAD_BYTES", 1<<30)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Environment: value("APP_ENV", "development"),
		Address:     value("APP_ADDR", ":8080"),
		DataFile:    value("APP_DATA_FILE", "./data/studio.json"),
		WebDir:      value("APP_WEB_DIR", "./web/dist"),
		AssetDir:    value("APP_ASSET_DIR", "./data/assets"),
		SecretKey:   strings.TrimSpace(lookup("APP_SECRET_KEY")),
		Admin: Admin{
			Username:         strings.TrimSpace(lookup("APP_ADMIN_USERNAME")),
			Password:         lookup("APP_ADMIN_PASSWORD"),
			SecureCookie:     secureCookie,
			SessionTTL:       sessionTTL,
			LoginMaxAttempts: loginMaxAttempts,
			LoginWindow:      loginWindow,
		},
		Database: Database{
			Host:     strings.TrimSpace(lookup("DATABASE_HOST")),
			Port:     value("DATABASE_PORT", "3306"),
			Name:     strings.TrimSpace(lookup("DATABASE_NAME")),
			User:     strings.TrimSpace(lookup("DATABASE_USER")),
			Password: lookup("DATABASE_PASSWORD"),
			Params:   value("DATABASE_PARAMS", "charset=utf8mb4&parseTime=true&loc=UTC"),
		},
		NewAPI: NewAPI{
			BaseURL:           strings.TrimRight(strings.TrimSpace(lookup("NEW_API_BASE_URL")), "/"),
			Token:             lookup("NEW_API_TOKEN"),
			ScriptModel:       value("SCRIPT_MODEL", "deepseek-v4-pro"),
			FastVideoModel:    value("VIDEO_FAST_MODEL", "doubao-seedance-2-0-fast-260128"),
			QualityVideoModel: value("VIDEO_QUALITY_MODEL", "doubao-seedance-2-0-260128"),
			DefaultVideoModel: value("VIDEO_DEFAULT_MODEL", "doubao-seedance-2-0-fast-260128"),
		},
		Video: Video{
			MaxConcurrency:   maxConcurrency,
			PollInterval:     pollInterval,
			TaskTimeout:      taskTimeout,
			MaxDownloadBytes: maxDownloadBytes,
		},
	}

	if cfg.Environment == "production" {
		required := []struct {
			key   string
			value string
		}{
			{"APP_SECRET_KEY", cfg.SecretKey},
			{"APP_ADMIN_USERNAME", cfg.Admin.Username},
			{"APP_ADMIN_PASSWORD", cfg.Admin.Password},
			{"DATABASE_HOST", cfg.Database.Host},
			{"DATABASE_NAME", cfg.Database.Name},
			{"DATABASE_USER", cfg.Database.User},
			{"DATABASE_PASSWORD", cfg.Database.Password},
			{"NEW_API_BASE_URL", cfg.NewAPI.BaseURL},
			{"NEW_API_TOKEN", cfg.NewAPI.Token},
		}
		for _, item := range required {
			if strings.TrimSpace(item.value) == "" {
				return Config{}, fmt.Errorf("%s is required in production", item.key)
			}
		}
		if len(cfg.SecretKey) < 32 {
			return Config{}, fmt.Errorf("APP_SECRET_KEY must contain at least 32 characters")
		}
	}

	if cfg.NewAPI.DefaultVideoModel != cfg.NewAPI.FastVideoModel && cfg.NewAPI.DefaultVideoModel != cfg.NewAPI.QualityVideoModel {
		return Config{}, fmt.Errorf("VIDEO_DEFAULT_MODEL must match VIDEO_FAST_MODEL or VIDEO_QUALITY_MODEL")
	}
	return cfg, nil
}
