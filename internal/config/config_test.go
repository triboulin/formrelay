package config

import (
	"os"
	"path/filepath"
	"testing"
)

var configKeys = []string{
	"PORT", "DATABASE_URL", "ADMIN_USER", "ADMIN_PASS",
	"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS",
	"FROM_EMAIL", "FROM_NAME",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range configKeys {
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range configKeys {
			os.Unsetenv(k)
		}
	})
}

func withWorkDir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	withWorkDir(t, t.TempDir())

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, attendu 8080", cfg.Port)
	}
	if cfg.DatabaseURL != "./data/formrelay.db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.AdminUser != "admin" {
		t.Errorf("AdminUser = %q", cfg.AdminUser)
	}
	if cfg.AdminPass != "changeme" {
		t.Errorf("AdminPass = %q", cfg.AdminPass)
	}
	if cfg.SMTPPort != 587 {
		t.Errorf("SMTPPort = %d, attendu 587", cfg.SMTPPort)
	}
	if cfg.FromEmail != "noreply@example.com" {
		t.Errorf("FromEmail = %q", cfg.FromEmail)
	}
	if cfg.FromName != "FormRelay Admin" {
		t.Errorf("FromName = %q", cfg.FromName)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	clearEnv(t)
	withWorkDir(t, t.TempDir())

	t.Setenv("PORT", "9999")
	t.Setenv("ADMIN_USER", "root")
	t.Setenv("ADMIN_PASS", "topsecret")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USER", "user@example.com")
	t.Setenv("SMTP_PASS", "pass")
	t.Setenv("FROM_EMAIL", "from@example.com")
	t.Setenv("FROM_NAME", "Custom Name")
	t.Setenv("DATABASE_URL", "/custom/path.db")

	cfg := Load()

	if cfg.Port != "9999" {
		t.Errorf("Port = %q", cfg.Port)
	}
	if cfg.AdminUser != "root" {
		t.Errorf("AdminUser = %q", cfg.AdminUser)
	}
	if cfg.AdminPass != "topsecret" {
		t.Errorf("AdminPass = %q", cfg.AdminPass)
	}
	if cfg.SMTPHost != "smtp.example.com" {
		t.Errorf("SMTPHost = %q", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 2525 {
		t.Errorf("SMTPPort = %d", cfg.SMTPPort)
	}
	if cfg.SMTPUser != "user@example.com" {
		t.Errorf("SMTPUser = %q", cfg.SMTPUser)
	}
	if cfg.SMTPPass != "pass" {
		t.Errorf("SMTPPass = %q", cfg.SMTPPass)
	}
	if cfg.FromEmail != "from@example.com" {
		t.Errorf("FromEmail = %q", cfg.FromEmail)
	}
	if cfg.FromName != "Custom Name" {
		t.Errorf("FromName = %q", cfg.FromName)
	}
	if cfg.DatabaseURL != "/custom/path.db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoad_DotEnvFile(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	content := "PORT=7070\n# ceci est un commentaire\n\nADMIN_USER=\"quoted\"\nADMIN_PASS='single'\nMALFORMED_LINE_NO_EQUALS\n  \nSMTP_HOST = spaced.example.com \n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	withWorkDir(t, dir)

	cfg := Load()

	if cfg.Port != "7070" {
		t.Errorf("Port = %q, attendu 7070", cfg.Port)
	}
	if cfg.AdminUser != "quoted" {
		t.Errorf("AdminUser = %q, attendu quoted (guillemets retirés)", cfg.AdminUser)
	}
	if cfg.AdminPass != "single" {
		t.Errorf("AdminPass = %q, attendu single (guillemets retirés)", cfg.AdminPass)
	}
	if cfg.SMTPHost != "spaced.example.com" {
		t.Errorf("SMTPHost = %q, attendu spaced.example.com (espaces retirés)", cfg.SMTPHost)
	}
}

func TestLoad_EnvTakesPrecedenceOverDotEnv(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("PORT=1111\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	withWorkDir(t, dir)
	t.Setenv("PORT", "2222")

	cfg := Load()

	if cfg.Port != "2222" {
		t.Errorf("Port = %q, la variable d'environnement doit primer sur .env", cfg.Port)
	}
}

func TestLoad_InvalidSMTPPortFallsBackToDefault(t *testing.T) {
	clearEnv(t)
	withWorkDir(t, t.TempDir())
	t.Setenv("SMTP_PORT", "not-a-number")

	cfg := Load()

	if cfg.SMTPPort != 587 {
		t.Errorf("SMTPPort = %d, attendu la valeur par défaut 587 si non numérique", cfg.SMTPPort)
	}
}

func TestLoad_NoDotEnvFilePresent(t *testing.T) {
	clearEnv(t)
	// Répertoire sans .env : loadDotEnv doit échouer silencieusement (os.Open erreur) sans paniquer.
	withWorkDir(t, t.TempDir())

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, attendu la valeur par défaut", cfg.Port)
	}
}
