package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Config regroupe toute la configuration de l'application, lue depuis
// les variables d'environnement (avec chargement optionnel d'un fichier .env).
type Config struct {
	Port        string
	DatabaseURL string

	AdminUser string
	AdminPass string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string

	FromEmail string
	FromName  string
}

// Load charge la configuration depuis les variables d'environnement.
// Un fichier .env à la racine est chargé au préalable s'il existe (sans écraser
// les variables déjà présentes dans l'environnement).
func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "./data/formrelay.db"),

		AdminUser: getEnv("ADMIN_USER", "admin"),
		AdminPass: getEnv("ADMIN_PASS", "changeme"),

		SMTPHost: getEnv("SMTP_HOST", ""),
		SMTPPort: getEnvInt("SMTP_PORT", 587),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),

		FromEmail: getEnv("FROM_EMAIL", "noreply@example.com"),
		FromName:  getEnv("FROM_NAME", "FormRelay Admin"),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// loadDotEnv charge un fichier .env minimal (KEY=VALUE par ligne) dans
// l'environnement du processus, sans écraser les variables déjà définies.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
