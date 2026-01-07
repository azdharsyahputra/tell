package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	CORSAllowedOrigins   []string
	CORSAllowCredentials bool

	JWTSecret string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		HTTPAddr:             getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:          mustGetenv("DATABASE_URL"),
		CORSAllowCredentials: getenv("CORS_ALLOW_CREDENTIALS", "false") == "true",
	}

	origins := strings.Split(getenv("CORS_ALLOWED_ORIGINS", ""), ",")
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
		}
	}

	cfg.JWTSecret = mustGetenv("JWT_SECRET")
	return cfg, nil
}

func getenv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func mustGetenv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		panic("missing env: " + key)
	}
	return v
}
