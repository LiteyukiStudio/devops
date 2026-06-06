package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	unsetEnv(t, "API_ADDR")
	unsetEnv(t, "DATABASE_URL")
	unsetEnv(t, "REDIS_ADDR")

	envFile := filepath.Join(t.TempDir(), ".env.local")
	content := []byte("API_ADDR=:19090\nDATABASE_URL=postgres://user:pass@db:5432/app?sslmode=disable\nREDIS_ADDR=redis:6379\n")
	if err := os.WriteFile(envFile, content, 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("ENV_FILE", envFile)

	cfg := Load()
	if cfg.APIAddr != ":19090" {
		t.Fatalf("APIAddr = %q", cfg.APIAddr)
	}
	if cfg.DatabaseURL != "postgres://user:pass@db:5432/app?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RedisAddr != "redis:6379" {
		t.Fatalf("RedisAddr = %q", cfg.RedisAddr)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	oldValue, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}

	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, oldValue)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func TestEnvOverridesEnvFile(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env.local")
	if err := os.WriteFile(envFile, []byte("API_ADDR=:19090\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("ENV_FILE", envFile)
	t.Setenv("API_ADDR", ":28080")

	cfg := Load()
	if cfg.APIAddr != ":28080" {
		t.Fatalf("APIAddr = %q", cfg.APIAddr)
	}
}
