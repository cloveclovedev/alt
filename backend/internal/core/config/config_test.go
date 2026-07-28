package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppliesTOMLThenEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "9090")

	path := filepath.Join(t.TempDir(), "alt.toml")
	err := os.WriteFile(path, []byte(`
[server]
port = 8181
shutdown_timeout = "21s"

[database]
max_connections = 7

[user]
id = "11111111-1111-1111-1111-111111111111"
display_name = "Test user"
timezone = "UTC"
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALT_CONFIG_FILE", path)

	cfg, err := Load(true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9090 {
		t.Fatalf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.ShutdownTimeout != 21*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 21s", cfg.ShutdownTimeout)
	}
	if cfg.MaxConnections != 7 {
		t.Fatalf("MaxConnections = %d, want 7", cfg.MaxConnections)
	}
	if cfg.UserTimezone != "UTC" {
		t.Fatalf("UserTimezone = %q, want UTC", cfg.UserTimezone)
	}
}

func TestLoadReadsDatabaseURLFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database-url")
	if err := os.WriteFile(path, []byte("postgres://from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATABASE_URL", "postgres://from-env")
	t.Setenv("DATABASE_URL_FILE", path)

	cfg, err := Load(true)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://from-file" {
		t.Fatalf("DatabaseURL = %q, want file value", cfg.DatabaseURL)
	}
}

func TestLoadRequiresDatabaseForWeb(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(true); err == nil {
		t.Fatal("Load(true) succeeded without a database URL")
	}
	if _, err := Load(false); err != nil {
		t.Fatalf("Load(false) failed: %v", err)
	}
}
