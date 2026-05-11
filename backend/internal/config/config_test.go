package config

import (
	"slices"
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("GEMINI_API_KEY", "key")
	t.Setenv("APP_PASSWORD", "password")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("ALLOWED_ORIGINS", " http://a.test ,, http://b.test")
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.MetricsAddr != ":9090" || cfg.DBMaxConns != 10 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if want := []string{"http://a.test", "http://b.test"}; !slices.Equal(cfg.AllowedOrigins, want) {
		t.Errorf("AllowedOrigins = %q, want %q", cfg.AllowedOrigins, want)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"missing key", map[string]string{"GEMINI_API_KEY": ""}, "GEMINI_API_KEY"},
		{"bad max conns", map[string]string{"DB_MAX_CONNS": "many"}, "DB_MAX_CONNS"},
		{"zero max conns", map[string]string{"DB_MAX_CONNS": "0"}, "DB_MAX_CONNS"},
		{"blank origins", map[string]string{"ALLOWED_ORIGINS": " , "}, "ALLOWED_ORIGINS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequired(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want it to mention %s", err, tt.wantErr)
			}
		})
	}
}
