package config

import "testing"

func clearEnv(t *testing.T) {
	t.Helper()
	keys := []string{"DATABASE_URL", "SESSION_SECRET", "HTTP_PORT", "DB_MAX_CONNS", "APP_ENV", "COOKIE_SECURE", "CORS_ALLOWED_ORIGINS"}
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("SESSION_SECRET", "01234567890123456789012345678901")
	if _, err := Load(); err == nil {
		t.Fatal("esperava erro ao faltar DATABASE_URL")
	}
}

func TestLoad_MissingOrShortSessionSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("SESSION_SECRET", "curto")
	if _, err := Load(); err == nil {
		t.Fatal("esperava erro ao SESSION_SECRET ser muito curto")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("SESSION_SECRET", "01234567890123456789012345678901")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort padrão incorreto: %s", cfg.HTTPPort)
	}
	if cfg.DBMaxConns != 10 {
		t.Errorf("DBMaxConns padrão incorreto: %d", cfg.DBMaxConns)
	}
	if cfg.CookieSecure {
		t.Errorf("CookieSecure deveria ser false por padrão")
	}
}

func TestLoad_InvalidMaxConns(t *testing.T) {
	clearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost/db")
	t.Setenv("SESSION_SECRET", "01234567890123456789012345678901")
	t.Setenv("DB_MAX_CONNS", "not-a-number")
	if _, err := Load(); err == nil {
		t.Fatal("esperava erro ao DB_MAX_CONNS ser inválido")
	}
}

func TestSplitAndTrim(t *testing.T) {
	got := splitAndTrim("http://a.com, http://b.com ,,http://c.com")
	want := []string{"http://a.com", "http://b.com", "http://c.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q want %q", i, got[i], want[i])
		}
	}
}
