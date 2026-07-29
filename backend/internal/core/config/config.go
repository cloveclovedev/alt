// Package config resolves application configuration from defaults, TOML, and
// deployment-time environment overrides.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const defaultConfigPath = "config/alt.toml"

// Config is the fully resolved runtime configuration.
type Config struct {
	Env                        string
	Port                       int
	ShutdownTimeout            time.Duration
	LogLevel                   string
	DatabaseURL                string
	MaxConnections             int32
	UserID                     string
	UserDisplayName            string
	UserTimezone               string
	OpenRouterAPIKey           string
	GoogleOAuthClientID        string
	GoogleOAuthClientSecret    string
	GoogleOAuthRedirectURL     string
	CalendarTokenEncryptionKey string
}

type fileConfig struct {
	Server struct {
		Port            int    `toml:"port"`
		ShutdownTimeout string `toml:"shutdown_timeout"`
	} `toml:"server"`
	Database struct {
		MaxConnections int32 `toml:"max_connections"`
	} `toml:"database"`
	User struct {
		ID          string `toml:"id"`
		DisplayName string `toml:"display_name"`
		Timezone    string `toml:"timezone"`
	} `toml:"user"`
}

// Load resolves defaults -> TOML -> environment. DatabaseURL is required when
// requireDatabase is true.
func Load(requireDatabase bool) (Config, error) {
	cfg := Config{
		Env:             "local",
		Port:            28080,
		ShutdownTimeout: 15 * time.Second,
		LogLevel:        "info",
		MaxConnections:  10,
		UserID:          "00000000-0000-0000-0000-000000000001",
		UserDisplayName: "Local user",
		UserTimezone:    "Asia/Tokyo",
	}

	configPath := getenv("ALT_CONFIG_FILE", defaultConfigPath)
	if err := applyTOML(configPath, configPath != defaultConfigPath, &cfg); err != nil {
		return Config{}, err
	}

	cfg.Env = getenv("APP_ENV", cfg.Env)
	cfg.LogLevel = getenv("LOG_LEVEL", cfg.LogLevel)
	cfg.UserID = getenv("ALT_USER_ID", cfg.UserID)
	cfg.UserDisplayName = getenv("ALT_USER_DISPLAY_NAME", cfg.UserDisplayName)
	cfg.UserTimezone = getenv("ALT_USER_TIMEZONE", cfg.UserTimezone)

	var err error
	if cfg.Port, err = getenvInt("PORT", cfg.Port); err != nil {
		return Config{}, err
	}
	if cfg.MaxConnections, err = getenvInt32("DATABASE_MAX_CONNECTIONS", cfg.MaxConnections); err != nil {
		return Config{}, err
	}
	if raw, ok := os.LookupEnv("SHUTDOWN_TIMEOUT"); ok {
		cfg.ShutdownTimeout, err = time.ParseDuration(strings.TrimSpace(raw))
		if err != nil {
			return Config{}, fmt.Errorf("parse SHUTDOWN_TIMEOUT: %w", err)
		}
	}
	cfg.DatabaseURL, _, err = lookupEnvOrFile("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	cfg.OpenRouterAPIKey, _, err = lookupApplicationSecret("OPENROUTER_API_KEY")
	if err != nil {
		return Config{}, err
	}
	cfg.GoogleOAuthClientSecret, _, err = lookupApplicationSecret("GOOGLE_OAUTH_CLIENT_SECRET")
	if err != nil {
		return Config{}, err
	}
	cfg.CalendarTokenEncryptionKey, _, err = lookupApplicationSecret("CALENDAR_TOKEN_ENCRYPTION_KEY")
	if err != nil {
		return Config{}, err
	}
	cfg.GoogleOAuthClientID = getenv("ALT_GOOGLE_OAUTH_CLIENT_ID", getenv("GOOGLE_OAUTH_CLIENT_ID", ""))
	cfg.GoogleOAuthRedirectURL = getenv("ALT_GOOGLE_OAUTH_REDIRECT_URL", getenv("GOOGLE_OAUTH_REDIRECT_URL", ""))

	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("shutdown timeout must be positive")
	}
	if cfg.MaxConnections < 1 {
		return Config{}, fmt.Errorf("database max_connections must be positive")
	}
	if strings.TrimSpace(cfg.UserID) == "" {
		return Config{}, fmt.Errorf("user id must not be blank")
	}
	if strings.TrimSpace(cfg.UserDisplayName) == "" {
		return Config{}, fmt.Errorf("user display_name must not be blank")
	}
	if _, err := time.LoadLocation(cfg.UserTimezone); err != nil {
		return Config{}, fmt.Errorf("load user timezone %q: %w", cfg.UserTimezone, err)
	}
	if requireDatabase && strings.TrimSpace(cfg.DatabaseURL) == "" {
		return Config{}, fmt.Errorf("DATABASE_URL or DATABASE_URL_FILE is required")
	}
	return cfg, nil
}

func applyTOML(path string, required bool, cfg *Config) error {
	var raw fileConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		if !required && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config file %q: %w", path, err)
	}

	if raw.Server.Port != 0 {
		cfg.Port = raw.Server.Port
	}
	if raw.Server.ShutdownTimeout != "" {
		d, err := time.ParseDuration(raw.Server.ShutdownTimeout)
		if err != nil {
			return fmt.Errorf("parse server.shutdown_timeout: %w", err)
		}
		cfg.ShutdownTimeout = d
	}
	if raw.Database.MaxConnections != 0 {
		cfg.MaxConnections = raw.Database.MaxConnections
	}
	if raw.User.ID != "" {
		cfg.UserID = raw.User.ID
	}
	if raw.User.DisplayName != "" {
		cfg.UserDisplayName = raw.User.DisplayName
	}
	if raw.User.Timezone != "" {
		cfg.UserTimezone = raw.User.Timezone
	}
	return nil
}

func lookupEnvOrFile(key string) (string, bool, error) {
	if path, ok := os.LookupEnv(key + "_FILE"); ok && strings.TrimSpace(path) != "" {
		value, err := os.ReadFile(path)
		if err != nil {
			return "", false, fmt.Errorf("read %s_FILE: %w", key, err)
		}
		return strings.TrimSpace(string(value)), true, nil
	}
	value, ok := os.LookupEnv(key)
	return strings.TrimSpace(value), ok, nil
}

// lookupApplicationSecret prefers product-prefixed values injected by the
// shared development Secrets Manager project, while preserving generic names
// for deployment systems that expose only this application's secrets.
func lookupApplicationSecret(key string) (string, bool, error) {
	if value, found, err := lookupEnvOrFile("ALT_" + key); err != nil || found {
		return value, found, err
	}
	return lookupEnvOrFile(key)
}

func getenv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func getenvInt(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}

func getenvInt32(key string, fallback int32) (int32, error) {
	value, err := getenvInt(key, int(fallback))
	if err != nil {
		return 0, err
	}
	if value > int(^uint32(0)>>1) {
		return 0, fmt.Errorf("%s is too large", key)
	}
	return int32(value), nil
}
